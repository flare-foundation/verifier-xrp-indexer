# Verifier XRP Indexer

This is an indexer for the XRP blockchain. It is using the Verifier Indexer
Framework (https://github.com/flare-foundation/verifier-indexer-framework). It
collects Blocks and Transactions from the XRP blockchain. The indexer is
configured via a configuration file - see `config.example.toml` for an example
configuration.

## Installation

To install the indexer, you need to have Go installed to the latest version -
currently 1.23.0.

Now you may install the indexer by running the following command:

```shell
go install github.com/flare-foundation/verifier-xrp-indexer/cmd/indexer@latest
```

This will install the `indexer` command globally.

## Running the indexer

### Prerequisites

#### Database

To run the indexer a PostgreSQL database needs to be deployed. An example database is
provided as docker image in `tests` directory which can be used with

```bash
docker compose up postgresdb
```

from the `tests` repository.

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
confirmations = 1 # number of confirmed blocks before the data is included in the DB
max_block_range = 100 # size of a batch to be repeatedly queried and saved in a dataset
max_concurrency = 10 # number of concurrent processes querying data from the RPC node
start_block_number = 780000

[blockchain]
url = "https://s.altnet.rippletest.net:51234"

[timeout]
request_timeout_millis = 3000 # timeout requests to the chain and the dataset
backoff_max_elapsed_time_seconds = 300 # maximum time between retries

[logger]
level = "DEBUG"
file = "logs/xrp_indexer.log"
console = true
```

### Running the code

Assuming that the indexer was installed globally you can use `indexer` command to
run it. Alternatively you can use `go run ./cmd/indexer` to build and run the binary.

By default the indexer will look for a configuration file named `config.toml`
in the current working directory. You can specify a different configuration
file by passing `--config <filepath>` when running the indexer. For example:

```go
go run cmd/indexer/main.go --config tests/config_test.toml
```
