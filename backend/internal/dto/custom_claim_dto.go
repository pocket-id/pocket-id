package dto

type CustomClaimDto struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CustomClaimCreateDto struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}
