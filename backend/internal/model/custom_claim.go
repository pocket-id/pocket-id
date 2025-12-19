package model

type CustomClaim struct {
	Base

	Key    string
	Value  string
	IsLdap bool

	UserID      *string
	UserGroupID *string
}
