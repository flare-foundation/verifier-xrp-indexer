package xrp

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/indexer"
)

// XRPTimeOffset is the seconds between the XRP epoch (Jan 1 2000) and the Unix epoch (Jan 1 1970).
const XRPTimeOffset = uint64(946684800)

// Config holds the configuration for the XRP RPC client.
type Config struct {
	Url string `toml:"url"`
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

// configFilePath resolves the config file path from the --config flag,
// CONFIG_FILE environment variable, or defaults to "config.toml".
func configFilePath() string {
	const prefix = "--config="
	for i, arg := range os.Args {
		if v, ok := strings.CutPrefix(arg, prefix); ok {
			return v
		}
		if arg == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	if path, ok := os.LookupEnv("CONFIG_FILE"); ok {
		return path
	}
	return "config.toml"
}

// ApplyEnvOverrides overrides config fields from environment variables.
// XRP_URL overrides Url.
func (c *Config) ApplyEnvOverrides() error {
	if url, ok := os.LookupEnv("XRP_URL"); ok {
		c.Url = url
	}
	return nil
}
