package oidc

import (
	"context"

	"github.com/ory/fosite/handler/rfc8628"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	oauthDeviceUserCodePrefix       = "E"
	oauthDeviceUserCodeRandomLength = 7
)

type deviceStrategy struct {
	*rfc8628.DefaultDeviceStrategy
}

func (s *deviceStrategy) GenerateUserCode(ctx context.Context) (string, string, error) {
	userCode, err := utils.GenerateRandomUppercaseUnambiguousString(oauthDeviceUserCodeRandomLength)
	if err != nil {
		return "", "", err
	}

	userCode = oauthDeviceUserCodePrefix + userCode
	signature, err := s.UserCodeSignature(ctx, userCode)
	if err != nil {
		return "", "", err
	}

	return userCode, signature, nil
}
