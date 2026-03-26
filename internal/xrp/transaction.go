package xrp

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/merkle"
)

const (
	xrpCurrency = "XRP"
	paymentType = "Payment"
)

type transaction struct {
	Hash            string                         `json:"hash"`
	Memos           []map[string]map[string]string `json:"Memos"`
	TransactionType string                         `json:"TransactionType"`
	Sequence        uint64                         `json:"Sequence"`
	TicketSequence  uint64                         `json:"TicketSequence"`
	Account         string                         `json:"Account"`
	Amount          json.RawMessage                `json:"Amount"`
	MetaData        json.RawMessage                `json:"metaData"`
	DestinationTag  *uint32                        `json:"DestinationTag"`
}

// firstMemoDataHash returns the Keccak256 hash of the first memo's MemoData field,
// or an empty string if no memo or MemoData is present.
func (tx *transaction) firstMemoDataHash() (string, error) {
	if len(tx.Memos) > 0 && tx.Memos[0] != nil {
		if memo, ok := tx.Memos[0]["Memo"]; ok {
			if memoData, ok := memo["MemoData"]; ok && memoData != "" {
				decoded, err := hex.DecodeString(memoData)
				if err != nil {
					return "", err
				}

				hash := crypto.Keccak256Hash(decoded)
				return strings.ToLower(hash.Hex()[2:]), nil
			}
		}
	}

	return "", nil
}

// paymentReference returns the MemoData of the transaction's single memo if it is
// exactly 64 hex characters, or an empty string otherwise.
func (tx *transaction) paymentReference() string {
	if len(tx.Memos) == 1 && tx.Memos[0] != nil {
		if memo, ok := tx.Memos[0]["Memo"]; ok {
			if memoData, ok := memo["MemoData"]; ok {
				if len(memoData) == 64 {
					return strings.ToLower(memoData)
				}
			}
		}
	}

	return ""
}

// isNativePayment reports whether the transaction Amount represents a native XRP drop
// value (a quoted integer string) rather than a token amount (a JSON object).
func (tx *transaction) isNativePayment() (bool, error) {
	if len(tx.Amount) == 0 {
		return false, errors.New("empty Amount field")
	}

	// XRP native amounts are JSON strings of digits ("1000000").
	// Token amounts are JSON objects. Branching on the first byte avoids a redundant unmarshal.
	if tx.Amount[0] == '"' {
		var amountStr string
		if err := json.Unmarshal(tx.Amount, &amountStr); err != nil {
			return false, fmt.Errorf("unmarshal amount string: %w", err)
		}
		_, err := strconv.Atoi(amountStr)
		return err == nil, nil
	}

	var amountObj amount
	if err := json.Unmarshal(tx.Amount, &amountObj); err != nil {
		return false, fmt.Errorf("unmarshal amount object: %w", err)
	}
	return amountObj.Currency == xrpCurrency, nil
}

// sourceAddressesRoot computes the Merkle root of the double-Keccak256-hashed addresses
// whose XRP balance decreased in the transaction. Returns an empty string if no such
// addresses exist.
func (tx *transaction) sourceAddressesRoot() (string, error) {
	var meta meta

	err := json.Unmarshal(tx.MetaData, &meta)
	if err != nil {
		return "", errors.New("unable to unmarshal source addresses")
	}

	sourceAddresses := make([]common.Hash, 0)
	for _, node := range meta.AffectedNodes {
		modifiedNode := node.ModifiedNode
		if modifiedNode.LedgerEntryType != "AccountRoot" || modifiedNode.FinalFields.Account == "" {
			continue
		}

		var balance string
		finalVal := big.NewInt(0)
		var check bool
		if len(modifiedNode.FinalFields.Balance) > 0 {
			err = json.Unmarshal(modifiedNode.FinalFields.Balance, &balance)
			if err != nil {
				return "", fmt.Errorf("unable to unmarshal final balance: %w", err)
			}
			finalVal, check = new(big.Int).SetString(balance, 10)
			if !check {
				return "", errors.New("unable to parse final balance")
			}
		}

		previousVal := big.NewInt(0)
		if len(modifiedNode.PreviousFields.Balance) > 0 {
			err = json.Unmarshal(modifiedNode.PreviousFields.Balance, &balance)
			if err != nil {
				return "", fmt.Errorf("unable to unmarshal previous balance: %w", err)
			}
			previousVal, check = new(big.Int).SetString(balance, 10)
			if !check {
				return "", errors.New("unable to parse previous balance")
			}
		}

		diff := new(big.Int).Sub(finalVal, previousVal)
		if diff.Cmp(big.NewInt(0)) < 0 {
			hashedAddress := crypto.Keccak256Hash(crypto.Keccak256Hash([]byte(modifiedNode.FinalFields.Account)).Bytes())
			sourceAddresses = append(sourceAddresses, hashedAddress)
		}
	}

	if len(sourceAddresses) > 0 {
		merkleTree := merkle.Build(sourceAddresses, false)
		root, err := merkleTree.Root()
		if err != nil {
			return "", fmt.Errorf("build merkle root: %w", err)
		}

		return strings.ToLower(root.Hex()[2:]), nil
	}

	return "", nil
}
