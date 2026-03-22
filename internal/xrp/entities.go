package xrp

import "github.com/flare-foundation/verifier-indexer-framework/pkg/database"

type Block struct {
	Hash         string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber  uint64 `gorm:"index"`
	Timestamp    uint64 `gorm:"index"`
	Transactions uint64
}

type Transaction struct {
	Hash                string  `gorm:"primaryKey;type:varchar(64);index:idx_block_hash_composite,priority:2,option:CONCURRENTLY"`
	BlockNumber         uint64  `gorm:"index:idx_block_hash_composite,priority:1,option:CONCURRENTLY"`
	Timestamp           uint64  `gorm:"index"`
	PaymentReference    string  `gorm:"index;type:varchar(64);default:null"`
	Response            string  `gorm:"type:varchar"`
	IsNativePayment     bool    `gorm:"index"`
	SourceAddressesRoot string  `gorm:"index;type:varchar(64);default:null"`
	FirstMemoDataHash   string  `gorm:"index;type:varchar(64);default:null"`
	DestinationTag      *uint32 `gorm:"index;default:null"`
}

func (b Block) GetBlockNumber() uint64 {
	return b.BlockNumber
}

func (b Block) GetTimestamp() uint64 {
	return b.Timestamp
}

func (b Block) TimestampField() string {
	return "timestamp"
}

func (b Block) HistoryDropOrder() []database.Deletable {
	var block Block
	var tx Transaction

	// The order does not particularly matter since there are no foreign key constraints.
	return []database.Deletable{
		&tx,
		&block,
	}
}

func (t Transaction) TimestampField() string {
	return "timestamp"
}
