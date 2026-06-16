package dto

type CustomFieldValueDto struct {
	CustomFieldID string `json:"customFieldId"`
	Key           string `json:"key,omitempty"`
	Value         string `json:"value"`
}

type CustomFieldValueCreateDto struct {
	CustomFieldID string `json:"customFieldId" binding:"required_without=Key" unorm:"nfc"`
	Key           string `json:"key,omitempty" unorm:"nfc"`
	Value         string `json:"value" unorm:"nfc"`
}

type CustomFieldType string

const (
	CustomFieldTypeString  CustomFieldType = "string"
	CustomFieldTypeNumber  CustomFieldType = "number"
	CustomFieldTypeBoolean CustomFieldType = "boolean"
)

type CustomFieldTarget string

const (
	CustomFieldTargetUser  CustomFieldTarget = "user"
	CustomFieldTargetGroup CustomFieldTarget = "group"
	CustomFieldTargetBoth  CustomFieldTarget = "both"
)

type CustomFieldDto struct {
	ID                     string            `json:"id" binding:"required,uuid"`
	Key                    string            `json:"key" binding:"required" unorm:"nfc"`
	DisplayName            string            `json:"displayName" binding:"required" unorm:"nfc"`
	Type                   CustomFieldType   `json:"type" binding:"required,oneof=string number boolean"`
	Target                 CustomFieldTarget `json:"target" binding:"required,oneof=user group both"`
	Required               bool              `json:"required"`
	UserEditable           bool              `json:"userEditable"`
	DefaultValue           string            `json:"defaultValue" unorm:"nfc"`
	ValidationRegex        string            `json:"validationRegex" binding:"regex" unorm:"nfc"`
	ValidationErrorMessage string            `json:"validationErrorMessage" unorm:"nfc"`
}
