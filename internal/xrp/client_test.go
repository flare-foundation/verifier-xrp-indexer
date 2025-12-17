// Helper functions for communicating with local rippled node via JSON-RPC for testing purposes.

package xrp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type RPCRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params,omitempty"`
}

type RPCResponse struct {
	Result json.RawMessage `json:"result"`
}

func callRPC(method string, params ...any) (json.RawMessage, error) {
	reqBody, err := json.Marshal(RPCRequest{
		Method: method,
		Params: params,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		"http://localhost:5005",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	return rpcResp.Result, nil
}

type Wallet struct {
	Account string `json:"account_id"`
	Seed    string `json:"master_seed"`
}

func createWallet(t *testing.T) Wallet {
	raw, err := callRPC("wallet_propose")
	require.NoError(t, err)

	var w Wallet
	require.NoError(t, json.Unmarshal(raw, &w))
	return w
}

const (
	genesisAccount = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"
	genesisSecret  = "snoPBrXtMeMyMHUVTgbuqAfg1SUTb"
)

type statusResponse struct {
	Status string `json:"status"`
}

func requireSuccess(t *testing.T, rsp json.RawMessage) {
	var status statusResponse
	require.NoError(t, json.Unmarshal(rsp, &status))
	require.Equal(t, "success", status.Status)
}

func fundAccount(t *testing.T, dest string, drops string) {
	rsp, err := callRPC("submit", map[string]any{
		"tx_json": map[string]any{
			"TransactionType": "Payment",
			"Account":         genesisAccount,
			"Destination":     dest,
			"Amount":          drops,
		},
		"secret": genesisSecret,
	})

	require.NoError(t, err)
	requireSuccess(t, rsp)
}

func submitPayment(t *testing.T, from Wallet, to string, drops string) {
	rsp, err := callRPC("submit", map[string]any{
		"tx_json": map[string]any{
			"TransactionType": "Payment",
			"Account":         from.Account,
			"Destination":     to,
			"Amount":          drops,
		},
		"secret": from.Seed,
	})

	require.NoError(t, err)
	requireSuccess(t, rsp)
}

type currentLedgerResponse struct {
	LedgerIndex uint64 `json:"ledger_index"`
}

func getValidatedLedgerIndex(t *testing.T) uint64 {
	params := map[string]any{
		"ledger_index": "validated",
	}
	raw, err := callRPC("ledger", params)
	require.NoError(t, err)

	var ledger currentLedgerResponse
	err = json.Unmarshal(raw, &ledger)
	require.NoError(t, err)

	return ledger.LedgerIndex
}

func awaitReady(t *testing.T) {
	require.Eventually(
		t,
		func() bool {
			raw, err := callRPC("server_info")
			if err != nil {
				return false
			}

			var status statusResponse
			if err := json.Unmarshal(raw, &status); err != nil {
				return false
			}

			if status.Status != "success" {
				return false
			}

			return true
		},
		30*time.Second, // Wait for up to 30 seconds
		time.Second,    // Check every second
	)
}
