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
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	LedgerIndex  uint64    `json:"ledger_index"`
	LedgerHash   string    `json:"ledger_hash"`
	Validated    bool      `json:"validated"`
	Ledger       xrpLedger `json:"ledger"`
}

type xrpLedger struct {
	CloseTime    uint64 `json:"close_time"`
	Transactions []json.RawMessage
}

type xrpTransaction struct {
	Hash            string                         `json:"hash"`
	Memos           []map[string]map[string]string `json:"Memos"`
	TransactionType string                         `json:"TransactionType"`
	Sequence        uint64                         `json:"Sequence"`
	TicketSequence  uint64                         `json:"TicketSequence"`
	Account         string                         `json:"Account"`
	Amount          json.RawMessage                `json:"Amount"`
	MetaData        json.RawMessage                `json:"metaData"`
	DestinationTag  *uint32                        `json:"DestinationTag"`
}

type xrpAmount struct {
	Currency string `json:"currency"`
}

type xrpMeta struct {
	AffectedNodes []xrpAffectedNodes `json:"AffectedNodes"`
}

type xrpAffectedNodes struct {
	ModifiedNode xrpModifiedNode `json:"ModifiedNode"`
}

type xrpModifiedNode struct {
	FinalFields     xrpFields `json:"FinalFields"`
	PreviousFields  xrpFields `json:"PreviousFields"`
	LedgerEntryType string    `json:"LedgerEntryType"`
}

type xrpFields struct {
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
