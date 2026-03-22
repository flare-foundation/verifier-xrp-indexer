# Contributing

This document describes the process of contributing to this project. It is
intended for anyone considering opening an issue or pull request.

## AI Assistance

> [!IMPORTANT]
>
> If you are using **any kind of AI assistance** to contribute to this project,
> it must be disclosed in the pull request.

If you are using any kind of AI assistance while contributing to this project,
**this must be disclosed in the pull request**, along with the extent to which
AI assistance was used. Trivial tab-completion doesn't need to be disclosed, as
long as it is limited to single keywords or short phrases.

An example disclosure:

> This PR was written primarily by Claude Code.

Or a more detailed disclosure:

> I consulted ChatGPT to understand the codebase but the solution was fully
> authored manually by myself.

## Quick start

If you'd like to contribute, report a bug, suggest a feature or you've
implemented a feature you should open an issue or pull request.

Any contribution to the project is expected to contain code that is formatted,
linted and that the existing tests still pass. Adding unit tests for new code is
also welcome.

## Dev environment

The indexer is implemented using Go - it is recommended to use version 1.25 or later.
An RPC connection URL for XRP and a Postgres database are required in order to run
the main indexer. Public RPC nodes suitable for local development may be found on 
[public-servers](https://xrpl.org/docs/tutorials/public-servers) although using a
private connection from a provider such as Quicknode or a self-hosted node is
recommended for live deployments. Configuration should be provided via a `config.toml`
file - you can copy `config.example.toml` and modify to connect to your specific RPC
provider and database instance as well as set other parameters.

## Running locally

Start a PostgreSQL instance using the docker compose file from the tests folder:

```
$ cd tests/
$ docker compose up postgresdb
```

In another terminal, run the indexer pointing at the test config:

```
$ go run ./cmd/indexer --config tests/config_test.toml
```

### Working with a local framework

If you need to develop against a local version of the
`verifier-indexer-framework`, add a `replace` directive to `go.mod`:

```
replace github.com/flare-foundation/verifier-indexer-framework => ../verifier-indexer-framework
```

Adjust the path to point to your local fork. Run `go mod tidy` after adding the
directive. Remove it before committing when you're ready to switch back to a
published version.

## Linting and formatting

`golangci-lint` is used for linting - install instructions may be found
\[[here](https://golangci-lint.run/docs/welcome/install/#local-installation)\]. Once installed you can run with:

```
$ golangci-lint run
```

For formatting use the standard `go fmt` command:

```
$ go fmt ./...
```

It is recommended to set up your editor to run both of these automatically. 

## Testing

Run the tests with:

```
$ go test ./...
```

## Release process (if applicable)

Releases are made by creating a tag. The CHANGELOG.md should also be updated with the release details.
