package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// CreateInitialAdminIfNeeded checks if initial admin should be created via environment variables
// and creates it if the database has no users.
func CreateInitialAdminIfNeeded(ctx context.Context, db *gorm.DB) error {
	// Check if initial admin should be created
	if common.EnvConfig.InitialAdminUsername == "" {
		slog.DebugContext(ctx, "Initial admin creation via environment variables is not configured")
		return nil
	}

	// Validate required fields
	if common.EnvConfig.InitialAdminFirstName == "" {
		return fmt.Errorf("INITIAL_ADMIN_FIRST_NAME is required when INITIAL_ADMIN_USERNAME is set")
	}
	if common.EnvConfig.InitialAdminEmail == "" {
		return fmt.Errorf("INITIAL_ADMIN_EMAIL is required when INITIAL_ADMIN_USERNAME is set")
	}

	// Check if any users exist
	var userCount int64
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&userCount).Error; err != nil {
		return fmt.Errorf("failed to check user count: %w", err)
	}

	if userCount > 0 {
		slog.InfoContext(ctx, "Database already has users, skipping initial admin creation via environment variables")
		return nil
	}

	slog.InfoContext(ctx, "Creating initial admin user from environment variables",
		"username", common.EnvConfig.InitialAdminUsername,
		"first_name", common.EnvConfig.InitialAdminFirstName,
		"email", common.EnvConfig.InitialAdminEmail,
	)

	// Create app config service first (needed for JWT service)
	appConfigService, err := service.NewAppConfigService(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to create app config service: %w", err)
	}

	// Create JWT service
	jwtService, err := service.NewJwtService(db, appConfigService)
	if err != nil {
		return fmt.Errorf("failed to create JWT service: %w", err)
	}

	// Start a transaction for user creation
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Prepare user data
	displayName := strings.TrimSpace(common.EnvConfig.InitialAdminFirstName + " " + common.EnvConfig.InitialAdminLastName)
	if displayName == "" {
		displayName = common.EnvConfig.InitialAdminFirstName
	}

	user := model.User{
		FirstName:   common.EnvConfig.InitialAdminFirstName,
		DisplayName: displayName,
		Username:    common.EnvConfig.InitialAdminUsername,
		IsAdmin:     true,
	}

	// Set required email field
	user.Email = &common.EnvConfig.InitialAdminEmail

	// Set optional last name field
	if common.EnvConfig.InitialAdminLastName != "" {
		user.LastName = common.EnvConfig.InitialAdminLastName
	}

	// Save the user
	if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Generate access token
	token, err := jwtService.GenerateAccessToken(user)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.InfoContext(ctx, "Initial admin user created successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	if user.Email != nil {
		slog.InfoContext(ctx, "Initial admin email", "email", *user.Email)
	}

	slog.InfoContext(ctx, "Initial admin access token (save this securely):", "token", token)
	slog.InfoContext(ctx, "Next steps: Use this token to authenticate in the web interface or generate an API key")

	return nil
}
