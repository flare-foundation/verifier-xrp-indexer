package xrp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRPCStub returns a client backed by a test server that always responds
// with the given body.
func newRPCStub(t *testing.T, body string) indexer.BlockchainClient[*Block, Transaction, struct{}] {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	client, err := New(&Config{Url: server.URL})
	require.NoError(t, err)

	return client
}

func TestLedgerNotFoundMapsToErrBlockNotFound(t *testing.T) {
	client := newRPCStub(t, `{"result":{"error":"lgrNotFound","error_code":21,"error_message":"ledgerNotFound","request":{"command":"ledger","ledger_index":"42"},"status":"error"}}`)

	_, err := client.GetBlockTimestamp(t.Context(), 42)
	require.ErrorIs(t, err, indexer.ErrBlockNotFound)

	_, err = client.GetBlockResult(t.Context(), 42)
	require.ErrorIs(t, err, indexer.ErrBlockNotFound)
}

func TestLedgerRPCErrorIsNotBlockNotFound(t *testing.T) {
	client := newRPCStub(t, `{"result":{"error":"internal","error_code":73,"error_message":"Internal error.","status":"error"}}`)

	_, err := client.GetBlockTimestamp(t.Context(), 42)
	require.NotErrorIs(t, err, indexer.ErrBlockNotFound)
	require.ErrorContains(t, err, "internal")
}
