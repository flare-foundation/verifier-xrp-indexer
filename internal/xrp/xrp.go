package xrp

import (
	"context"
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/xrpscan/xrpl-go"
	"gitlab.com/flarenetwork/fdc/verifier-indexer-framework/pkg/indexer"
)

type Config struct {
	WebsocketURL string `toml:"websocket_url"`
}

type Client struct {
	xrp *xrpl.Client
}

func New(cfg *Config) (indexer.BlockchainClient[Block, Transaction], error) {
	if cfg.WebsocketURL == "" {
		return nil, errors.New("websocket_url must be provided")
	}

	client := xrpl.NewClient(xrpl.ClientConfig{URL: cfg.WebsocketURL})

	return Client{xrp: client}, nil
}

type xrpResponse struct {
	Status string
	Result map[string]interface{}
}

type ledgerResult struct {
	LedgerIndex uint64 `mapstructure:"ledger_index"`
	LedgerHash  string `mapstructure:"ledger_hash"`
	Ledger      ledgerInfo
}

type ledgerInfo struct {
	CloseTime    uint64 `mapstructure:"close_time"`
	Transactions []transactionInfo
}

type transactionInfo struct {
	Hash  string                   `mapstructure:"hash"`
	Memos []map[string]interface{} `mapstructure:"Memos"`
}

func (c Client) GetLatestBlockNumber(context.Context) (uint64, error) {
	rsp, err := c.xrp.Request(xrpl.BaseRequest{
		"command":      "ledger",
		"ledger_index": "validated",
		"transactions": false,
		"expand":       false,
		"owner_funds":  false,
	})
	if err != nil {
		return 0, err
	}

	var parsedRsp xrpResponse
	if err := mapstructure.Decode(rsp, &parsedRsp); err != nil {
		return 0, errors.Wrap(err, "mapstructure.Decode(rsp)")
	}

	if parsedRsp.Status != "success" {
		return 0, errors.Errorf("unexpected response status: %v", parsedRsp.Status)
	}

	var result ledgerResult
	if err := mapstructure.Decode(parsedRsp.Result, &result); err != nil {
		return 0, errors.Wrap(err, "mapstructure.Decode(result)")
	}

	return result.LedgerIndex, nil
}

func (c Client) GetBlockResult(
	ctx context.Context, blockNum uint64,
) (*indexer.BlockResult[Block, Transaction], error) {
	rsp, err := c.xrp.Request(xrpl.BaseRequest{
		"command":      "ledger",
		"ledger_index": blockNum,
		"transactions": true,
		"expand":       true,
		"owner_funds":  false,
	})
	if err != nil {
		return nil, err
	}

	var parsedRsp xrpResponse
	if err := mapstructure.Decode(rsp, &parsedRsp); err != nil {
		return nil, errors.Wrap(err, "mapstructure.Decode(rsp)")
	}

	if parsedRsp.Status != "success" {
		return nil, errors.Errorf("unexpected response status: %v", parsedRsp.Status)
	}

	var result ledgerResult
	if err := mapstructure.Decode(parsedRsp.Result, &result); err != nil {
		return nil, errors.Wrap(err, "mapstructure.Decode(result)")
	}

	block := Block{
		Hash:      result.LedgerHash,
		Number:    result.LedgerIndex,
		Timestamp: result.Ledger.CloseTime,
	}

	transactions := make([]Transaction, len(result.Ledger.Transactions))
	for i := range transactions {
		tx := &result.Ledger.Transactions[i]

		memosJSON, err := encodeMemos(tx.Memos)
		if err != nil {
			return nil, errors.Wrap(err, "json.Marshal(tx.Memos)")
		}

		transactions[i] = Transaction{
			Hash:      tx.Hash,
			BlockHash: result.LedgerHash,
			Memos:     memosJSON,
		}
	}

	return &indexer.BlockResult[Block, Transaction]{Block: block, Transactions: transactions}, nil
}

func encodeMemos(memos []map[string]interface{}) ([]json.RawMessage, error) {
	memosJSON := make([]json.RawMessage, len(memos))
	for i, memo := range memos {
		memoJSON, err := json.Marshal(memo)
		if err != nil {
			return nil, err
		}

		memosJSON[i] = memoJSON
	}

	return memosJSON, nil
}
