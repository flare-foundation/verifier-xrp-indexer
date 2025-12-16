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
	testBlockNum       = uint64(13261049)
	testBlockTimestamp = uint64(1765912553)
)

// Tests for TestGetLatestBlockInfo, TestGetBlockTimestamp and TestGetBlockResult
type XrpTestSuite struct {
	suite.Suite
	chainClient indexer.BlockchainClient[xrp.Block, xrp.Transaction, struct{}]
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func TestXrpTestSuite(t *testing.T) {
	suite.Run(t, &XrpTestSuite{})
}

func (suite *XrpTestSuite) SetupTest() {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:51234"}

	var err error
	suite.chainClient, err = xrp.New(&cfg)
	require.NoError(suite.T(), err)

	suite.ctx, suite.cancelFunc = context.WithTimeout(context.Background(), 3*time.Second)
}

func (suite *XrpTestSuite) TestGetLatestBlockInfo() {
	blockInfo, err := suite.chainClient.GetLatestBlockInfo(suite.ctx)
	suite.cancelFunc()
	suite.NoError(err)

	timeNow := uint64(time.Now().Unix())
	suite.Greater(timeNow, blockInfo.Timestamp-10)
	suite.Greater(blockInfo.Timestamp+60, timeNow)

	suite.Greater(blockInfo.BlockNumber, testBlockNum)
}

func (suite *XrpTestSuite) TestGetBlockTimestamp() {
	timestamp, err := suite.chainClient.GetBlockTimestamp(suite.ctx, testBlockNum)
	suite.cancelFunc()
	suite.NoError(err)

	suite.Equal(timestamp, testBlockTimestamp)
}

func (suite *XrpTestSuite) TestGetBlockResult() {
	blockResult, err := suite.chainClient.GetBlockResult(suite.ctx, testBlockNum)
	suite.cancelFunc()
	suite.Require().NoError(err)
	suite.Require().NotNil(blockResult)

	suite.Equal(blockResult.Block.GetBlockNumber(), testBlockNum)
	suite.Equal(blockResult.Block.GetTimestamp(), testBlockTimestamp)

	cupaloy.SnapshotT(suite.T(), blockResult)
}

// Tests for TestGetLatestBlockInfo, TestGetBlockTimestamp and TestGetBlockResult
// with wrong urls in client
type XrpWrongUrlTestSuite struct {
	suite.Suite
	chainClient indexer.BlockchainClient[xrp.Block, xrp.Transaction, struct{}]
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func TestXrpWrongUrlTestSuite(t *testing.T) {
	suite.Run(t, &XrpWrongUrlTestSuite{})
}

func (suite *XrpWrongUrlTestSuite) SetupTest() {
	cfg := xrp.Config{"https://s.altnet.rippletest.net:512345"}

	var err error
	suite.chainClient, err = xrp.New(&cfg)
	require.NoError(suite.T(), err)

	suite.ctx, suite.cancelFunc = context.WithTimeout(context.Background(), 3*time.Second)
}

func (suite *XrpWrongUrlTestSuite) TestGetLatestBlockInfo() {
	_, err := suite.chainClient.GetLatestBlockInfo(suite.ctx)
	suite.cancelFunc()
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockTimestamp() {
	_, err := suite.chainClient.GetBlockTimestamp(suite.ctx, testBlockNum)
	suite.cancelFunc()
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockResult() {
	_, err := suite.chainClient.GetBlockResult(suite.ctx, testBlockNum)
	suite.cancelFunc()
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
