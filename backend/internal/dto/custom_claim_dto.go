package dto

type CustomClaimDto struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CustomClaimCreateDto struct {
	Key   string `json:"key" required:"true" unorm:"nfc"`
	Value string `json:"value" required:"true" unorm:"nfc"`
}
