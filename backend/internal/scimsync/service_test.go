package scimsync

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

func TestServiceProviderOperationsReturnSpecificNotFoundErrors(t *testing.T) {
	service := newService(testutils.NewDatabaseForTest(t), nil)

	_, err := service.CreateServiceProvider(t.Context(), &ScimServiceProviderCreateDTO{
		Endpoint:     "https://scim.example.com",
		OidcClientID: "missing-client",
	})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	_, err = service.GetServiceProvider(t.Context(), "missing-provider")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	err = service.DeleteServiceProvider(t.Context(), "missing-provider")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func TestServiceProviderCreateAndUpdate(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db, nil)

	// Create two clients so provider creation and reassignment both satisfy the foreign key
	require.NoError(t, db.Create(&[]model.OidcClient{
		{Base: model.Base{ID: "client-1"}, Name: "Client 1"},
		{Base: model.Base{ID: "client-2"}, Name: "Client 2"},
	}).Error)

	// Create the provider with its initial client in one transaction
	provider, err := service.CreateServiceProvider(t.Context(), &ScimServiceProviderCreateDTO{
		Endpoint:     "https://scim.example.com/v1",
		OidcClientID: "client-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, provider.ID)

	// Move the provider to the second client in one transaction
	provider, err = service.UpdateServiceProvider(t.Context(), provider.ID, &ScimServiceProviderCreateDTO{
		Endpoint:     "https://scim.example.com/v2",
		OidcClientID: "client-2",
	})
	require.NoError(t, err)
	require.Equal(t, "https://scim.example.com/v2", provider.Endpoint)
	require.Equal(t, "client-2", provider.OidcClientID)

	// Verify the committed provider retains both updated values
	persisted, err := service.GetServiceProvider(t.Context(), provider.ID)
	require.NoError(t, err)
	require.Equal(t, provider.Endpoint, persisted.Endpoint)
	require.Equal(t, provider.OidcClientID, persisted.OidcClientID)
}
