package model

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type Storage struct {
	Path      string            `gorm:"primaryKey;not null"`
	Data      []byte            `gorm:"not null"`
	Size      int64             `gorm:"not null"`
	ModTime   datatype.DateTime `gorm:"not null"`
	CreatedAt datatype.DateTime `gorm:"not null"`
}

func (Storage) TableName() string {
	return "storage"
}
