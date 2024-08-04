package xrp

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/xrpscan/xrpl-go"
	"gitlab.com/ryancollingham/flare-indexer-framework/pkg/database"
	"gitlab.com/ryancollingham/flare-indexer-framework/pkg/indexer"
)

type Config struct {
	WebsocketURL string `toml:"websocket_url"`
}

type Client struct {
	xrp *xrpl.Client
}

func New(cfg *Config) indexer.BlockchainClient {
	client := xrpl.NewClient(xrpl.ClientConfig{URL: cfg.WebsocketURL})

	return Client{xrp: client}
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
	CloseTime    uint64
	Transactions []string
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

func (c Client) GetBlockResult(context.Context, uint64) (*indexer.BlockResult, error) {
	rsp, err := c.xrp.Request(xrpl.BaseRequest{
		"command":      "ledger",
		"ledger_index": "validated",
		"transactions": true,
		"expand":       false,
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

	block := database.Block{
		Hash:      result.LedgerHash,
		Number:    result.LedgerIndex,
		Timestamp: result.Ledger.CloseTime,
	}

	transactions := make([]database.Transaction, len(result.Ledger.Transactions))
	for i := range transactions {
		transactions[i] = database.Transaction{
			Hash:      result.Ledger.Transactions[i],
			BlockHash: result.LedgerHash,
		}
	}

	return &indexer.BlockResult{Block: &block, Transactions: transactions}, nil
}
