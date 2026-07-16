package devicelogin

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type requestCreateDto struct {
	ID                      string            `json:"id"`
	UserCode                string            `json:"userCode"`
	VerificationURI         string            `json:"verificationUri"`
	VerificationURIComplete string            `json:"verificationUriComplete"`
	ExpiresAt               datatype.DateTime `json:"expiresAt"`
	Interval                int               `json:"interval"`
}

type verificationDto struct {
	Code string `json:"code" binding:"required"`
}

type decisionDto struct {
	Code     string `json:"code" binding:"required"`
	Decision string `json:"decision" binding:"required,oneof=approve deny"`
}

type verificationInfoDto struct {
	UserCode  string            `json:"userCode"`
	Device    string            `json:"device"`
	IPAddress string            `json:"ipAddress"`
	ExpiresAt datatype.DateTime `json:"expiresAt"`
}
