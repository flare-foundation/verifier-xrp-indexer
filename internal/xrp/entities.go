package xrp

import (
	"github.com/flare-foundation/verifier-indexer-framework/pkg/database"
)

// Transaction is the database model for an XRP ledger payment transaction.
type Transaction struct {
	Hash        string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber uint64 `gorm:"index;index:idx_payref_block,priority:2;index:idx_dest_block,priority:2"`
	Timestamp   uint64 `gorm:"index"`
	// TransactionStatus is true when the transaction result is tesSUCCESS or a
	// destination-related failure (tecDST_TAG_NEEDED, tecNO_DST, tecNO_DST_INSUF_XRP, and
	// tecNO_PERMISSION without defined DomainID), and false for all other results.
	TransactionStatus bool   `gorm:"index"`
	PaymentReference  string `gorm:"index:idx_payref_block,priority:1;type:varchar(64);default:null"`
	Response          string `gorm:"type:varchar"`
	IsNativePayment   bool   `gorm:"index"`
	// IntendedReceivingAmount is the intended drops of a native XRP payment,
	// zero for a non-native payment (see client.go) and null for a non-payment.
	IntendedReceivingAmount *uint64 `gorm:"index:idx_payref_block,priority:3;index:idx_dest_block,priority:3"`
	DestinationAddressHash  *string `gorm:"index:idx_dest_block,priority:1;type:varchar(64);default:null"`
	SourceAddressesRoot     string  `gorm:"index;type:varchar(64);default:null"`
	Sequence                uint64  `gorm:"index:idx_source_sequence,priority:2"`
	TicketSequence          uint64
	SourceAddress           string  `gorm:"index:idx_source_sequence,priority:1;type:varchar(64)"`
	FirstMemoDataHash       string  `gorm:"index;type:varchar(64);default:null"`
	DestinationTag          *uint32 `gorm:"index;default:null"`
}

// TimestampField returns the database column name used for timestamp-based queries.
func (t *Transaction) TimestampField() string {
	return "timestamp"
}

var _ database.Block = new(Block)

// Block is the database model for an XRP ledger block.
type Block struct {
	Hash         string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber  uint64 `gorm:"index"`
	Timestamp    uint64 `gorm:"index"`
	Transactions uint64
}

// GetBlockNumber returns the ledger index of the block.
func (b *Block) GetBlockNumber() uint64 {
	return b.BlockNumber
}

// GetTimestamp returns the Unix close time of the block.
func (b *Block) GetTimestamp() uint64 {
	return b.Timestamp
}

// TimestampField returns the database column name used for timestamp-based queries.
func (b *Block) TimestampField() string {
	return "timestamp"
}

// HistoryDropOrder returns the tables to prune when dropping historical data.
func (b *Block) HistoryDropOrder() []database.Deletable {
	var block Block
	var tx Transaction

	// The order does not particularly matter since there are no foreign key constraints.
	return []database.Deletable{
		&tx,
		&block,
	}
}
