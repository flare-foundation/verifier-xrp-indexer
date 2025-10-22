//go:build integration
// +build integration

package framework

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/config"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/database"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultConfigFile = "../../tests/test_config.toml"

func TestRun(t *testing.T) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = defaultConfigFile
	}

	input := Input[dbBlock, *ExampleConfig, dbTransaction]{
		NewBlockchainClient: NewTestBlockchain,
	}

	args := CLIArgs{ConfigFile: configFile}
	err := runWithArgs(input, args)
	require.NoError(t, err)

	cfg := config.BaseConfig{}
	err = config.ReadFile(configFile, &cfg)
	require.NoError(t, err)

	db, err := database.Connect(&config.DB{Host: cfg.DB.Host, Port: cfg.DB.Port, Username: cfg.DB.Username, Password: cfg.DB.Password, DBName: cfg.DB.DBName})
	require.NoError(t, err)

	state := new(database.State)
	err = db.First(state, 1).Error
	require.NoError(t, err)

	assert.GreaterOrEqual(t, state.FirstIndexedBlockNumber, uint64(300))
	assert.GreaterOrEqual(t, uint64(315), state.FirstIndexedBlockNumber)
	assert.GreaterOrEqual(t, state.LastIndexedBlockNumber, uint64(509))
	assert.GreaterOrEqual(t, uint64(512), state.LastIndexedBlockNumber)
}

type TestBlockchain struct {
	startTime time.Time
}

func NewTestBlockchain(cfg *ExampleConfig) (indexer.BlockchainClient[dbBlock, dbTransaction], error) {
	return TestBlockchain{startTime: time.Now()}, nil
}

func (e TestBlockchain) GetLatestBlockInfo(context.Context) (*indexer.BlockInfo, error) {
	timeSince := uint64(time.Since(e.startTime).Seconds())

	return &indexer.BlockInfo{BlockNumber: timeSince + 500, Timestamp: timeSince + 1000}, nil
}

func (e TestBlockchain) GetBlockResult(ctx context.Context, blockNum uint64) (*indexer.BlockResult[dbBlock, dbTransaction], error) {
	hash := strconv.Itoa(int(blockNum))
	hash = strings.Repeat("0", 64-len(hash)) + hash
	block := dbBlock{
		BlockNumber: blockNum,
		Timestamp:   blockNum + 500,
		Hash:        hash,
	}

	transactions := []dbTransaction{{Hash: strings.Repeat("f", 64), Timestamp: blockNum + 500, BlockNumber: blockNum}, {Hash: strings.Repeat("e", 64), Timestamp: blockNum + 500, BlockNumber: blockNum}}

	return &indexer.BlockResult[dbBlock, dbTransaction]{Block: block, Transactions: transactions}, nil
}

func (e TestBlockchain) GetBlockTimestamp(ctx context.Context, blockNum uint64) (uint64, error) {
	return blockNum + 500, nil
}

func (e TestBlockchain) GetServerInfo(ctx context.Context) (string, error) {
	return "0.0.1_test", nil
}

type ExampleConfig struct{}

// Required for interface but not used in this example
func (e *ExampleConfig) ApplyEnvOverrides() {}

type dbBlock struct {
	Hash        string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber uint64 `gorm:"index"`
	Timestamp   uint64 `gorm:"index"`
}

func (e dbBlock) GetBlockNumber() uint64 {
	return e.BlockNumber
}

func (e dbBlock) GetTimestamp() uint64 {
	return e.Timestamp
}

type dbTransaction struct {
	Hash        string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber uint64 `gorm:"index"`
	Timestamp   uint64 `gorm:"index"`
}
