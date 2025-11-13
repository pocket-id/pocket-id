package model

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type Storage struct {
	Path      string `gorm:"primaryKey;not null"`
	Data      []byte
	Size      int64
	ModTime   datatype.DateTime
	CreatedAt datatype.DateTime
}

func (Storage) TableName() string {
	return "storage"
}
