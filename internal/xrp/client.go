package xrp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// endpoint is scheme and host only: hosted providers put the API key in the path
	endpoint string
}

// New creates a new XRP blockchain client from the given configuration.
func New(cfg *Config) (indexer.BlockchainClient[*Block, Transaction, struct{}], error) {
	if cfg.URL == "" {
		return nil, errors.New("url must be provided")
	}

	// url.Parse errors quote the URL, so they are not returned
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, errors.New("url must be absolute, with a scheme and host")
	}

	return &xrpClient{
		client:   newHTTPClient(),
		url:      cfg.URL,
		endpoint: u.Scheme + "://" + u.Host,
	}, nil
}

// newHTTPClient keeps every idle connection to the node: the default two per
// host make concurrent fetches re-handshake on each batch. Timeouts come from
// the per-request context.
func newHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultClient
	}

	transport = transport.Clone()
	transport.MaxIdleConnsPerHost = transport.MaxIdleConns

	return &http.Client{Transport: transport}
}

// getResponse sends a JSON-RPC POST request to the node and returns the raw response body.
func (c *xrpClient) getResponse(ctx context.Context, request xrpRequest) ([]byte, error) {
	getReq, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	buf := bytes.NewBuffer(getReq)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		// *url.Error prints the full URL; its cause names at most host and port
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}

		return nil, fmt.Errorf("send request to %s: %w", c.endpoint, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debugf("error closing response body: %v", err)
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

		// permanent only for a numeric index; for "validated" it means the node
		// has no validated ledger yet and must be retried
		if respStruct.Result.Error == errCodeLedgerNotFound && params.byNumber() {
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
		APIVersion:   apiVersion,
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
		APIVersion:   apiVersion,
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
		return nil, fmt.Errorf("ledger %d not validated yet", blockNum)
	}

	block := &Block{
		Hash:         strings.ToLower(respStruct.Result.LedgerHash),
		BlockNumber:  respStruct.Result.LedgerIndex,
		Timestamp:    respStruct.Result.Ledger.CloseTime + XRPTimeOffset,
		Transactions: uint64(len(respStruct.Result.Ledger.Transactions)),
	}

	transactions := make([]Transaction, len(respStruct.Result.Ledger.Transactions))

	for i := range transactions {
		var tx transaction
		err = json.Unmarshal([]byte(respStruct.Result.Ledger.Transactions[i]), &tx)
		if err != nil {
			return nil, fmt.Errorf("unmarshal transaction %d in ledger %d: %w: %w", i, blockNum, err, indexer.ErrInvalidData)
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
				return nil, fmt.Errorf("isNativePayment for tx %s in ledger %d: %w: %w", tx.Hash, blockNum, err, indexer.ErrInvalidData)
			}

			if transactions[i].IsNativePayment {
				transactions[i].IntendedReceivingAmount, err = tx.intendedReceivingAmount()
				if err != nil {
					return nil, fmt.Errorf("intended receiving amount for tx %s in ledger %d: %w: %w", tx.Hash, blockNum, err, indexer.ErrInvalidData)
				}
			} else {
				// zero keeps non-native payments out of the nonexistence amount
				// filter: a sender can force one to fail with a receiver-fault
				// code, and a failed payment counts with its stated Amount
				transactions[i].IntendedReceivingAmount = new(uint64)
			}
		}

		transactions[i].TransactionStatus, err = tx.transactionStatus()
		if err != nil {
			return nil, fmt.Errorf("transaction status for tx %s in ledger %d: %w: %w", tx.Hash, blockNum, err, indexer.ErrInvalidData)
		}

		transactions[i].SourceAddressesRoot, err = tx.sourceAddressesRoot()
		if err != nil {
			return nil, fmt.Errorf("source addresses root for tx %s in ledger %d: %w: %w", tx.Hash, blockNum, err, indexer.ErrInvalidData)
		}

		transactions[i].FirstMemoDataHash, err = tx.firstMemoDataHash()
		if err != nil {
			return nil, fmt.Errorf("first memo hash for tx %s in ledger %d: %w: %w", tx.Hash, blockNum, err, indexer.ErrInvalidData)
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
