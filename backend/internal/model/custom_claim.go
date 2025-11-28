package model

type CustomClaim struct {
	Base

	Key   string
	Value string

	UserID      *string
	UserGroupID *string
}
