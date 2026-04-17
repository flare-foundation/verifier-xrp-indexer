# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [[Unreleased]]

## [2.0.0] - 2026-04-17

### Added

- `TransactionStatus` field indicating whether the transaction result is a success or destination-related failure.
- `DestinationAddressHash` field storing the Keccak256 hash of the destination address (NULL when not set).
- `IntendedReceivingAmount` field storing the native XRP drop amount the sender intended to deliver (NULL for non-payments and non-native tokens).
- Composite indexes `idx_payref_block` and `idx_dest_block` for payment reference and destination address queries.

### Changed

- Updated framework version. This includes changes to avoid locking up the DB excessively when running large history drop iterations.
- Fixed `sourceAddressesRoot` calculation.
