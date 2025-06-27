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
	xrpCurrency  = "XRP"
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

	return xrpClient{
		client: http.DefaultClient,
		url:    cfg.Url,
	}, nil
}

type xrpClient struct {
	client *http.Client
	url    string
}

type xrpRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
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
	LedgerIndex uint64    `json:"ledger_index"`
	LedgerHash  string    `json:"ledger_hash"`
	Validated   bool      `json:"validated"`
	Ledger      xrpLedger `json:"ledger"`
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

func (c xrpClient) GetResponse(ctx context.Context, request xrpRequest) ([]byte, error) {
	getReq, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(getReq)
	req, err := http.NewRequest("POST", c.url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
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

func (c xrpClient) GetLedgerResponse(ctx context.Context, params ledgerParams) (*ledgerResponse, error) {
	request := xrpRequest{
		Method: "ledger",
		Params: []interface{}{params},
	}

	resBody, err := c.GetResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	var respStruct ledgerResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return nil, err
	}

	return &respStruct, nil
}

func (c xrpClient) GetLatestBlockInfo(ctx context.Context) (*indexer.BlockInfo, error) {
	respStruct, err := c.GetLedgerResponse(ctx, getLatestParams)
	if err != nil {
		return nil, err
	}

	return &indexer.BlockInfo{
		BlockNumber: respStruct.Result.LedgerIndex,
		Timestamp:   respStruct.Result.Ledger.CloseTime + XRPTimeToUTD,
	}, nil
}

func (c xrpClient) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	getBlockParams := ledgerParams{
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

func (c xrpClient) GetBlockResult(ctx context.Context, blockNum uint64,
) (*indexer.BlockResult[Block, Transaction], error) {
	getBlockParams := ledgerParams{
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
		var tx xrpTransaction
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

func paymentReference(tx xrpTransaction) string {
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

func isNativePayment(tx xrpTransaction) bool {
	var amountStr string
	err := json.Unmarshal(tx.Amount, &amountStr)
	if err == nil {
		_, err = strconv.Atoi(amountStr)
		if err == nil {
			return true
		}
	}
	var amountStruct xrpAmount
	err = json.Unmarshal(tx.Amount, &amountStruct)

	return err == nil && amountStruct.Currency == xrpCurrency
}

func sourceAddressesRoot(tx xrpTransaction) (string, error) {
	var meta xrpMeta

	err := json.Unmarshal(tx.MetaData, &meta)
	if err != nil {
		return "", errors.New("unable to unmarshall source addresses")
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

func (c xrpClient) GetServerInfo(ctx context.Context) (string, error) {
	resBody, err := c.GetResponse(ctx, getServerState)
	if err != nil {
		return "", err
	}
	var respStruct serverInfoResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return "", err
	}

	return respStruct.Result.Info.BuildVersion + "_" + respStruct.Result.Info.ServerState, nil
}
