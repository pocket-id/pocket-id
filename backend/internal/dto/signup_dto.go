package dto

type SignUpDto struct {
	Username  string  `json:"username" binding:"required,username,min=2,max=50" unorm:"nfc"`
	Email     *string `json:"email" binding:"omitempty,email" unorm:"nfc"`
	FirstName string  `json:"firstName" binding:"required,min=1,max=50" unorm:"nfc"`
	LastName  string  `json:"lastName" binding:"max=50" unorm:"nfc"`
	Token     string  `json:"token"`
}
