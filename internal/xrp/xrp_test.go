package xrp_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/flare-foundation/verifier-xrp-indexer/v2/internal/xrp"
	"github.com/stretchr/testify/require"
)

const defaultRPCURL = "http://localhost:5005"

var rpcURL string

func init() {
	rpcURL = os.Getenv("XRP_RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}
}

func TestXrp(t *testing.T) {
	awaitReady(t)

	source := createWallet(t)
	dest := createWallet(t)

	fundAccount(t, source.Account, "100000000")
	_, err := callRPC("ledger_accept")
	require.NoError(t, err)

	submitPayment(t, source, dest.Account, "1000000")
	_, err = callRPC("ledger_accept")
	require.NoError(t, err)

	currentLedgerIndex := getValidatedLedgerIndex(t)
	t.Log("ledger_current_index:", currentLedgerIndex)
	testLedger := currentLedgerIndex - 1

	chainClient, err := xrp.New(&xrp.Config{Url: rpcURL})
	require.NoError(t, err)

	t.Run("GetLatestBlockInfo", func(t *testing.T) {
		blockInfo, err := chainClient.GetLatestBlockInfo(t.Context())
		require.NoError(t, err)

		timeNow := uint64(time.Now().Unix())
		require.Greater(t, timeNow, blockInfo.Timestamp-10)
		require.Greater(t, blockInfo.Timestamp+60, timeNow)
		require.GreaterOrEqual(t, blockInfo.BlockNumber, testLedger)
	})

	t.Run("GetBlockTimestamp", func(t *testing.T) {
		timestamp, err := chainClient.GetBlockTimestamp(t.Context(), testLedger)
		require.NoError(t, err)
		require.NotZero(t, timestamp)
	})

	t.Run("GetBlockResult", func(t *testing.T) {
		blockResult, err := chainClient.GetBlockResult(t.Context(), testLedger)
		require.NoError(t, err)
		require.NotNil(t, blockResult)

		require.Equal(t, testLedger, blockResult.Block.GetBlockNumber())
		require.NotZero(t, blockResult.Block.GetTimestamp())
		require.Equal(t, uint64(1), blockResult.Block.Transactions)

		require.Len(t, blockResult.Transactions, 1)
		tx := blockResult.Transactions[0]
		require.NotEmpty(t, tx.Hash)
		require.Equal(t, testLedger, tx.BlockNumber)
		require.NotZero(t, tx.Timestamp)
		require.Empty(t, tx.PaymentReference)
		require.NotEmpty(t, tx.Response)
		require.True(t, tx.IsNativePayment)
		require.NotEmpty(t, tx.SourceAddressesRoot)

		require.NotEmpty(t, tx.SourceAddress)
		require.NotZero(t, tx.Sequence)
		require.Zero(t, tx.TicketSequence)
		require.Empty(t, tx.FirstMemoDataHash)
		require.Nil(t, tx.DestinationTag)

		require.NotNil(t, tx.DestinationAddressHash)
		require.NotNil(t, tx.IntendedReceivingAmount)
		require.NotZero(t, *tx.IntendedReceivingAmount)
		require.True(t, tx.TransactionStatus)

		require.Empty(t, blockResult.Events)
	})
}

func TestWrongURL(t *testing.T) {
	chainClient, err := xrp.New(&xrp.Config{Url: "https://s.altnet.rippletest.net:512345"}) // wrong port
	require.NoError(t, err)

	t.Run("GetLatestBlockInfo", func(t *testing.T) {
		_, err := chainClient.GetLatestBlockInfo(t.Context())
		require.Error(t, err)
	})

	t.Run("GetBlockTimestamp", func(t *testing.T) {
		_, err := chainClient.GetBlockTimestamp(t.Context(), 1)
		require.Error(t, err)
	})

	t.Run("GetBlockResult", func(t *testing.T) {
		_, err := chainClient.GetBlockResult(t.Context(), 1)
		require.Error(t, err)
	})
}

func TestEmptyUrl(t *testing.T) {
	_, err := xrp.New(&xrp.Config{Url: ""})
	require.Error(t, err)
}

func TestGetServerInfo(t *testing.T) {
	chainClient, err := xrp.New(&xrp.Config{Url: rpcURL})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	serverInfo, err := chainClient.GetServerInfo(ctx)
	require.NoError(t, err)

	_, err = strconv.Atoi(serverInfo[0:1])
	require.NoError(t, err)
}
