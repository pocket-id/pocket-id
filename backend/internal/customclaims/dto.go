package customclaims

type CustomClaimCreateDto struct {
	Key   string `json:"key" binding:"required" unorm:"nfc"`
	Value string `json:"value" binding:"required" unorm:"nfc"`
}
