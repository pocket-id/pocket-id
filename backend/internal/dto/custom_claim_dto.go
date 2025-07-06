package dto

import (
	"golang.org/x/text/unicode/norm"
)

type CustomClaimDto struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CustomClaimCreateDto struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

func (c *CustomClaimCreateDto) Normalize() {
	c.Key = norm.NFC.String(c.Key)
	c.Value = norm.NFC.String(c.Value)
}
