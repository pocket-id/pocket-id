package service

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

func TestScimServiceProviderOperationsReturnSpecificNotFoundErrors(t *testing.T) {
	service := NewScimService(testutils.NewDatabaseForTest(t), nil, nil)

	_, err := service.CreateServiceProvider(t.Context(), &dto.ScimServiceProviderCreateDTO{
		Endpoint:     "https://scim.example.com",
		OidcClientID: "missing-client",
	})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	_, err = service.GetServiceProvider(t.Context(), "missing-provider")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	err = service.DeleteServiceProvider(t.Context(), "missing-provider")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}
