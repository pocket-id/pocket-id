//go:build unit

package claimmappingpolicy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestSaveCustomScope(t *testing.T) {
	service := newTestService(t)
	kuber, err := service.CreateCustomScope(t.Context(), CustomScopeDto{Key: "kuber", Name: "Kubernetes role", Description: "Your Kubernetes role"})
	require.NoError(t, err)

	t.Run("the key can't change", func(t *testing.T) {
		_, err := service.UpdateCustomScope(t.Context(), kuber.ID, dto.Replace(CustomScopeDto{Key: "kube", Name: "Kubernetes role", Description: "Your Kubernetes role"}))
		require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed), "got %v", err)
	})

	t.Run("reserved keys are rejected", func(t *testing.T) {
		require.NoError(t, service.db.Exec(`INSERT INTO apis (id, created_at, name, audience) VALUES ('11111111-1111-1111-1111-111111111111', CURRENT_TIMESTAMP, 'Orders', 'https://orders.example.com')`).Error)
		require.NoError(t, service.db.Exec(`INSERT INTO api_permissions (id, created_at, api_id, key, name) VALUES ('22222222-2222-2222-2222-222222222222', CURRENT_TIMESTAMP, '11111111-1111-1111-1111-111111111111', 'read:orders', 'Read')`).Error)

		for _, key := range []string{"Profile", "users:read", "read:orders"} {
			_, err := service.CreateCustomScope(t.Context(), CustomScopeDto{Key: key, Name: "Name", Description: "Description"})
			require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed), "key %q must be rejected, got %v", key, err)
		}
	})
}

func TestClientPolicy(t *testing.T) {
	service := newTestService(t)
	require.NoError(t, Bootstrap(t.Context(), service.db))
	custom, err := service.CreateClaimMappingPolicy(t.Context(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Custom"},
	})
	require.NoError(t, err)
	require.NoError(t, service.db.Create([]model.OidcClient{
		{Base: model.Base{ID: "client-default"}, Name: "Default Client"},
		{Base: model.Base{ID: "client-custom"}, Name: "Custom Client", ClaimMappingPolicyId: &custom.ID},
	}).Error)

	tests := []struct {
		clientID string
		want     string
	}{
		{"client-default", BuiltInClaimMappingPolicyID},
		{"client-custom", custom.ID},
		{"missing-client", BuiltInClaimMappingPolicyID},
	}
	for _, tt := range tests {
		t.Run(tt.clientID, func(t *testing.T) {
			policy, err := ClientPolicy(t.Context(), service.db, tt.clientID)
			require.NoError(t, err)
			require.Equal(t, tt.want, policy.ID)
		})
	}
}

func TestClientCustomScopeKeysFollowThePolicy(t *testing.T) {
	service := newTestService(t)
	require.NoError(t, Bootstrap(t.Context(), service.db))
	for _, key := range []string{"kuber", "caddy"} {
		_, err := service.CreateCustomScope(t.Context(), CustomScopeDto{Key: key, Name: key, Description: key})
		require.NoError(t, err)
	}

	customClaim := func(key string, scopes ...string) OidcClaimMappingDto {
		return OidcClaimMappingDto{ClaimName: key, SourceType: string(MappingSourceCustomClaim), SourceValue: key, Scope: scopes, IDToken: true}
	}
	policy, err := service.CreateClaimMappingPolicy(t.Context(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Kubernetes"},
		ClaimMappings: []OidcClaimMappingDto{
			customClaim("k8s_role", "kuber", "profile"),
			customClaim("k8s_groups", "kuber"),
			customClaim("caddyfile", "caddy"),
		},
	})
	require.NoError(t, err)
	// Deletes caddy behind the policy's back, like a delete racing a policy save that still listed it
	require.NoError(t, service.db.Delete(&CustomScope{}, "key = ?", "caddy").Error)

	require.NoError(t, service.db.Create([]model.OidcClient{
		{Base: model.Base{ID: "k3s"}, Name: "K3S", ClaimMappingPolicyId: &policy.ID},
		{Base: model.Base{ID: "nextcloud"}, Name: "Nextcloud"},
		{Base: model.Base{ID: "https://app.example.com/client.json"}, Name: "CIMD", ClientType: model.OidcClientTypeCIMD, ClaimMappingPolicyId: &policy.ID},
	}).Error)

	tests := []struct {
		clientID string
		want     []string
	}{
		{"k3s", []string{"kuber"}},
		{"nextcloud", nil},
		{"https://app.example.com/client.json", nil},
	}
	for _, tt := range tests {
		t.Run(tt.clientID, func(t *testing.T) {
			var client model.OidcClient
			require.NoError(t, service.db.First(&client, "id = ?", tt.clientID).Error)
			keys, err := ClientCustomScopeKeys(t.Context(), service.db, client)
			require.NoError(t, err)
			require.Equal(t, tt.want, keys)
		})
	}
}

func TestClaimMappingsReleasedWithCustomScope(t *testing.T) {
	service := newTestService(t)
	kuber, err := service.CreateCustomScope(t.Context(), CustomScopeDto{Key: "kuber", Name: "Kubernetes role", Description: "Your Kubernetes role"})
	require.NoError(t, err)

	policyWithScope := func(scope string) OidcClaimMappingPolicyDto {
		return OidcClaimMappingPolicyDto{
			OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Kubernetes"},
			ClaimMappings: []OidcClaimMappingDto{
				{ClaimName: "k8s_role", SourceType: string(MappingSourceCustomClaim), SourceValue: "k8s_role", Scope: []string{scope}, IDToken: true},
			},
		}
	}

	_, err = service.CreateClaimMappingPolicy(t.Context(), policyWithScope("unknown"))
	require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed), "a scope that doesn't exist must be rejected, got %v", err)

	_, err = service.CreateClaimMappingPolicy(t.Context(), policyWithScope("kuber"))
	require.NoError(t, err)

	scopes, _, err := service.ListCustomScopes(t.Context(), "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"Kubernetes"}, scopes[0].Policies)

	err = service.DeleteCustomScope(t.Context(), kuber.ID)
	require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed), "a scope a policy releases claims with must not be deleted, got %v", err)
}
