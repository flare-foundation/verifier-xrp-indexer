# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Opt-in readiness endpoint from the framework: with `[health] enabled = true` the indexer serves `GET /health`, answering 200 while the indexed range is current and 503 with a JSON status otherwise, including `chain_stale` when the node has not been reached recently. Off by default.

### Changed

- **Breaking:** Go 1.26.8 is required (go directive, CI image and Docker builder). Earlier toolchains carry reachable stdlib vulnerabilities.
- Unknown keys in the TOML configuration are logged as a warning at startup instead of silently ignored.
- `IsNativePayment` is true only for XRP-to-XRP payments; payments with `SendMax` or a non-XRP `Amount` are no longer native.
- `IntendedReceivingAmount` is zero for non-native payments so they never pass the nonexistence amount filter: a sender can force such a payment to fail with a result code classified as receiver's fault, and a failed payment would count with its stated `Amount`. Column semantics: null = not a payment, zero = non-native payment, positive = intended drops of a native XRP payment.
- Re-indexing a range overwrites existing rows instead of skipping them (framework), so rows written by older versions are repaired.
- A signal-initiated shutdown exits zero.
- `api_version` 1 is pinned in every rippled request so parsing does not depend on the node's default API version.
- The RPC client keeps every idle connection to the node instead of the default two per host, so concurrent fetches no longer re-handshake on each batch.
- The shipped configurations set `confirmations = 3`. Public XRPL endpoints are load balanced and their backends were measured two ledgers apart; at `confirmations = 1` a ledger a lagging backend did not have yet was reported not found and the indexer exited on it. Such a ledger is now retried for the backoff window before the indexer exits (framework).

### Fixed

- History drop deletes blocks before transactions. The previous order left block rows whose transactions were already gone for the duration of a drop, and indefinitely after a crash mid-drop; the verifier API gates coverage on the block row alone, so a payment in that window could be attested as nonexistent.
- A `start_block_number` the node has pruned no longer crash-loops: rippled's `lgrNotFound` for a numeric ledger index is reported as the framework's block-not-found sentinel and the oldest available ledger is used instead. `lgrNotFound` for the latest validated ledger stays transient.
- Transactions in a validated ledger that cannot be parsed abort the indexer immediately with a clear error instead of being retried for the full backoff window.
- The error for a not-yet-validated ledger no longer includes the whole ledger response.

### Security

- The configuration file is no longer dumped to the debug log at startup, which exposed the database password.
- `github.com/ethereum/go-ethereum` bumped from v1.14.12 to v1.17.5, closing GO-2025-3436, GO-2026-4314, GO-2026-4315 and GO-2026-4508.

## [2.0.0] - 2026-04-17

### Added

- `TransactionStatus` field indicating whether the transaction result is a success or destination-related failure.
- `DestinationAddressHash` field storing the Keccak256 hash of the destination address (NULL when not set).
- `IntendedReceivingAmount` field storing the native XRP drop amount the sender intended to deliver (NULL for non-payments and non-native tokens).
- Composite indexes `idx_payref_block` and `idx_dest_block` for payment reference and destination address queries.

### Changed

- Updated framework version. This includes changes to avoid locking up the DB excessively when running large history drop iterations.
- Fixed `sourceAddressesRoot` calculation.
