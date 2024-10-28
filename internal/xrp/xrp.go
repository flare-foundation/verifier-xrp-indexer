package xrp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/pkg/errors"
)

const XRPCurrency = "XRP"

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

type LedgerRequest struct {
	Method string       `json:"method"`
	Params []XRPParamas `json:"params"`
}

type XRPParamas struct {
	LedgerIndex  string `json:"ledger_index"`
	Transactions bool   `json:"transactions"`
	Expand       bool   `json:"expand"`
	OwnerFunds   bool   `json:"owner_funds"`
}

type LedgerResponse struct {
	Result XRPResult `json:"result"`
}

type XRPResult struct {
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
}

type XRPAmount struct {
	Currency string `json:"currency"`
}

var getLatestStruct LedgerRequest

func init() {
	getLatestStruct = LedgerRequest{
		Method: "ledger",
		Params: []XRPParamas{{
			LedgerIndex:  "validated",
			Transactions: false,
			Expand:       false,
			OwnerFunds:   false,
		}},
	}
}

func (c XRPClient) GetLedgerResponse(ctx context.Context, request LedgerRequest) (*LedgerResponse, error) {
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

	var respStruct LedgerResponse
	err = json.Unmarshal(resBody, &respStruct)
	if err != nil {
		return nil, err
	}

	return &respStruct, nil
}

func (c XRPClient) GetLatestBlockInfo(ctx context.Context) (*indexer.BlockInfo, error) {
	respStruct, err := c.GetLedgerResponse(ctx, getLatestStruct)
	if err != nil {
		return nil, err
	}

	return &indexer.BlockInfo{
		BlockNumber: respStruct.Result.LedgerIndex,
		Timestamp:   respStruct.Result.Ledger.CloseTime,
	}, nil
}

func (c XRPClient) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	getBlockStruct := LedgerRequest{
		Method: "ledger",
		Params: []XRPParamas{{
			LedgerIndex:  strconv.Itoa(int(blockNum)),
			Transactions: false,
			Expand:       false,
			OwnerFunds:   false,
		}},
	}
	respStruct, err := c.GetLedgerResponse(ctx, getBlockStruct)
	if err != nil {
		return 0, err
	}

	return respStruct.Result.Ledger.CloseTime, nil
}

func (c XRPClient) GetBlockResult(ctx context.Context, blockNum uint64,
) (*indexer.BlockResult[Block, Transaction], error) {
	getBlockStruct := LedgerRequest{
		Method: "ledger",
		Params: []XRPParamas{{
			LedgerIndex:  strconv.Itoa(int(blockNum)),
			Transactions: true,
			Expand:       true,
			OwnerFunds:   false,
		}},
	}
	respStruct, err := c.GetLedgerResponse(ctx, getBlockStruct)
	if err != nil {
		return nil, err
	}
	if !respStruct.Result.Validated {
		return nil, errors.New("error block not validated")
	}

	block := Block{
		Hash:         respStruct.Result.LedgerHash,
		BlockNumber:  respStruct.Result.LedgerIndex,
		Timestamp:    respStruct.Result.Ledger.CloseTime,
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
			Hash:        tx.Hash,
			BlockNumber: respStruct.Result.LedgerIndex,
			Timestamp:   respStruct.Result.Ledger.CloseTime,
			Response:    string(respStruct.Result.Ledger.Transactions[i]),
		}

		transactions[i].PaymentReference = paymentReference(tx)
		transactions[i].IsNativePayment = isNativePayment(tx)
	}

	return &indexer.BlockResult[Block, Transaction]{Block: block, Transactions: transactions}, nil
}

func paymentReference(tx XRPTransaction) string {
	if len(tx.Memos) == 1 {
		if memo, ok := tx.Memos[0]["Memo"]; ok {
			if memoData, ok := memo["MemoData"]; ok {
				if len(memoData) == 64 {
					return memoData
				}
			}
		}
	}

	return ""
}

func isNativePayment(tx XRPTransaction) bool {
	if tx.TransactionType == "Payment" {
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
		if err == nil && amountStruct.Currency == XRPCurrency {
			return true
		}
	}

	return false
}
