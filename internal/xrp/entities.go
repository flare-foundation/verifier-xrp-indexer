package xrp

import "github.com/flare-foundation/verifier-indexer-framework/pkg/database"

type Block struct {
	Hash         string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber  uint64 `gorm:"index"`
	Timestamp    uint64 `gorm:"index"`
	Transactions uint64
}

func (b Block) GetBlockNumber() uint64 {
	return b.BlockNumber
}

func (b Block) GetTimestamp() uint64 {
	return b.Timestamp
}

func (b Block) TimestampQuery() string {
	return "timestamp < ?"
}

func (b Block) HistoryDropOrder() []database.Deletable {
	// It does not particularly matter the order as there are no foreign key constraints.
	// However we drop transactions first to avoid leaving orphaned records in case of partial failures.
	return []database.Deletable{
		Transaction{},
		Block{},
	}
}

type Transaction struct {
	Hash                string `gorm:"primaryKey;type:varchar(64);index:idx_block_hash_composite,priority:2,option:CONCURRENTLY"`
	BlockNumber         uint64 `gorm:"index:idx_block_hash_composite,priority:1,option:CONCURRENTLY"`
	Timestamp           uint64 `gorm:"index"`
	PaymentReference    string `gorm:"index;type:varchar(64);default:null"`
	Response            string `gorm:"type:varchar"`
	IsNativePayment     bool   `gorm:"index"`
	SourceAddressesRoot string `gorm:"index;type:varchar(64);default:null"`
}

func (t Transaction) TimestampQuery() string {
	return "timestamp < ?"
}
