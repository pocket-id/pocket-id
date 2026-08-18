package scimsync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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

	// Verify SQLite accepts the read-only snapshot used by synchronization
	snapshot, err := service.loadSyncSnapshot(t.Context(), provider.ID)
	require.NoError(t, err)
	require.Equal(t, provider.ID, snapshot.provider.ID)
}

func TestSyncSnapshotTxOptions(t *testing.T) {
	require.Equal(t, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, syncSnapshotTxOptions("postgres"))
	require.Equal(t, &sql.TxOptions{ReadOnly: true}, syncSnapshotTxOptions("sqlite"))
}

func TestSyncServiceProvidersLimitsConcurrencyAndJoinsErrors(t *testing.T) {
	providers := make([]ServiceProvider, 8)
	for i := range providers {
		providers[i].ID = fmt.Sprintf("provider-%d", i)
	}

	started := make(chan string, len(providers))
	release := make(chan struct{})
	done := make(chan error, 1)
	var active atomic.Int32
	var maximum atomic.Int32

	go func() {
		done <- syncServiceProviders(t.Context(), providers, func(_ context.Context, providerID string) error {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			started <- providerID
			<-release
			active.Add(-1)

			if providerID == "provider-0" || providerID == "provider-7" {
				return errors.New(providerID + " failed")
			}
			return nil
		})
	}()

	// Keep the first batch blocked so a fifth provider would expose a broken concurrency limit
	firstBatch := make([]string, 0, syncProviderConcurrency)

firstBatchLoop:
	for range syncProviderConcurrency {
		select {
		case providerID := <-started:
			firstBatch = append(firstBatch, providerID)
		case <-time.After(2 * time.Second):
			break firstBatchLoop
		}
	}

	var unexpectedProvider string
	select {
	case unexpectedProvider = <-started:
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	err := <-done
	require.Len(t, firstBatch, syncProviderConcurrency)
	require.Empty(t, unexpectedProvider)
	require.EqualValues(t, syncProviderConcurrency, maximum.Load())
	require.ErrorContains(t, err, "provider-0 failed")
	require.ErrorContains(t, err, "provider-7 failed")
}
