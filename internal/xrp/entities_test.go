package xrp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistoryDropOrderBlocksFirst(t *testing.T) {
	order := (&Block{}).HistoryDropOrder()

	require.Len(t, order, 2)
	require.IsType(t, &Block{}, order[0])
	require.IsType(t, &Transaction{}, order[1])
}
