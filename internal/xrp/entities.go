package xrp

type Block struct {
	Hash         string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber  uint64 `gorm:"index"`
	Timestamp    uint64 `gorm:"index"`
	Transactions uint64
}

type Transaction struct {
	Hash                string `gorm:"primaryKey;type:varchar(64)"`
	BlockNumber         uint64 `gorm:"index"`
	Timestamp           uint64 `gorm:"index"`
	PaymentReference    string `gorm:"index;type:varchar(64);default:null"`
	Response            string `gorm:"type:varchar"`
	IsNativePayment     bool   `gorm:"index"`
	SourceAddressesRoot string `gorm:"index;type:varchar(64);default:null"`
}

func (b Block) GetBlockNumber() uint64 {
	return b.BlockNumber
}

func (b Block) GetTimestamp() uint64 {
	return b.Timestamp
}
