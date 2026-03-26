package xrp

import (
	"os"
	"strings"

	"github.com/flare-foundation/verifier-indexer-framework/pkg/config"
)

// XRPTimeOffset is the seconds between the XRP epoch (Jan 1 2000) and the Unix epoch (Jan 1 1970).
const XRPTimeOffset = uint64(946684800)

var _ config.EnvOverrideable = new(Config)

// Config holds the configuration for the XRP RPC client.
type Config struct {
	Url string `toml:"url"`
}

// ApplyEnvOverrides overrides config fields from environment variables.
// XRP_URL overrides Url.
func (c *Config) ApplyEnvOverrides() error {
	if url, ok := os.LookupEnv("XRP_URL"); ok {
		c.Url = url
	}
	return nil
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
