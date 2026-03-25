# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XRP Ledger blockchain indexer built in Go. Collects blocks and transactions from XRP and stores them in PostgreSQL. Built on the `verifier-indexer-framework` which provides the indexing loop, database management, and configuration loading — this repo implements the XRP-specific `BlockchainClient` interface.

## Build & Run Commands

```bash
# Build
go build ./cmd/indexer

# Run locally (two terminals):
# Terminal 1: start PostgreSQL
cd tests/ && docker compose up postgresdb
# Terminal 2: run the indexer
go run ./cmd/indexer --config tests/config_test.toml

# Test (requires running xrpld — see below)
go test ./...

# Run a single test
go test ./internal/xrp/ -run TestFirstMemoDataHash
# Run a single subtest
go test ./internal/xrp/ -run TestXrpTestSuite/TestGetBlockResult

# Lint (uses golangci-lint v2+)
golangci-lint run --timeout 5m0s

# Format
go fmt ./...
```

## Test Infrastructure

There are two types of tests:
- **Unit tests** (e.g., `TestFirstMemoDataHash` in `memo_test.go`): Run without external dependencies, use `package xrp` to test unexported functions.
- **Integration tests** (e.g., `XrpTestSuite`): Require a running xrpld standalone node. The test setup creates wallets, funds accounts, and submits transactions against the local xrpld.

```bash
# Start xrpld standalone node (from repo root)
docker compose up -d
# Tests connect to xrpld RPC at http://localhost:5005
# Override with XRP_RPC_URL env var
```

Test suites use `testify/suite`. Integration tests are in `xrp_test.go` (package `xrp_test`), unit tests are in `memo_test.go` (package `xrp`).

## Architecture

**Entry point**: `cmd/indexer/main.go` — minimal bootstrap that calls the framework's `Run()` with XRP-specific types. The framework handles config loading, CLI arg parsing (`--config`, default `config.toml`), logger setup, database init, and the indexing loop.

**Core implementation**: `internal/xrp/` (only 3 source files):
- `xrp.go` — `xrpClient` implementing `indexer.BlockchainClient` interface (GetLatestBlockInfo, GetBlockTimestamp, GetBlockResult, GetServerInfo). Handles JSON-RPC communication with XRP nodes, transaction parsing, and field extraction.
- `entities.go` — GORM models for `Block` and `Transaction` tables. Implements `database.Deletable` for history cleanup.

**Transaction field extraction** (in `GetBlockResult`):
- `PaymentReference`: From first memo's MemoData if exactly 64 hex chars (payment transactions only)
- `FirstMemoDataHash`: Keccak256 hash of hex-decoded first memo MemoData
- `IsNativePayment`: True if Amount is a digit string (drops) or has `currency: "XRP"`
- `SourceAddressesRoot`: Merkle root of Keccak256-hashed accounts with decreased balance
- `SourceAddress`, `Sequence`, `TicketSequence`, `DestinationTag`: Passed through from XRP transaction JSON

**Key domain details**:
- XRP timestamps use a custom epoch offset: `XRPTimeToUTD = 946684800` (seconds from Unix epoch to 2000-01-01)
- XRP MemoData is hex-encoded — decode before hashing
- Only `validated: true` ledgers are processed
- The XRPL testnet is periodically reset; old blocks become unavailable

## Configuration

TOML-based config (see `config.example.toml`). The framework loads `config.toml` by default, overridable with `--config` flag or `CONFIG_FILE` env var. The `[blockchain].url` field can be overridden with the `XRP_URL` env var.

Key `[indexer]` settings: `start_block_number` and `end_block_number` control the block range. With `history_drop` enabled (seconds), `start_block_number` is ignored and the indexer starts from the history window.

## Local Framework Development

To develop against a local fork of `verifier-indexer-framework`, add a `replace` directive to `go.mod` (outside the `require` block):

```
replace github.com/flare-foundation/verifier-indexer-framework => ../verifier-indexer-framework
```

Keep the original `require` line with version intact. Remove the `replace` before committing.

## Linting

Strict `golangci-lint` config in `.golangci.yml` with ~28 linters enabled including `exhaustive` (switch completeness), `nolintlint` (requires specific linter in nolint directives), and `bodyclose`.
