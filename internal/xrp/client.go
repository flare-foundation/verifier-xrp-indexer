package xrp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
)

// errCodeLedgerNotFound is rippled's error code for a ledger the node does not have.
const errCodeLedgerNotFound = "lgrNotFound"

type xrpClient struct {
	client *http.Client
	url    string
}

// New creates a new XRP blockchain client from the given configuration.
func New(cfg *Config) (indexer.BlockchainClient[*Block, Transaction, struct{}], error) {
	if cfg.Url == "" {
		return nil, errors.New("url must be provided")
	}

	configPath := configFilePath()
	if data, err := os.ReadFile(configPath); err == nil {
		logger.Debugf("Config file (%s):\n%s", configPath, string(data))
	} else {
		logger.Debugf("Could not read config file %s: %v", configPath, err)
	}

	return &xrpClient{
		client: http.DefaultClient,
		url:    cfg.Url,
	}, nil
}

// getResponse sends a JSON-RPC POST request to the node and returns the raw response body.
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

// getLedgerResponse sends a ledger RPC request and returns the parsed, validated response.
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
		logger.Debugf(
			"ledger RPC error detail: status=%s error=%s message=%s",
			respStruct.Result.Status, respStruct.Result.Error, respStruct.Result.ErrorMessage,
		)

		if respStruct.Result.Error == errCodeLedgerNotFound {
			return nil, indexer.ErrBlockNotFound
		}

		return nil, fmt.Errorf("ledger RPC request failed (status: %s, error: %s)", respStruct.Result.Status, respStruct.Result.Error)
	}

	return &respStruct, nil
}

// GetLatestBlockInfo returns the index and close time of the latest validated ledger.
func (c *xrpClient) GetLatestBlockInfo(ctx context.Context) (*indexer.BlockInfo, error) {
	respStruct, err := c.getLedgerResponse(ctx, getLatestParams)
	if err != nil {
		return nil, fmt.Errorf("get latest block info: %w", err)
	}

	return &indexer.BlockInfo{
		BlockNumber: respStruct.Result.LedgerIndex,
		Timestamp:   respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
	}, nil
}

// GetBlockTimestamp returns the Unix close time of the ledger at blockNum.
func (c *xrpClient) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	getBlockParams := ledgerParams{
		LedgerIndex:  strconv.FormatUint(blockNum, 10),
		Transactions: false,
		Expand:       false,
		OwnerFunds:   false,
	}
	respStruct, err := c.getLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return 0, fmt.Errorf("get block timestamp for %d: %w", blockNum, err)
	}

	return respStruct.Result.Ledger.CloseTime + XRPTimeOffset, nil
}

// GetBlockResult returns the block header and all transactions for the ledger at blockNum.
func (c *xrpClient) GetBlockResult(ctx context.Context, blockNum uint64) (*indexer.BlockResult[*Block, Transaction, struct{}], error) {
	getBlockParams := ledgerParams{
		LedgerIndex:  strconv.FormatUint(blockNum, 10),
		Transactions: true,
		Expand:       true,
		OwnerFunds:   false,
	}
	respStruct, err := c.getLedgerResponse(ctx, getBlockParams)
	if err != nil {
		return nil, fmt.Errorf("get block result for %d: %w", blockNum, err)
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
		var tx transaction
		err = json.Unmarshal([]byte(respStruct.Result.Ledger.Transactions[i]), &tx)
		if err != nil {
			return nil, fmt.Errorf("unmarshal transaction %d: %w", i, err)
		}

		transactions[i] = Transaction{
			Hash:                   strings.ToLower(tx.Hash),
			BlockNumber:            respStruct.Result.LedgerIndex,
			Timestamp:              respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
			Response:               string(respStruct.Result.Ledger.Transactions[i]),
			DestinationTag:         tx.DestinationTag,
			SourceAddress:          tx.Account,
			Sequence:               tx.Sequence,
			TicketSequence:         tx.TicketSequence,
			DestinationAddressHash: tx.destinationAddressHash(),
		}

		// case-insensitive check for transaction type
		if strings.EqualFold(tx.TransactionType, paymentType) {
			transactions[i].PaymentReference = tx.paymentReference()
			transactions[i].IsNativePayment, err = tx.isNativePayment()
			if err != nil {
				return nil, fmt.Errorf("isNativePayment for tx %s: %w", tx.Hash, err)
			}

			transactions[i].IntendedReceivingAmount, err = tx.intendedReceivingAmount()
			if err != nil {
				return nil, fmt.Errorf("intended receiving amount for tx %s: %w", tx.Hash, err)
			}
		}

		transactions[i].TransactionStatus, err = tx.transactionStatus()
		if err != nil {
			return nil, fmt.Errorf("transaction status for tx %s: %w", tx.Hash, err)
		}

		transactions[i].SourceAddressesRoot, err = tx.sourceAddressesRoot()
		if err != nil {
			return nil, fmt.Errorf("source addresses root for tx %s: %w", tx.Hash, err)
		}

		transactions[i].FirstMemoDataHash, err = tx.firstMemoDataHash()
		if err != nil {
			return nil, fmt.Errorf("first memo hash for tx %s: %w", tx.Hash, err)
		}
	}

	return &indexer.BlockResult[*Block, Transaction, struct{}]{
		Block:        block,
		Transactions: transactions,
	}, nil
}

// GetServerInfo returns a string of the form "<version>_<state>" identifying the node.
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
