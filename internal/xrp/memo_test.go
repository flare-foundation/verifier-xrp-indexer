package xrp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstMemoDataHash(t *testing.T) {
	tests := []struct {
		name     string
		memoData string
		expected string
	}{
		{
			name:     "long memo data",
			memoData: "5468697320746F6B656E2064726F70206973206120726573756C74206F6620796F75722041534320686F6C64696E67732070726F706F7274696F6E61746520746F20746F74616C20737570706C792C2063616C63756C617465642061732070726F6365656473206F6620343525206F6620616E6E75616C2031322C3030302C3030302041534320737570706C7920696E63726561736520646973747269627574656420746F20616C6C2041534320686F6C646572732077686572652044726F70733E46656573206F6E20616E20686F75726C792C206461696C792C207765656B6C792C206F72206D6F6E74686C792062617369732E20466F72206D6F726520696E666F726D6174696F6E2076697369742068747470733A2F2F7777772E617363656E73696F6E696E6465782E636F6D2F",
			expected: "fe42159c2a27cf1451f890cec1aa0a07c8465da151794ec6a5a1294a261655d3",
		},
		{
			name:     "short memo data",
			memoData: "3639303835373338",
			expected: "e8a4c3409f0f9db91fb53de91eb47e02e58f8f1062b7f1930a4defbf6d66e017",
		},
		{
			name:     "binary-like memo data",
			memoData: "46425052664100020000000000000000000000000000000000000000004D5760",
			expected: "7513e735152cafd288c9247151d760f1ac038d318d6edadbe12c8cd0025739a6",
		},
		{
			name:     "no memos",
			memoData: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tx transaction
			if tt.memoData != "" {
				tx.Memos = []map[string]map[string]string{
					{"Memo": {"MemoData": tt.memoData}},
				}
			}
			result, err := tx.firstMemoDataHash()
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}
