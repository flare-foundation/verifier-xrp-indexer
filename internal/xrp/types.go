package xrp

import "encoding/json"

type xrpRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type ledgerParams struct {
	LedgerIndex  string `json:"ledger_index"`
	Transactions bool   `json:"transactions"`
	Expand       bool   `json:"expand"`
	OwnerFunds   bool   `json:"owner_funds"`
}

type ledgerResponse struct {
	Result ledgerResult `json:"result"`
}

type ledgerResult struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	LedgerIndex  uint64 `json:"ledger_index"`
	LedgerHash   string `json:"ledger_hash"`
	Validated    bool   `json:"validated"`
	Ledger       ledger `json:"ledger"`
}

type ledger struct {
	CloseTime    uint64 `json:"close_time"`
	Transactions []json.RawMessage
}

type amount struct {
	Currency string `json:"currency"`
}

type meta struct {
	AffectedNodes []affectedNodes `json:"AffectedNodes"`
}

type affectedNodes struct {
	ModifiedNode nodeChange  `json:"ModifiedNode"`
	CreatedNode  createdNode `json:"CreatedNode"`
	DeletedNode  nodeChange  `json:"DeletedNode"`
}

// nodeChange represents a ModifiedNode or DeletedNode entry in transaction
// metadata. Both share the same structure with FinalFields and PreviousFields.
type nodeChange struct {
	FinalFields     fields `json:"FinalFields"`
	PreviousFields  fields `json:"PreviousFields"`
	LedgerEntryType string `json:"LedgerEntryType"`
}

// createdNode represents a CreatedNode entry in transaction metadata.
// Unlike modified and deleted nodes, it uses NewFields instead of
// FinalFields/PreviousFields.
type createdNode struct {
	NewFields       fields `json:"NewFields"`
	LedgerEntryType string `json:"LedgerEntryType"`
}

type fields struct {
	Account string          `json:"Account"`
	Balance json.RawMessage `json:"Balance"`
}

type serverInfoResponse struct {
	Result serverInfoResult `json:"result"`
}

type serverInfoResult struct {
	Info serverInfo `json:"info"`
}

type serverInfo struct {
	BuildVersion string `json:"build_version"`
	ServerState  string `json:"server_state"`

	// Some nodes return "rippled_version" instead of "build_version"
	RippledVersion string `json:"rippled_version"`
}

var getLatestParams = ledgerParams{
	LedgerIndex:  "validated",
	Transactions: false,
	Expand:       false,
	OwnerFunds:   false,
}

var getServerState = xrpRequest{
	Method: "server_info",
}
