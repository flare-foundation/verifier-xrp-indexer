package xrp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionStatus(t *testing.T) {
	tests := []struct {
		name     string
		metaData string
		expected bool
	}{
		{
			name:     "success",
			metaData: `{"AffectedNodes":[],"TransactionResult":"tesSUCCESS"}`,
			expected: true,
		},
		{
			name:     "sender failure tecUNFUNDED_PAYMENT",
			metaData: `{"AffectedNodes":[],"TransactionResult":"tecUNFUNDED_PAYMENT"}`,
			expected: false,
		},
		{
			name:     "sender failure tecNO_DST",
			metaData: `{"AffectedNodes":[],"TransactionResult":"tecNO_DST"}`,
			expected: true,
		},
		{
			name:     "tecNO_PERMISSION without DomainID",
			metaData: `{"AffectedNodes":[],"TransactionResult":"tecNO_PERMISSION"}`,
			expected: true,
		},
		{
			name:     "unknown result",
			metaData: `{"AffectedNodes":[],"TransactionResult":"temBAD_AMOUNT"}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := transaction{MetaData: json.RawMessage(tt.metaData)}
			status, err := tx.transactionStatus()
			require.NoError(t, err)
			require.Equal(t, tt.expected, status)
		})
	}
}

// TestTransactionStatusFromRawTx verifies transactionStatus end-to-end on the raw
// JSON of testnet tx C935D183BC7D63C3D91AD8AE73564A42315E2EE0C5C9A4E0B1F67A28F517FE25:
// a Payment with DomainID that the ledger rejected with tecNO_PERMISSION. This also
// exercises the JSON binding of the DomainID field.
func TestTransactionStatusFromRawTx(t *testing.T) {
	raw := `{
		"Account": "rGqCqM1djMK19wgn4PwoEmGrpit8T4Akmt",
		"Amount": "1000000",
		"DeliverMax": "1000000",
		"Destination": "rUvJftL8ZkvLveY4b2E2rfkyiG2VPfDTFe",
		"DomainID": "87BC3EBBE3DB4AE7F40822037BCFF4FDED8B59F949594AC8BE4A0E40CF9DA226",
		"Fee": "12",
		"Flags": 0,
		"LastLedgerSequence": 17501605,
		"Sequence": 17501583,
		"SigningPubKey": "EDEE63EC44DC59A8C5CAA91268446E0BAAD96F8D2601CF329F3E572B294D60BC7F",
		"TransactionType": "Payment",
		"TxnSignature": "5CA4E52DAC85708B77529FB4857A944171A82BBA152F8DFECB90E99C542105560864DC20F8F1AA414144E96ACFA5DA6443E4FF962BD0D8FDDE1E887481F8AD06",
		"hash": "C935D183BC7D63C3D91AD8AE73564A42315E2EE0C5C9A4E0B1F67A28F517FE25",
		"metaData": {
			"AffectedNodes": [
				{
					"ModifiedNode": {
						"FinalFields": {
							"Account": "rGqCqM1djMK19wgn4PwoEmGrpit8T4Akmt",
							"Balance": "99999988",
							"Flags": 0,
							"OwnerCount": 0,
							"Sequence": 17501584
						},
						"LedgerEntryType": "AccountRoot",
						"LedgerIndex": "BC30C5B2B5C55A57E7EB170058627077E9EA175C194AF60A015B996DF4C564EF",
						"PreviousFields": {
							"Balance": "100000000",
							"Sequence": 17501583
						},
						"PreviousTxnID": "EFBB0CDC7F819909876D997F845C35DA193B5BA8BE6D82D3ED3BA09D0F66285D",
						"PreviousTxnLgrSeq": 17501583
					}
				}
			],
			"TransactionIndex": 1,
			"TransactionResult": "tecNO_PERMISSION"
		}
	}`

	var tx transaction
	require.NoError(t, json.Unmarshal([]byte(raw), &tx))
	require.NotNil(t, tx.DomainID)
	require.Equal(t, "87BC3EBBE3DB4AE7F40822037BCFF4FDED8B59F949594AC8BE4A0E40CF9DA226", *tx.DomainID)

	status, err := tx.transactionStatus()
	require.NoError(t, err)
	require.False(t, status)
}

// TestIsNativePaymentFromRawTx verifies isNativePayment on the raw JSON of real
// testnet transactions.
func TestIsNativePaymentFromRawTx(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected bool
	}{
		{
			// XRP-drops Amount, no SendMax.
			name: "native xrp to xrp",
			raw: `{
				"Account": "raVKHJGAo9YpeKgAHjN8XCHyfYwiyzirvY",
				"Amount": "100120",
				"DeliverMax": "100120",
				"Destination": "rJG3Xu7efRiJUfHX81D6vtoKjGKkeyp9XX",
				"DestinationTag": 1857044189,
				"Fee": "12",
				"Flags": 0,
				"LastLedgerSequence": 17870400,
				"Sequence": 17445749,
				"TransactionType": "Payment",
				"hash": "585310057EE0F00A926EC11A81E7F6C8FC6CD058ADB18594D85083EF4AA6F562"
			}`,
			expected: true,
		},
		{
			// XRP-drops Amount but SendMax in USD: cross-currency, not native.
			name: "cross-currency with SendMax",
			raw: `{
				"Account": "rBBUxL9d68WF6pxEczcfNCY4fNqtDPC8be",
				"Amount": "1000000",
				"DeliverMax": "1000000",
				"Destination": "rLe7NgT7GhbLGJ7RUfrH7zQezutWyNU3Md",
				"DomainID": "FC052BBC0585BFEE8730FFFB31D9E9888A5D5CE15242C4522C690B1317C90A2B",
				"Fee": "12",
				"Flags": 0,
				"LastLedgerSequence": 17281396,
				"SendMax": {"currency": "USD", "issuer": "rQDqYo6X5gNJjHHGfbWQYcBZVFNcySXiC9", "value": "5"},
				"Sequence": 17281363,
				"TransactionType": "Payment",
				"hash": "CB7E1D062D8E08650B54E4C904ED3344B7CC52EB8CF6CF053AC265BBD2D4CA66"
			}`,
			expected: false,
		},
		{
			// IOU (USD token) Amount object, currency != XRP.
			name: "iou transfer",
			raw: `{
				"Account": "rGbzc1r3QZHeU7KyEDVAYvzX4ckBD7Shoy",
				"Amount": {"currency": "USD", "issuer": "rGbzc1r3QZHeU7KyEDVAYvzX4ckBD7Shoy", "value": "10"},
				"Destination": "rsHyEm3apWSEF1fLg7XJe7jZxpdzXevpqk",
				"Fee": "12",
				"Flags": 0,
				"LastLedgerSequence": 17870538,
				"Sequence": 17870514,
				"TransactionType": "Payment",
				"hash": "FBC93B0F9721D255A19BD113ADC0B4A446987CBF1818990C78E981AF81A41967"
			}`,
			expected: false,
		},
		{
			// MPT Amount object: no currency field, so not XRP.
			name: "mpt transfer",
			raw: `{
				"Account": "rGbzc1r3QZHeU7KyEDVAYvzX4ckBD7Shoy",
				"Amount": {"mpt_issuance_id": "0110AEB3AB2820761E65E8052FE81907B390764AA5BB3B75", "value": "10"},
				"Destination": "rsHyEm3apWSEF1fLg7XJe7jZxpdzXevpqk",
				"Fee": "12",
				"Flags": 0,
				"LastLedgerSequence": 17870544,
				"Sequence": 17870516,
				"TransactionType": "Payment",
				"hash": "855144D6960720E9582981DB0E8A9CE55161531C7A5FD8A9EF3B84B49CC8F12F"
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tx transaction
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &tx))

			native, err := tx.isNativePayment()
			require.NoError(t, err)
			require.Equal(t, tt.expected, native)
		})
	}
}

func TestIntendedReceivingAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		expected *uint64
	}{
		{
			name:     "native XRP drops",
			amount:   `"15000000"`,
			expected: new(uint64(15000000)),
		},
		{
			name:     "large native amount",
			amount:   `"100000000000000000"`,
			expected: new(uint64(100000000000000000)),
		},
		{
			name:     "token amount object",
			amount:   `{"currency":"USD","issuer":"rf1BiGeXwwQoi8Z2ueFYTEXSwuJYfV2Jpn","value":"100"}`,
			expected: nil,
		},
		{
			name:     "empty amount",
			amount:   ``,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := transaction{Amount: json.RawMessage(tt.amount)}
			result, err := tx.intendedReceivingAmount()
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSourceAddressesRoot(t *testing.T) {
	tests := []struct {
		name     string
		metaData string
		expected string
	}{
		{
			// Real mainnet tx 72F485775B8BE0E5E0F398CB29BAD007EE9E29E450649778FCED98E735BFAE44
			// Payment that creates (funds) a new account.
			// Sender rnJrjec2vrTJAAQUTMTjj7U6xdXrk9N4mT balance decreased (ModifiedNode).
			// Receiver r3iE6gLEVq6XXN8ddPWxffcYKJLxG2tC51 created with balance 15000000 (CreatedNode).
			name: "fund account",
			metaData: `{
				"AffectedNodes": [
					{
						"ModifiedNode": {
							"FinalFields": {
								"Account": "rnJrjec2vrTJAAQUTMTjj7U6xdXrk9N4mT",
								"Balance": "38695141402626",
								"Flags": 0,
								"OwnerCount": 0,
								"Sequence": 99628112
							},
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "32CC7D5F026163A313EF59E0A610D136F9D66DFE6B5F7C926C68790C384F1C4C",
							"PreviousFields": {
								"Balance": "38695156408626",
								"Sequence": 99628111
							},
							"PreviousTxnID": "17C7E445DA7EFE3F5FCF67AE6270E29AD4FF147BFB6C3527F370EAEB9DC00C47",
							"PreviousTxnLgrSeq": 103489700
						}
					},
					{
						"CreatedNode": {
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "71BED8C87259463266008FE80FAF3A0B89EB3577C7DD474173A1C8F6142572C2",
							"NewFields": {
								"Account": "r3iE6gLEVq6XXN8ddPWxffcYKJLxG2tC51",
								"Balance": "15000000",
								"Sequence": 103489700
							}
						}
					}
				],
				"TransactionIndex": 57,
				"TransactionResult": "tesSUCCESS"
			}`,
			expected: "e9f8d46cc405736e946e2e299ae9106ef494b7454f802028be9fe665398e69e3",
		},
		{
			// Real mainnet tx 3C2B0D8C1938271D9A95E657DD7B3D38FE77E84338AB63DFFF5A6079D944EFD0
			// AccountDelete that removes rGbHCiVTbyFH9gJ4mJCPXgLgjpgnTjTeko.
			// Destination rwn3XsPQcJcEASeT4fMUditHPrFnUDU83t balance increased (ModifiedNode).
			// Deleted account rGbHCiVTbyFH9gJ4mJCPXgLgjpgnTjTeko balance 1999980 -> 0 (DeletedNode).
			name: "account delete",
			metaData: `{
				"AffectedNodes": [
					{
						"ModifiedNode": {
							"FinalFields": {
								"Account": "rwn3XsPQcJcEASeT4fMUditHPrFnUDU83t",
								"Balance": "503599950",
								"Flags": 0,
								"OwnerCount": 23,
								"Sequence": 92601466
							},
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "980E43BF431AC9B314AB1BB76A1E038E88B35051214FA2282ADB78D9913E4ABE",
							"PreviousFields": {
								"Balance": "501799970"
							},
							"PreviousTxnID": "F8AD2B7752FEE1E9877AE65CDE5446C8DEB2B2CD51A177050D446C9CDAFB58EC",
							"PreviousTxnLgrSeq": 103489665
						}
					},
					{
						"DeletedNode": {
							"FinalFields": {
								"Account": "rGbHCiVTbyFH9gJ4mJCPXgLgjpgnTjTeko",
								"Balance": "0",
								"Flags": 536870912,
								"OwnerCount": 0,
								"PreviousTxnID": "66FDFD95D19BD9414DED96C1B7884839121D55AD1EDF116228AF4EC556C687D5",
								"PreviousTxnLgrSeq": 103488743,
								"Sequence": 103488586
							},
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "9EBD91EE34022E785E3679CA1116D18353484D9CCAB63EAC878048FDB1650BE5",
							"PreviousFields": {
								"Balance": "1999980",
								"Sequence": 103488585
							}
						}
					}
				],
				"TransactionIndex": 56,
				"TransactionResult": "tesSUCCESS"
			}`,
			expected: "97f123575f0fa54abec71fbc3740802261b15252e51f0051b11e39dec51c5602",
		},
		{
			// Real mainnet tx C94C88F70799DE3DEA6C5ECD65729EBAA5A6DC086D38E4BD40CA236A508BF056
			// Regular payment between two existing accounts.
			// Sender rpKrHjNFVPLbnvQj7WZiawLc6QKkeyMsau balance decreased (ModifiedNode).
			// Receiver ryouhapPYV5KNHmFUKrjNqsjxhnxvQiVt balance increased (ModifiedNode).
			name: "regular payment",
			metaData: `{
				"AffectedNodes": [
					{
						"ModifiedNode": {
							"FinalFields": {
								"Account": "ryouhapPYV5KNHmFUKrjNqsjxhnxvQiVt",
								"Balance": "2532322665",
								"Domain": "78616D616E2E617070",
								"EmailHash": "5B0F31F3678A837376CD437DAA2FCC77",
								"Flags": 1048576,
								"OwnerCount": 81,
								"RegularKey": "r33MBTjz1hzaZSsyvRwHRmaMn9S5xpp9w6",
								"Sequence": 80617032,
								"TicketCount": 50
							},
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "9185B96A67C38FC7BBA39BC44BB6CC9C19DBAE20778A2E3BBB0EBB60BE5F7E0E",
							"PreviousFields": {
								"Balance": "2532272615"
							},
							"PreviousTxnID": "9B265D9B1849D197391D0AB667248C1BE7FF531B2C15F5ECFEEF7C2793CB4CD2",
							"PreviousTxnLgrSeq": 103490053
						}
					},
					{
						"ModifiedNode": {
							"FinalFields": {
								"Account": "rpKrHjNFVPLbnvQj7WZiawLc6QKkeyMsau",
								"Balance": "70977048",
								"Flags": 0,
								"OwnerCount": 4,
								"Sequence": 102610436
							},
							"LedgerEntryType": "AccountRoot",
							"LedgerIndex": "961A230EEDA5BFF71046208002265EF37AD05F481EA1A210DCC0C86713106E1C",
							"PreviousFields": {
								"Balance": "71027110",
								"Sequence": 102610435
							},
							"PreviousTxnID": "DA6253D7825FA8C3E966624B45422B093C82006CA9C5057C179D9DFE68694884",
							"PreviousTxnLgrSeq": 103490056
						}
					}
				],
				"TransactionIndex": 0,
				"TransactionResult": "tesSUCCESS"
			}`,
			expected: "2f0cdffefa7f270919a31d1898cb2129eaa255ac75ed22d89155466f93254c1b",
		},
		{
			name: "no affected nodes",
			metaData: `{
				"AffectedNodes": [],
				"TransactionResult": "tesSUCCESS"
			}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := transaction{MetaData: json.RawMessage(tt.metaData)}
			result, err := tx.sourceAddressesRoot()

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDestinationAddressHash(t *testing.T) {
	tests := []struct {
		address  string
		expected *string
	}{
		{
			address:  "ryouhapPYV5KNHmFUKrjNqsjxhnxvQiVt",
			expected: new("8f40c24f58fb0abfe1312837a2345d2294762ca46994acc99ff13ccba01fe1b2"),
		},
		{
			address:  "r3iE6gLEVq6XXN8ddPWxffcYKJLxG2tC51",
			expected: new("32ba9ed8519cb562a6f845df24258d3d093fab95c049ebe45a25662c226dfb43"),
		},
		{
			address:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		tx := transaction{Destination: tt.address}
		require.Equal(t, tt.expected, tx.destinationAddressHash())
	}
}
