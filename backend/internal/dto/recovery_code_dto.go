package dto

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type RecoveryCodeDto struct {
	ID        string             `json:"id"`
	CreatedAt datatype.DateTime  `json:"createdAt"`
	UsedAt    *datatype.DateTime `json:"usedAt"`
}

type RecoveryCodeStatusDto struct {
	Total  int `json:"total"`
	Unused int `json:"unused"`
}

type RecoveryCodeGenerateResponseDto struct {
	Codes []string `json:"codes"`
}
