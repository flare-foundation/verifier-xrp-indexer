# Contributing

Thank you for considering improving out source code.
All contributions are welcome.

## Issues

_Sensitive security-related issues should not be reported publicly.
See [Security](SECURITY.md)._

To share ideas, considerations, or concerned open an issue.
Before filing an issue make sure the issue has not been already raised.
In the issue, answer the following questions:

- What is the issue?
- Why it is an issue?
- How do you propose to change it?

### Pull request

Before opening a pull request open an issue on why the request is needed.

To contribute: fork the repo, make your improvements, commit, and open a pull request.
The maintainers will review the request.

The request must:

- Reference the relevant issue,
- Follow standard golang guidelines,
- Be well documented,
- Be well tested,
- Compile,
- Pass all the tests,
- Pass all the linters,
- Be based on opened against `main` branch.

## Dev environment

The indexer is implemented using Go - it is recommended to use version 1.27.3 or later.
An RPC connection URL for XRPL and a Postgres database are required in order to run the main indexer.
Public RPC nodes suitable for local development may be found on [public-servers](https://xrpl.org/docs/tutorials/public-servers) although using a private connection from a provider such as Quicknode or a self-hosted node is recommended for live deployments. Configuration should be provided via a `config.toml` file - you can copy `config.example.toml` and modify to connect to your specific RPC provider and database instance as well as set other parameters.

## Running locally

Start a PostgreSQL instance using the docker compose file from the tests folder:

```bash
 cd tests/
 docker compose up postgresdb
```

In another terminal, run the indexer pointing at the test config:

```bash
 go run ./cmd/indexer --config tests/config_test.toml
```

### Working with a local framework

If you need to develop against a local version of the `verifier-indexer-framework`, add a `replace` directive to `go.mod`:

```
replace github.com/flare-foundation/verifier-indexer-framework => ../verifier-indexer-framework
```

Adjust the path to point to your local fork.
Run `go mod tidy` after adding the directive.
Remove it before committing when you are ready to switch back to a published version.
If an issue needs changes in `verifier-indexer-framework`, open an issue there first.

## Linting and formatting

[`golangci-lint`](https://golangci-lint.run/) is used for linting.
Make sure you have it installed.

```bash
 golangci-lint run
```

For formatting, use the standard `go fmt` command:

```bash
 go fmt ./...
```

## Testing

Run the tests with:

The test need a

```
go test -v ./...
```

## Release process (if applicable)

Releases are made by creating a tag.
The CHANGELOG.md should also be updated with the release details.

## AI Assistance

Any significant use of the AI assistance in the contribution MUST be disclosed in the pull request along with the extent of the use.

An example disclosure:

> This PR was written primarily by Claude Code.

Or a more detailed disclosure:

> I consulted ChatGPT for the following code snippets: ...
