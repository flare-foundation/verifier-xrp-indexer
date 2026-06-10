package xrp

import (
	"os"

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
