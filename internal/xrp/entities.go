package xrp

import "encoding/json"

type Block struct {
	Hash      string `gorm:"primaryKey;type:varchar(64)"`
	Number    uint64 `gorm:"index"`
	Timestamp uint64 `gorm:"index"`
}

type Transaction struct {
	Hash      string `gorm:"primaryKey;type:varchar(64)"`
	BlockHash string `gorm:"type:varchar(64)"`
	Block     *Block
	Memos     []json.RawMessage `gorm:"type:jsonb"`
}
