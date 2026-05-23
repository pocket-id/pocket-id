package model

type CustomFieldValue struct {
	Base

	CustomFieldID string
	Value         string

	UserID      *string
	UserGroupID *string
}
