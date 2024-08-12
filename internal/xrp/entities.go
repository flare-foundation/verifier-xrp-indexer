package xrp

type Block struct {
	Hash      string `gorm:"primaryKey;type:varchar(64)"`
	Number    uint64 `gorm:"index"`
	Timestamp uint64 `gorm:"index"`
}

type Transaction struct {
	Hash      string `gorm:"primaryKey;type:varchar(64)"`
	Timestamp uint64 `gorm:"index"`
	BlockHash string `gorm:"type:varchar(64)"`
	Block     *Block
}
