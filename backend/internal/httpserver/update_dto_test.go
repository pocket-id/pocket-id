package httpserver_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// A PATCH fills the update DTO from the current resource, so a field the resource can't fill is reset whenever the body leaves it out
// Only fields whose zero value keeps the stored state, or that the update ignores, may be listed as unmapped
func TestUpdateDTOsAreFilledFromTheirResource(t *testing.T) {
	tests := []struct {
		name     string
		resource any
		update   any
		unmapped []string
	}{
		// Updates ignore the groups, which have their own endpoint
		{name: "user", resource: model.User{}, update: dto.UserCreateDto{}, unmapped: []string{"UserGroupIds"}},
		{name: "user group", resource: model.UserGroup{}, update: dto.UserGroupCreateDto{}},
		// A nil logo URL keeps the current logo
		{name: "OIDC client", resource: model.OidcClient{}, update: dto.OidcClientUpdateDto{}, unmapped: []string{"LogoURL", "DarkLogoURL"}},
		{name: "app config", resource: appconfig.AppConfigModel{}, update: dto.AppConfigUpdateDto{}},
		// The counts are only set in responses
		{name: "claim mapping policy", resource: claimmappingpolicy.OidcClaimMappingPolicy{}, update: claimmappingpolicy.OidcClaimMappingPolicyDto{}, unmapped: []string{"ClaimCount", "ClientCount"}},
		{name: "custom scope", resource: claimmappingpolicy.CustomScope{}, update: claimmappingpolicy.CustomScopeDto{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.ElementsMatch(t, test.unmapped, testutils.UnmappedFields(test.resource, test.update))
		})
	}
}
