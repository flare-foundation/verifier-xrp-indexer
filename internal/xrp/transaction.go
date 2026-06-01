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
	SendMax         json.RawMessage                `json:"SendMax"`
	MetaData        json.RawMessage                `json:"metaData"`
	DomainID        *string                        `json:"DomainID"`
	Destination     string                         `json:"Destination"`
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

// isNativePayment reports whether the transaction is native XRP transfer.
func (tx *transaction) isNativePayment() (bool, error) {
	if len(tx.Amount) == 0 {
		return false, errors.New("empty Amount field")
	}

	if tx.SendMax != nil {
		return false, nil
	}

	// XRP native amounts are JSON strings of digits ("1000000").
	// Token amounts are JSON objects. Branching on the first byte avoids a redundant unmarshal.
	if tx.Amount[0] == '"' {
		var amountStr string
		if err := json.Unmarshal(tx.Amount, &amountStr); err != nil {
			return false, fmt.Errorf("unmarshal amount string: %w", err)
		}

		_, err := strconv.ParseUint(amountStr, 10, 64)
		return err == nil, nil
	}

	var amountObj amount
	if err := json.Unmarshal(tx.Amount, &amountObj); err != nil {
		return false, fmt.Errorf("unmarshal amount object: %w", err)
	}
	return amountObj.Currency == xrpCurrency, nil
}

// destinationAddressHash returns the Keccak256 hash of the transaction's
// Destination address, or nil if no destination is set.
func (tx *transaction) destinationAddressHash() *string {
	if len(tx.Destination) > 0 {
		hash := strings.ToLower(crypto.Keccak256Hash([]byte(tx.Destination)).Hex()[2:])
		return &hash
	}

	return nil
}

// transactionStatus returns true when the transaction result is tesSUCCESS or a
// destination-related failure (tecDST_TAG_NEEDED, tecNO_DST, tecNO_DST_INSUF_XRP, and
// tecNO_PERMISSION without set DomainID), and false for all other results.
func (tx *transaction) transactionStatus() (bool, error) {
	var meta struct {
		TransactionResult string `json:"TransactionResult"`
	}

	err := json.Unmarshal(tx.MetaData, &meta)
	if err != nil {
		return false, fmt.Errorf("unmarshal meta data: %w", err)
	}

	switch meta.TransactionResult {
	case "tesSUCCESS":
		return true, nil
	case "tecDST_TAG_NEEDED", "tecNO_DST", "tecNO_DST_INSUF_XRP":
		return true, nil
	case "tecNO_PERMISSION":
		if tx.DomainID != nil {
			return false, nil
		}
		return true, nil

	default:
		return false, nil
	}
}

// intendedReceivingAmount returns the native XRP amount in drops that the
// sender intended to deliver. Returns nil for non-native (token) payments.
func (tx *transaction) intendedReceivingAmount() (*uint64, error) {
	if len(tx.Amount) == 0 || tx.Amount[0] != '"' {
		return nil, nil
	}

	var s string
	if err := json.Unmarshal(tx.Amount, &s); err != nil {
		return nil, fmt.Errorf("unmarshal amount: %w", err)
	}

	drops, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse drops: %w", err)
	}

	return &drops, nil
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

	sourceAddresses := make([]common.Hash, 0, 1)
	for _, node := range meta.AffectedNodes {
		// ModifiedNode and DeletedNode share the same structure and can both
		// represent source addresses with decreased balances. CreatedNode entries
		// only have increased balances (from zero) and are not source addresses.
		var change nodeChange
		switch {
		case node.ModifiedNode.LedgerEntryType != "":
			change = node.ModifiedNode
		case node.DeletedNode.LedgerEntryType != "":
			change = node.DeletedNode
		default:
			continue
		}

		if change.LedgerEntryType != "AccountRoot" || change.FinalFields.Account == "" {
			continue
		}

		var balance string
		finalVal := big.NewInt(0)
		var check bool
		if len(change.FinalFields.Balance) > 0 {
			err = json.Unmarshal(change.FinalFields.Balance, &balance)
			if err != nil {
				return "", fmt.Errorf("unable to unmarshal final balance: %w", err)
			}
			finalVal, check = new(big.Int).SetString(balance, 10)
			if !check {
				return "", errors.New("unable to parse final balance")
			}
		}

		previousVal := big.NewInt(0)
		if len(change.PreviousFields.Balance) > 0 {
			err = json.Unmarshal(change.PreviousFields.Balance, &balance)
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
			hashedAddress := crypto.Keccak256Hash(crypto.Keccak256Hash([]byte(change.FinalFields.Account)).Bytes())
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
