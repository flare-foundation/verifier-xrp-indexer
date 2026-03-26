package xrp

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/merkle"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
)

const (
	xrpCurrency = "XRP"
	// XRPTimeOffset is the seconds between the XRP epoch (Jan 1 2000) and the Unix epoch (Jan 1 1970).
	XRPTimeOffset = uint64(946684800)
	paymentType   = "Payment"
)

type Config struct {
	Url string `toml:"url"`
}

type xrpClient struct {
	client *http.Client
	url    string
}

func New(cfg *Config) (indexer.BlockchainClient[*Block, Transaction, struct{}], error) {
	if cfg.Url == "" {
		return nil, errors.New("url must be provided")
	}

	return &xrpClient{
		client: http.DefaultClient,
		url:    cfg.Url,
	}, nil
}

func (c *Config) ApplyEnvOverrides() error {
	if url, ok := os.LookupEnv("XRP_URL"); ok {
		c.Url = url
	}
	return nil
}

type xrpRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type ledgerParams struct {
	LedgerIndex  string `json:"ledger_index"`
	Transactions bool   `json:"transactions"`
	Expand       bool   `json:"expand"`
	OwnerFunds   bool   `json:"owner_funds"`
}

type ledgerResponse struct {
	Result ledgerResult `json:"result"`
}

type ledgerResult struct {
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	LedgerIndex  uint64    `json:"ledger_index"`
	LedgerHash   string    `json:"ledger_hash"`
	Validated    bool      `json:"validated"`
	Ledger       xrpLedger `json:"ledger"`
}

type xrpLedger struct {
	CloseTime    uint64 `json:"close_time"`
	Transactions []json.RawMessage
}

type xrpTransaction struct {
	Hash            string                         `json:"hash"`
	Memos           []map[string]map[string]string `json:"Memos"`
	TransactionType string                         `json:"TransactionType"`
	Amount          json.RawMessage                `json:"Amount"`
	MetaData        json.RawMessage                `json:"metaData"`
	Sequence        uint64                         `json:"Sequence"`
	TicketSequence  uint64                         `json:"TicketSequence"`
	Account         string                         `json:"Account"`
	DestinationTag  *uint32                        `json:"DestinationTag"`
}

type xrpAmount struct {
	Currency string `json:"currency"`
}

type xrpMeta struct {
	AffectedNodes []xrpAffectedNodes `json:"AffectedNodes"`
}

type xrpAffectedNodes struct {
	ModifiedNode xrpModifiedNode `json:"ModifiedNode"`
}

type xrpModifiedNode struct {
	FinalFields     xrpFields `json:"FinalFields"`
	PreviousFields  xrpFields `json:"PreviousFields"`
	LedgerEntryType string    `json:"LedgerEntryType"`
}

type xrpFields struct {
	Account string          `json:"Account"`
	Balance json.RawMessage `json:"Balance"`
}

type serverInfoResponse struct {
	Result serverInfoResult `json:"result"`
}

type serverInfoResult struct {
	Info serverInfo `json:"info"`
}

type serverInfo struct {
	BuildVersion string `json:"build_version"`
	ServerState  string `json:"server_state"`

	// Some nodes return "rippled_version" instead of "build_version"
	RippledVersion string `json:"rippled_version"`
}

var getLatestParams = ledgerParams{
	LedgerIndex:  "validated",
	Transactions: false,
	Expand:       false,
	OwnerFunds:   false,
}

var getServerState = xrpRequest{
	Method: "server_info",
}

func (c *xrpClient) getResponse(ctx context.Context, request xrpRequest) ([]byte, error) {
	getReq, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	buf := bytes.NewBuffer(getReq)
	req, err := http.NewRequest(http.MethodPost, c.url, buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return resBody, nil
}

func (c *xrpClient) getLedgerResponse(ctx context.Context, params ledgerParams) (*ledgerResponse, error) {
	request := xrpRequest{
		Method: "ledger",
		Params: []any{params},
	}

	resBody, err := c.getResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("ledger request: %w", err)
	}

	var respStruct ledgerResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ledger response: %w", err)
	}

	if respStruct.Result.Status != "success" {
		logger.Debugf("ledger RPC error detail: status=%s message=%s", respStruct.Result.Status, respStruct.Result.ErrorMessage)
		return nil, fmt.Errorf("ledger RPC request failed (status: %s)", respStruct.Result.Status)
	}

	return &respStruct, nil
}

func (c *xrpClient) GetLatestBlockInfo(ctx context.Context) (*indexer.BlockInfo, error) {
	respStruct, err := c.getLedgerResponse(ctx, getLatestParams)
	if err != nil {
		return nil, err
	}

	return &indexer.BlockInfo{
		BlockNumber: respStruct.Result.LedgerIndex,
		Timestamp:   respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
	}, nil
}

func (c *xrpClient) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	getBlockParams := ledgerParams{
		LedgerIndex:  strconv.Itoa(int(blockNum)),
		Transactions: false,
		Expand:       false,
		OwnerFunds:   false,
	}
	respStruct, err := c.getLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return 0, err
	}

	return respStruct.Result.Ledger.CloseTime + XRPTimeOffset, nil
}

func (c *xrpClient) GetBlockResult(ctx context.Context, blockNum uint64) (*indexer.BlockResult[*Block, Transaction, struct{}], error) {
	getBlockParams := ledgerParams{
		LedgerIndex:  strconv.Itoa(int(blockNum)),
		Transactions: true,
		Expand:       true,
		OwnerFunds:   false,
	}
	respStruct, err := c.getLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return nil, err
	}

	if !respStruct.Result.Validated {
		return nil, fmt.Errorf("block not validated in response: %+v", respStruct)
	}

	block := &Block{
		Hash:         strings.ToLower(respStruct.Result.LedgerHash),
		BlockNumber:  respStruct.Result.LedgerIndex,
		Timestamp:    respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
		Transactions: uint64(len(respStruct.Result.Ledger.Transactions)),
	}

	transactions := make([]Transaction, len(respStruct.Result.Ledger.Transactions))

	logger.Debugf("Processing ledger %d with %d transactions", blockNum, len(respStruct.Result.Ledger.Transactions))

	for i := range transactions {
		var tx xrpTransaction
		err = json.Unmarshal([]byte(respStruct.Result.Ledger.Transactions[i]), &tx)
		if err != nil {
			return nil, fmt.Errorf("unmarshal transaction %d: %w", i, err)
		}

		transactions[i] = Transaction{
			Hash:              strings.ToLower(tx.Hash),
			BlockNumber:       respStruct.Result.LedgerIndex,
			Timestamp:         respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
			Response:          string(respStruct.Result.Ledger.Transactions[i]),
			FirstMemoDataHash: firstMemoDataHash(tx),
			DestinationTag:    tx.DestinationTag,
			SourceAddress:     tx.Account,
			Sequence:          tx.Sequence,
			TicketSequence:    tx.TicketSequence,
		}

		// case-insensitive check for transaction type
		if strings.EqualFold(tx.TransactionType, paymentType) {
			transactions[i].PaymentReference = paymentReference(tx)
			isNative, err := isNativePayment(tx)
			if err != nil {
				return nil, fmt.Errorf("isNativePayment: %w", err)
			}
			transactions[i].IsNativePayment = isNative
		}

		transactions[i].SourceAddressesRoot, err = sourceAddressesRoot(tx)
		transactions[i].SourceAddress = tx.Account
		transactions[i].Sequence = tx.Sequence
		transactions[i].TicketSequence = tx.TicketSequence
		if err != nil {
			return nil, fmt.Errorf("source addresses root for tx %s: %w", tx.Hash, err)
		}
	}

	return &indexer.BlockResult[*Block, Transaction, struct{}]{Block: block, Transactions: transactions}, nil
}

func firstMemoDataHash(tx xrpTransaction) string {
	if len(tx.Memos) > 0 && tx.Memos[0] != nil {
		if memo, ok := tx.Memos[0]["Memo"]; ok {
			if memoData, ok := memo["MemoData"]; ok && memoData != "" {
				decoded, err := hex.DecodeString(memoData)
				if err != nil {
					return ""
				}
				hash := crypto.Keccak256Hash(decoded)
				return strings.ToLower(hash.Hex()[2:])
			}
		}
	}

	return ""
}

func paymentReference(tx xrpTransaction) string {
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

func isNativePayment(tx xrpTransaction) (bool, error) {
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

	var amountObj xrpAmount
	if err := json.Unmarshal(tx.Amount, &amountObj); err != nil {
		return false, fmt.Errorf("unmarshal amount object: %w", err)
	}
	return amountObj.Currency == xrpCurrency, nil
}

func sourceAddressesRoot(tx xrpTransaction) (string, error) {
	var meta xrpMeta

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

func (c *xrpClient) GetServerInfo(ctx context.Context) (string, error) {
	resBody, err := c.getResponse(ctx, getServerState)
	if err != nil {
		return "", fmt.Errorf("server info request: %w", err)
	}

	var respStruct serverInfoResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return "", fmt.Errorf("unmarshal server info: %w", err)
	}

	info := respStruct.Result.Info
	if info.BuildVersion == "" && info.RippledVersion == "" {
		return "", errors.New("unable to get server version")
	}

	if info.BuildVersion == "" {
		info.BuildVersion = info.RippledVersion
	}

	if info.ServerState == "" {
		info.ServerState = "unknown"
	}

	return fmt.Sprintf("%s_%s", info.BuildVersion, info.ServerState), nil
}
