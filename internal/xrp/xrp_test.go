package xrp_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/flare-foundation/verifier-xrp-indexer/internal/xrp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const defaultRPCURL = "http://localhost:5005"

var rpcURL string

func init() {
	rpcURL = os.Getenv("XRP_RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}
}

// Tests for TestGetLatestBlockInfo, TestGetBlockTimestamp and TestGetBlockResult
type XrpTestSuite struct {
	suite.Suite
	chainClient indexer.BlockchainClient[xrp.Block, xrp.Transaction, struct{}]
	testLedger  uint64
}

func TestXrpTestSuite(t *testing.T) {
	suite.Run(t, &XrpTestSuite{})
}

func (suite *XrpTestSuite) SetupSuite() {
	t := suite.T()

	awaitReady(t)

	// 1. Create wallets
	source := createWallet(t)
	dest := createWallet(t)

	// 2. Fund source
	fundAccount(t, source.Account, "100000000")
	_, err := callRPC("ledger_accept")
	suite.Require().NoError(err)

	// 3. Submit payment
	submitPayment(t, source, dest.Account, "1000000")
	_, err = callRPC("ledger_accept")
	suite.Require().NoError(err)

	// 4. Query ledger index
	currentLedgerIndex := getValidatedLedgerIndex(t)
	t.Log("ledger_current_index:", currentLedgerIndex)
	suite.testLedger = currentLedgerIndex - 1

	cfg := xrp.Config{Url: rpcURL}

	chainClient, err := xrp.New(&cfg)
	suite.Require().NoError(err)

	suite.chainClient = chainClient
}

func (suite *XrpTestSuite) TestGetLatestBlockInfo() {
	ctx := context.Background()
	blockInfo, err := suite.chainClient.GetLatestBlockInfo(ctx)
	suite.NoError(err)

	timeNow := uint64(time.Now().Unix())
	suite.Greater(timeNow, blockInfo.Timestamp-10)
	suite.Greater(blockInfo.Timestamp+60, timeNow)

	suite.GreaterOrEqual(blockInfo.BlockNumber, suite.testLedger)
}

func (suite *XrpTestSuite) TestGetBlockTimestamp() {
	ctx := context.Background()
	timestamp, err := suite.chainClient.GetBlockTimestamp(ctx, suite.testLedger)
	suite.Require().NoError(err)
	suite.NotZero(timestamp)
}

func (suite *XrpTestSuite) TestGetBlockResult() {
	ctx := context.Background()
	blockResult, err := suite.chainClient.GetBlockResult(ctx, suite.testLedger)
	suite.Require().NoError(err)
	suite.Require().NotNil(blockResult)

	suite.Require().Len(blockResult.Transactions, 1)
	tx := blockResult.Transactions[0]
	suite.NotEmpty(tx.Hash)
	suite.Equal(tx.BlockNumber, suite.testLedger)
	suite.NotZero(tx.Timestamp)
	suite.Empty(tx.PaymentReference)
	suite.NotEmpty(tx.Response)
	suite.True(tx.IsNativePayment)
	suite.NotEmpty(tx.SourceAddressesRoot)
	suite.NotEmpty(tx.SourceAddress)
	suite.NotZero(tx.Sequence)
	suite.Zero(tx.TicketSequence)
	suite.Empty(tx.FirstMemoDataHash)
	suite.Nil(tx.DestinationTag)

	suite.Empty(blockResult.Events)
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
	cfg := xrp.Config{"https://s.altnet.rippletest.net:512345"} // wrong port

	var err error
	suite.chainClient, err = xrp.New(&cfg)
	require.NoError(suite.T(), err)
}

func (suite *XrpWrongUrlTestSuite) TestGetLatestBlockInfo() {
	ctx := context.Background()
	_, err := suite.chainClient.GetLatestBlockInfo(ctx)
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockTimestamp() {
	ctx := context.Background()
	_, err := suite.chainClient.GetBlockTimestamp(ctx, 1)
	suite.Error(err)
}

func (suite *XrpWrongUrlTestSuite) TestGetBlockResult() {
	ctx := context.Background()
	_, err := suite.chainClient.GetBlockResult(ctx, 1)
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
	cfg := xrp.Config{Url: rpcURL}

	chainClient, err := xrp.New(&cfg)
	require.NoError(t, err)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	serverInfo, err := chainClient.GetServerInfo(ctx)
	cancelFunc()
	require.NoError(t, err)

	_, err = strconv.Atoi(serverInfo[0:1])
	require.NoError(t, err)
}
