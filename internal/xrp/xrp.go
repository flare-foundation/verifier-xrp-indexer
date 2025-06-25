package xrp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/merkle"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/pkg/errors"
)

const (
	XRPCurrency  = "XRP"
	XRPTimeToUTD = uint64(946684800)
	paymentType  = "Payment"
)

type Config struct {
	Url string `toml:"url"`
}

func New(cfg *Config) (indexer.BlockchainClient[Block, Transaction], error) {
	if cfg.Url == "" {
		return nil, errors.New("url must be provided")
	}

	xrpClient := XRPClient{
		Client: http.DefaultClient,
		Url:    cfg.Url}

	return xrpClient, nil
}

type XRPClient struct {
	Client  *http.Client
	Url     string
	Headers http.Header
}

type XRPRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type LedgerParams struct {
	LedgerIndex  string `json:"ledger_index"`
	Transactions bool   `json:"transactions"`
	Expand       bool   `json:"expand"`
	OwnerFunds   bool   `json:"owner_funds"`
}

type LedgerResponse struct {
	Result LedgerResult `json:"result"`
}

type LedgerResult struct {
	LedgerIndex uint64    `json:"ledger_index"`
	LedgerHash  string    `json:"ledger_hash"`
	Validated   bool      `json:"validated"`
	Ledger      XRPLedger `json:"ledger"`
}

type XRPLedger struct {
	CloseTime    uint64 `json:"close_time"`
	Transactions []json.RawMessage
}

type XRPTransaction struct {
	Hash            string                         `json:"hash"`
	Memos           []map[string]map[string]string `json:"Memos"`
	TransactionType string                         `json:"TransactionType"`
	Amount          json.RawMessage                `json:"Amount"`
	MetaData        json.RawMessage                `json:"metaData"`
}

type XRPAmount struct {
	Currency string `json:"currency"`
}

type XRPMeta struct {
	AffectedNodes []XRPAffectedNodes `json:"AffectedNodes"`
}

type XRPAffectedNodes struct {
	XRPModifiedNode XRPModifiedNode `json:"ModifiedNode"`
}

type XRPModifiedNode struct {
	FinalFields     XRPFields `json:"FinalFields"`
	PreviousFields  XRPFields `json:"PreviousFields"`
	LedgerEntryType string    `json:"LedgerEntryType"`
}

type XRPFields struct {
	Account string          `json:"Account"`
	Balance json.RawMessage `json:"Balance"`
}

type ServerInfoRequest struct {
	Method string `json:"method"`
}

type ServerInfoResponse struct {
	Result ServerInfoResult `json:"result"`
}

type ServerInfoResult struct {
	Info ServerInfo `json:"info"`
}

type ServerInfo struct {
	BuildVersion string `json:"build_version"`
	ServerState  string `json:"server_state"`
}

var getLatestParams LedgerParams
var getServerState XRPRequest

func init() {
	getLatestParams = LedgerParams{
		LedgerIndex:  "validated",
		Transactions: false,
		Expand:       false,
		OwnerFunds:   false,
	}
	getServerState = XRPRequest{
		Method: "server_info",
	}
}

func (c XRPClient) GetResponse(ctx context.Context, request XRPRequest) ([]byte, error) {
	getReq, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(getReq)
	req, err := http.NewRequest("POST", c.Url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req = req.WithContext(ctx)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("error response status")
	}

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

func (c XRPClient) GetLedgerResponse(ctx context.Context, params LedgerParams) (*LedgerResponse, error) {
	request := XRPRequest{
		Method: "ledger",
		Params: []interface{}{params},
	}

	resBody, err := c.GetResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	var respStruct LedgerResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return nil, err
	}

	return &respStruct, nil
}

func (c XRPClient) GetLatestBlockInfo(ctx context.Context) (*indexer.BlockInfo, error) {
	respStruct, err := c.GetLedgerResponse(ctx, getLatestParams)
	if err != nil {
		return nil, err
	}

	return &indexer.BlockInfo{
		BlockNumber: respStruct.Result.LedgerIndex,
		Timestamp:   respStruct.Result.Ledger.CloseTime + XRPTimeToUTD,
	}, nil
}

func (c XRPClient) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	getBlockParams := LedgerParams{
		LedgerIndex:  strconv.Itoa(int(blockNum)),
		Transactions: false,
		Expand:       false,
		OwnerFunds:   false,
	}
	respStruct, err := c.GetLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return 0, err
	}

	return respStruct.Result.Ledger.CloseTime + XRPTimeToUTD, nil
}

func (c XRPClient) GetBlockResult(ctx context.Context, blockNum uint64,
) (*indexer.BlockResult[Block, Transaction], error) {
	getBlockParams := LedgerParams{
		LedgerIndex:  strconv.Itoa(int(blockNum)),
		Transactions: true,
		Expand:       true,
		OwnerFunds:   false,
	}
	respStruct, err := c.GetLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return nil, err
	}
	if !respStruct.Result.Validated {
		return nil, errors.New("error block not validated")
	}

	block := Block{
		Hash:         strings.ToLower(respStruct.Result.LedgerHash),
		BlockNumber:  respStruct.Result.LedgerIndex,
		Timestamp:    respStruct.Result.Ledger.CloseTime + XRPTimeToUTD,
		Transactions: uint64(len(respStruct.Result.Ledger.Transactions)),
	}

	transactions := make([]Transaction, len(respStruct.Result.Ledger.Transactions))
	for i := range transactions {
		var tx XRPTransaction
		err = json.Unmarshal([]byte(respStruct.Result.Ledger.Transactions[i]), &tx)
		if err != nil {
			return nil, err
		}

		transactions[i] = Transaction{
			Hash:        strings.ToLower(tx.Hash),
			BlockNumber: respStruct.Result.LedgerIndex,
			Timestamp:   respStruct.Result.Ledger.CloseTime + XRPTimeToUTD,
			Response:    string(respStruct.Result.Ledger.Transactions[i]),
		}

		// case-insensitive check for transaction type
		if strings.EqualFold(tx.TransactionType, paymentType) {
			transactions[i].PaymentReference = paymentReference(tx)
			transactions[i].IsNativePayment = isNativePayment(tx)
		}

		transactions[i].SourceAddressesRoot, err = sourceAddressesRoot(tx)
		if err != nil {
			return nil, err
		}
	}

	return &indexer.BlockResult[Block, Transaction]{Block: block, Transactions: transactions}, nil
}

func paymentReference(tx XRPTransaction) string {
	if len(tx.Memos) == 1 {
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

func isNativePayment(tx XRPTransaction) bool {
	var amountStr string
	err := json.Unmarshal(tx.Amount, &amountStr)
	if err == nil {
		_, err = strconv.Atoi(amountStr)
		if err == nil {
			return true
		}
	}
	var amountStruct XRPAmount
	err = json.Unmarshal(tx.Amount, &amountStruct)

	return err == nil && amountStruct.Currency == XRPCurrency
}

func sourceAddressesRoot(tx XRPTransaction) (string, error) {
	var meta XRPMeta

	err := json.Unmarshal(tx.MetaData, &meta)
	if err != nil {
		return "", errors.New("unable to unmarshall source addresses")
	}

	sourceAddresses := make([]common.Hash, 0)
	for _, node := range meta.AffectedNodes {
		modifiedNode := node.XRPModifiedNode
		if modifiedNode.LedgerEntryType != "AccountRoot" || modifiedNode.FinalFields.Account == "" {
			continue
		}

		var balance string
		finalVal := big.NewInt(0)
		var check bool
		if len(modifiedNode.FinalFields.Balance) > 0 {
			err = json.Unmarshal(modifiedNode.FinalFields.Balance, &balance)
			if err != nil {
				return "", errors.Wrap(err, "unable to unmarshall final balance")
			}
			finalVal, check = new(big.Int).SetString(balance, 10)
			if !check {
				return "", errors.New("unable to parse balance")
			}
		}

		previousVal := big.NewInt(0)
		if len(modifiedNode.PreviousFields.Balance) > 0 {
			err = json.Unmarshal(modifiedNode.PreviousFields.Balance, &balance)
			if err != nil {
				return "", errors.Wrap(err, "unable to unmarshall previous balance")
			}
			previousVal, check = new(big.Int).SetString(balance, 10)
			if !check {
				return "", errors.New("unable to parse balance")
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
			return "", err
		}

		return strings.ToLower(root.Hex()[2:]), nil
	}

	return "", nil
}

func (c XRPClient) GetServerInfo(ctx context.Context) (string, error) {
	resBody, err := c.GetResponse(ctx, getServerState)
	if err != nil {
		return "", err
	}
	var respStruct ServerInfoResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return "", err
	}

	return respStruct.Result.Info.BuildVersion + "_" + respStruct.Result.Info.ServerState, nil
}
