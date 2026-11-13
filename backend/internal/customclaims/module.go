package customclaims

import (
	"context"
	"net/http"

	"github.com/go-fuego/fuego/option"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// Module manages the custom claims admins attach to users and user groups
type Module struct {
	service *Service
	handler *handler
}

func New(db *gorm.DB) *Module {
	service := &Service{db: db}
	return &Module{
		service: service,
		handler: &handler{service: service},
	}
}

// RegisterRoutes mounts the custom claim endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	customClaimsGroup := r.Group("/admin/custom-claims")
	customClaimsGroup.GET("/suggestions", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.getSuggestions),
		option.Summary("Get custom claim suggestions"),
		option.Description("Get a list of suggested custom claim names"),
		option.Tags("Custom Claims"),
	)
	customClaimsGroup.PUT("/user/:userId", authz.UsersWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateForUser),
		option.Summary("Update custom claims for a user"),
		option.Description("Update or create custom claims for a specific user"),
		option.Tags("Custom Claims"),
		option.Path("userId", "User ID"),
	)
	customClaimsGroup.PUT("/user-group/:userGroupId", authz.GroupsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateForUserGroup),
		option.Summary("Update custom claims for a user group"),
		option.Description("Update or create custom claims for a specific user group"),
		option.Tags("Custom Claims"),
		option.Path("userGroupId", "User Group ID"),
	)
}

// UpdateCustomClaimsForUserInternal replaces a user's custom claims within an existing transaction
// It is consumed by the users module, which applies the default custom claims to new users
func (m *Module) UpdateCustomClaimsForUserInternal(ctx context.Context, userID string, claims []CustomClaimCreateDto, tx *gorm.DB) ([]model.CustomClaim, error) {
	return m.service.updateCustomClaimsInternal(ctx, ownerUser, userID, claims, tx)
}

// GetCustomClaimsForUserWithUserGroups implements the OIDC module's CustomClaimSource interface
func (m *Module) GetCustomClaimsForUserWithUserGroups(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomClaim, error) {
	return m.service.GetCustomClaimsForUserWithUserGroups(ctx, userID, tx)
}
