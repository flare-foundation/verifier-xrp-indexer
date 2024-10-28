package xrp_test

import (
	"context"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/flare-foundation/verifier-xrp-indexer/internal/xrp"
	"github.com/stretchr/testify/require"
)

var testBlockNum = uint64(1725668)
var testBlockTimestamp = uint64(783002761)

func TestGetLatestBlockInfo(t *testing.T) {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	chainClient, err := xrp.New(&cfg)
	require.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	blockInfo, err := chainClient.GetLatestBlockInfo(ctx)
	cancelFunc()
	require.NoError(t, err)

	const layout = "2006-Jan-02"
	startTime, err := time.Parse(layout, "2000-Jan-01")
	require.NoError(t, err)

	timeNow := uint64(time.Since(startTime).Seconds())

	require.Greater(t, timeNow, blockInfo.Timestamp)
	require.Greater(t, blockInfo.Timestamp+60, timeNow)

	require.Greater(t, blockInfo.BlockNumber, testBlockNum)
}

func TestGetBlockTimestamp(t *testing.T) {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	chainClient, err := xrp.New(&cfg)
	require.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	timestamp, err := chainClient.GetBlockTimestamp(ctx, testBlockNum)
	cancelFunc()
	require.NoError(t, err)

	require.Equal(t, timestamp, testBlockTimestamp)
}

func TestGetBlockResult(t *testing.T) {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	chainClient, err := xrp.New(&cfg)
	require.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	blockResult, err := chainClient.GetBlockResult(ctx, testBlockNum)
	cancelFunc()
	require.NoError(t, err)

	require.Equal(t, testBlockNum, blockResult.Block.GetBlockNumber())
	require.Equal(t, testBlockTimestamp, blockResult.Block.GetTimestamp())

	cupaloy.SnapshotT(t, blockResult)
}
