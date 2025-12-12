package xrp_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/flare-foundation/verifier-xrp-indexer/internal/xrp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testBlockNum       = uint64(1725668)
	testBlockTimestamp = uint64(783002761) + xrp.XRPTimeToUTD
)

// Tests for TestGetLatestBlockInfo, TestGetBlockTimestamp and TestGetBlockResult
type XrpTestSuite struct {
	suite.Suite
	chainClient indexer.BlockchainClient[xrp.Block, xrp.Transaction, struct{}]
}

func TestXrpTestSuite(t *testing.T) {
	suite.Run(t, &XrpTestSuite{})
}

func (suite *XrpTestSuite) SetupTest() {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	var err error
	suite.chainClient, err = xrp.New(&cfg)
	require.NoError(suite.T(), err)
}

func (suite *XrpTestSuite) TestGetLatestBlockInfo(ctx context.Context) {
	blockInfo, err := suite.chainClient.GetLatestBlockInfo(ctx)
	suite.NoError(err)

	timeNow := uint64(time.Now().Unix())
	suite.Greater(timeNow, blockInfo.Timestamp-10)
	suite.Greater(blockInfo.Timestamp+60, timeNow)

	suite.Greater(blockInfo.BlockNumber, testBlockNum)
}

func (suite *XrpTestSuite) TestGetBlockTimestamp(ctx context.Context) {
	timestamp, err := suite.chainClient.GetBlockTimestamp(ctx, testBlockNum)
	suite.NoError(err)

	suite.Equal(timestamp, testBlockTimestamp)
}

func (suite *XrpTestSuite) TestGetBlockResult(ctx context.Context) {
	blockResult, err := suite.chainClient.GetBlockResult(ctx, testBlockNum)
	suite.NoError(err)

	suite.Equal(blockResult.Block.GetBlockNumber(), testBlockNum)
	suite.Equal(blockResult.Block.GetTimestamp(), testBlockTimestamp)
	suite.Equal(blockResult.Block.Transactions, uint64(10))
	suite.Equal(blockResult.Block.Hash, "e6ed42458de170a4d95544561c7df715c3a808ead9a3d1d669d187366fe568f6")
	suite.Equal(blockResult.Transactions[0].Hash, "1f572e746a69edde0c134824491567cc438cfb18a40aa0fd321e8143e70e9064")
	suite.Equal(blockResult.Transactions[0].BlockNumber, testBlockNum)
	suite.Equal(blockResult.Transactions[0].Timestamp, testBlockTimestamp)
	suite.Equal(blockResult.Transactions[0].PaymentReference, "")
	suite.Equal(blockResult.Transactions[0].IsNativePayment, true)
	suite.Equal(blockResult.Transactions[0].SourceAddressesRoot, "674fa9a46079864ce1744486bd1a7069794c8aade76b2d0424c4e716fba4f4ef")

	cupaloy.SnapshotT(suite.T(), blockResult)
}

// Tests for TestGetLatestBlockInfo, TestGetBlockTimestamp and TestGetBlockResult
// with wrong urls in client
type XrpWrongUrlTestSuite struct {
	suite.Suite
	chainClient indexer.BlockchainClient[xrp.Block, xrp.Transaction, struct{}]
}

func TestXrpWrongUrlTestSuite(t *testing.T) {
	suite.Run(t, &XrpWrongUrlTestSuite{})
}

func (suite *XrpWrongUrlTestSuite) SetupTest() {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:512345"}

	var err error
	suite.chainClient, err = xrp.New(&cfg)
	require.NoError(suite.T(), err)
}

func (suite *XrpWrongUrlTestSuite) TestGetLatestBlockInfo(ctx context.Context) {
	_, err := suite.chainClient.GetLatestBlockInfo(ctx)
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockTimestamp(ctx context.Context) {
	_, err := suite.chainClient.GetBlockTimestamp(ctx, testBlockNum)
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockResult(ctx context.Context) {
	_, err := suite.chainClient.GetBlockResult(ctx, testBlockNum)
	suite.Error(err)
}

// Test for defining client with empty url
func TestEmptyUrl(t *testing.T) {
	cfg := xrp.Config{""}

	var err error
	_, err = xrp.New(&cfg)
	require.Error(t, err)
}

func TestGetServerInfo(t *testing.T) {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	chainClient, err := xrp.New(&cfg)
	require.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	serverInfo, err := chainClient.GetServerInfo(ctx)
	cancelFunc()
	require.NoError(t, err)

	_, err = strconv.Atoi(serverInfo[0:1])
	require.NoError(t, err)
}
