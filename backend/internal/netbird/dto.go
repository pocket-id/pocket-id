package netbird

type identityDto struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type loginDto struct {
	UserID string `json:"userId" binding:"required"`
}
