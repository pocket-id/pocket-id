package model

type KV struct {
	Key   string `gorm:"primaryKey;not null"`
	Value *string
}

// TableName overrides the table name used by KV to `kv`
func (KV) TableName() string {
	return "kv"
}
