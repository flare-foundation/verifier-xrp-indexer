<!-- LOGO -->

<div align="center">
  <a href="https://flare.network/" target="blank">
    <img src="https://content.flare.network/Flare-2.svg" width="300" alt="Flare Logo" />
  </a>
  <br />
  Indexer for the XRP blockchain 
  <br />
  <a href="CONTRIBUTING.md">Contributing</a>
  ·
  <a href="SECURITY.md">Security</a>
  ·
  <a href="CHANGELOG.md">Changelog</a>
</div>

# Verifier XRP Indexer

This is an indexer for the XRP blockchain.
It is using the [Verifier Indexer Framework](https://github.com/flare-foundation/verifier-indexer-framework).
It collects Blocks and Transactions from the XRP blockchain.
The indexer is configured via a configuration file - see `config.example.toml` for an example configuration.

## Installation

To install the indexer, you need Go 1.26.8 or later.

Now you may install the indexer by running the following command:

```shell
go install github.com/flare-foundation/verifier-xrp-indexer/v2/cmd/indexer@latest
```

This will install the `indexer` command globally.

## Running the indexer

### Prerequisites

#### Database

To run the indexer a PostgreSQL database needs to be deployed.
An example database is provided as docker image in `tests` directory which can be used with

```bash
docker compose up postgresdb
```

from the `tests` directory.

##### Config file

Provide a `.toml` config file with the following fields (mostly self explanatory)

```toml
[db]
username = "username" # can be specified with env DB_USERNAME
password = "password" # can be specified with env DB_PASSWORD
db_name = "flare_xrp_indexer"
port = 5432
drop_table_at_start = false
history_drop = 3600 # delete all historic data in database older than this value, in seconds
history_drop_frequency = 600 # frequency of history drops, in seconds

[indexer]
confirmations = 1 # number of confirmed blocks before the data is included in the DB; raise above the backend skew of load-balanced public endpoints, see below
max_block_range = 100 # size of a batch to be repeatedly queried and saved in a dataset
max_concurrency = 10 # number of concurrent processes querying data from the RPC node
start_block_number = 780000

[blockchain]
url = "https://s.altnet.rippletest.net:51234" # can be specified with env XRP_URL

[timeout]
request_timeout_millis = 3000 # timeout requests to the chain and the dataset
backoff_max_elapsed_time_seconds = 300 # maximum time between retries

[health]
enabled = false # opt-in GET /health readiness endpoint, see below
listen_address = ":8080"

[logger]
level = "DEBUG"
file = "logs/xrp_indexer.log"
console = true
```

Unknown keys in the config file are logged as a warning at startup and ignored, so check the log after editing it.

Public XRPL endpoints such as `xrplcluster.com` or the altnet cluster are load balanced.
Their backends can disagree on the latest validated ledger, and a ledger that a lagging backend does not have yet is reported as not found, which the indexer treats as permanent and exits on.
Keep `confirmations` above the expected skew when using such an endpoint; a dedicated node is not affected.

### Running the code

Assuming that the indexer was installed globally you can use `indexer` command to run it.
Alternatively you can use `go run ./cmd/indexer` to build and run the binary.

By default the indexer will look for a configuration file named `config.toml` in the current working directory.
You can specify a different configuration file by passing `--config <filepath>` when running the indexer.
For example:

```bash
go run cmd/indexer/main.go --config tests/config_test.toml
```

### Health endpoint

With `enabled = true` in the `[health]` section the indexer serves `GET /health` on `listen_address` (default `:8080`).
It answers 200 while the indexed range is current and 503 with a JSON status (`initializing`, `chain_stale`, `catching_up`, `stalled` or `unavailable`) otherwise.
`chain_stale` covers a node that has not been reached recently, including during the startup retries after a restart.
Nothing listens unless it is enabled.

The tolerated block lag, progress age and chain-poll age derive from `confirmations`, `max_block_range`, `max_concurrency` and `request_timeout_millis` when `max_block_lag`, `max_progress_age_seconds` and `max_chain_age_seconds` are left at zero.
The effective values are echoed in every response.

On Kubernetes use it for `readinessProbe` and `startupProbe` only.
Do not add a `livenessProbe`: the indexer exits on persistent errors, so restart-on-exit already covers liveness, and a legitimate backfill answers 503 for its whole duration.
The port is unauthenticated; restrict it with a network policy or bind `listen_address` to a specific interface.

### Running the tests

The indexer tests depend on a local rippled instance running in standalone mode.
The easiest way to create this is to use the `docker-compose.yaml` file provided:

Start the rippled instance:

```bash
docker compose up -d
```

Run the tests:

```bash
go test ./...
```

Stop the rippled instance:

```bash
docker compose down
```
