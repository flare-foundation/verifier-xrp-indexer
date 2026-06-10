package xrp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRPCStub returns a client backed by a test server that always responds
// with the given body and records the received request bodies.
func newRPCStub(t *testing.T, body string) (indexer.BlockchainClient[*Block, Transaction, struct{}], *[][]byte) {
	t.Helper()

	var requests [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		requests = append(requests, data)

		_, err = w.Write([]byte(body))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	client, err := New(&Config{Url: server.URL})
	require.NoError(t, err)

	return client, &requests
}

// requireAPIVersionPinned asserts that every recorded request carries a single
// params object with api_version 1.
func requireAPIVersionPinned(t *testing.T, requests [][]byte) {
	t.Helper()

	require.NotEmpty(t, requests)
	for _, req := range requests {
		var rpcReq struct {
			Params []struct {
				APIVersion int `json:"api_version"`
			} `json:"params"`
		}

		require.NoError(t, json.Unmarshal(req, &rpcReq))
		require.Len(t, rpcReq.Params, 1)
		require.Equal(t, 1, rpcReq.Params[0].APIVersion)
	}
}

func TestLedgerRequestsPinAPIVersion(t *testing.T) {
	client, requests := newRPCStub(t, `{"result":{"status":"success","validated":true,"ledger_index":42,"ledger_hash":"ABC","ledger":{"close_time":100,"transactions":[]}}}`)

	_, err := client.GetLatestBlockInfo(t.Context())
	require.NoError(t, err)

	_, err = client.GetBlockTimestamp(t.Context(), 42)
	require.NoError(t, err)

	_, err = client.GetBlockResult(t.Context(), 42)
	require.NoError(t, err)

	requireAPIVersionPinned(t, *requests)
	require.Len(t, *requests, 3)
}

func TestServerInfoRequestPinsAPIVersion(t *testing.T) {
	client, requests := newRPCStub(t, `{"result":{"status":"success","info":{"build_version":"2.5.0","server_state":"full"}}}`)

	info, err := client.GetServerInfo(t.Context())
	require.NoError(t, err)
	require.Equal(t, "2.5.0_full", info)

	requireAPIVersionPinned(t, *requests)
}

func TestLedgerNotFoundMapsToErrBlockNotFound(t *testing.T) {
	client, _ := newRPCStub(t, `{"result":{"error":"lgrNotFound","error_code":21,"error_message":"ledgerNotFound","request":{"command":"ledger","ledger_index":"42"},"status":"error"}}`)

	_, err := client.GetBlockTimestamp(t.Context(), 42)
	require.ErrorIs(t, err, indexer.ErrBlockNotFound)

	_, err = client.GetBlockResult(t.Context(), 42)
	require.ErrorIs(t, err, indexer.ErrBlockNotFound)
}

func TestIntendedReceivingAmountMapping(t *testing.T) {
	meta := `"metaData":{"TransactionResult":"tesSUCCESS","AffectedNodes":[]}`
	body := `{"result":{"status":"success","validated":true,"ledger_index":100,"ledger_hash":"AB","ledger":{"close_time":500,"transactions":[
		{"hash":"A1","TransactionType":"Payment","Account":"rSender","Destination":"rDest","Amount":"5000000",` + meta + `},
		{"hash":"A2","TransactionType":"Payment","Account":"rSender","Destination":"rDest","Amount":"5000000","SendMax":{"currency":"USD","issuer":"rIssuer","value":"5"},` + meta + `},
		{"hash":"A3","TransactionType":"Payment","Account":"rSender","Destination":"rDest","Amount":{"currency":"USD","issuer":"rIssuer","value":"5"},` + meta + `},
		{"hash":"A4","TransactionType":"AccountSet","Account":"rSender",` + meta + `}
	]}}}`

	client, _ := newRPCStub(t, body)

	result, err := client.GetBlockResult(t.Context(), 100)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 4)

	native := result.Transactions[0]
	require.True(t, native.IsNativePayment)
	require.Equal(t, new(uint64(5000000)), native.IntendedReceivingAmount)

	// A cross-currency payment delivering XRP states its Amount in drops but
	// can carry tfPartialPayment; the indexed amount must be zero.
	crossCurrency := result.Transactions[1]
	require.False(t, crossCurrency.IsNativePayment)
	require.Equal(t, new(uint64), crossCurrency.IntendedReceivingAmount)

	iou := result.Transactions[2]
	require.False(t, iou.IsNativePayment)
	require.Equal(t, new(uint64), iou.IntendedReceivingAmount)

	nonPayment := result.Transactions[3]
	require.False(t, nonPayment.IsNativePayment)
	require.Nil(t, nonPayment.IntendedReceivingAmount)
}

func TestLedgerRPCErrorIsNotBlockNotFound(t *testing.T) {
	client, _ := newRPCStub(t, `{"result":{"error":"internal","error_code":73,"error_message":"Internal error.","status":"error"}}`)

	_, err := client.GetBlockTimestamp(t.Context(), 42)
	require.NotErrorIs(t, err, indexer.ErrBlockNotFound)
	require.ErrorContains(t, err, "internal")
}
