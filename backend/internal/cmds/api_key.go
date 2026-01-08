package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// apiKeyCmd represents the api-key command
var apiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Manage API keys",
	Long:  `Generate, list, and revoke API keys for accessing the Pocket ID API.`,
}

func init() {
	// Add api-key command to root
	rootCmd.AddCommand(apiKeyCmd)

	// Add subcommands
	apiKeyCmd.AddCommand(apiKeyCreateCmd)
	apiKeyCmd.AddCommand(apiKeyListCmd)
	apiKeyCmd.AddCommand(apiKeyRevokeCmd)
	apiKeyCmd.AddCommand(apiKeyGenerateCmd)
}

// apiKeyGenerateCmd generates an API key directly in the database (for setup/backup)
var apiKeyGenerateCmd = &cobra.Command{
	Use:   "generate [username or email]",
	Short: "Generate an API key directly in database",
	Long: `Generate an API key directly in the database for the given user.

This command bypasses the API and writes directly to the database.
It's useful for initial setup or recovery scenarios when the API is not accessible.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the username or email of the user
		userArg := args[0]

		// Parse flags
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		expiresIn, _ := cmd.Flags().GetString("expires-in")
		showToken, _ := cmd.Flags().GetBool("show-token")

		// Validate name
		if name == "" {
			return fmt.Errorf("API key name is required")
		}

		// Parse expiration duration
		var expiresAt time.Time
		if expiresIn != "" {
			duration, err := time.ParseDuration(expiresIn)
			if err != nil {
				return fmt.Errorf("invalid duration format for --expires-in: %w", err)
			}
			expiresAt = time.Now().Add(duration)
		} else {
			// Default: 1 year
			expiresAt = time.Now().Add(365 * 24 * time.Hour)
		}

		// Connect to the database
		db, err := bootstrap.NewDatabase()
		if err != nil {
			return err
		}

		// Generate the API key
		var apiKey *model.ApiKey
		var token string
		err = db.Transaction(func(tx *gorm.DB) error {
			// Load the user to retrieve the user ID
			var user model.User
			queryCtx, queryCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer queryCancel()
			txErr := tx.
				WithContext(queryCtx).
				Where("username = ? OR email = ?", userArg, userArg).
				First(&user).
				Error
			switch {
			case errors.Is(txErr, gorm.ErrRecordNotFound):
				return errors.New("user not found")
			case txErr != nil:
				return fmt.Errorf("failed to query for user: %w", txErr)
			case user.ID == "":
				return errors.New("invalid user loaded: ID is empty")
			}

			// Generate a secure random API key token
			token, txErr = utils.GenerateRandomAlphanumericString(32)
			if txErr != nil {
				return fmt.Errorf("failed to generate API key token: %w", txErr)
			}

			// Create the API key
			apiKey = &model.ApiKey{
				Base: model.Base{
					ID: uuid.New().String(),
				},
				Name:      name,
				Key:       utils.CreateSha256Hash(token),
				UserID:    user.ID,
				ExpiresAt: datatype.DateTime(expiresAt),
			}

			// Set description if provided
			if description != "" {
				apiKey.Description = &description
			}

			queryCtx, queryCancel = context.WithTimeout(cmd.Context(), 10*time.Second)
			defer queryCancel()
			txErr = tx.
				WithContext(queryCtx).
				Create(apiKey).
				Error
			if txErr != nil {
				return fmt.Errorf("failed to save API key: %w", txErr)
			}

			return nil
		})
		if err != nil {
			return err
		}

		// Output result based on format
		switch strings.ToLower(globalFlags.format) {
		case "json":
			// Create API key DTO for JSON output
			apiKeyDto := dto.ApiKeyDto{
				ID:                  apiKey.ID,
				Name:                apiKey.Name,
				Description:         apiKey.Description,
				ExpiresAt:           apiKey.ExpiresAt,
				LastUsedAt:          apiKey.LastUsedAt,
				CreatedAt:           datatype.DateTime(apiKey.CreatedAt),
				ExpirationEmailSent: apiKey.ExpirationEmailSent,
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(map[string]interface{}{
				"apiKey": apiKeyDto,
				"token":  token,
			})
		case "yaml":
			// Create API key DTO for YAML output
			apiKeyDto := dto.ApiKeyDto{
				ID:                  apiKey.ID,
				Name:                apiKey.Name,
				Description:         apiKey.Description,
				ExpiresAt:           apiKey.ExpiresAt,
				LastUsedAt:          apiKey.LastUsedAt,
				CreatedAt:           datatype.DateTime(apiKey.CreatedAt),
				ExpirationEmailSent: apiKey.ExpirationEmailSent,
			}

			// For now, fall back to JSON if YAML is not available
			yamlData, err := json.Marshal(map[string]interface{}{
				"apiKey": apiKeyDto,
				"token":  token,
			})
			if err != nil {
				return err
			}
			fmt.Println(string(yamlData))
			return nil
		default: // table or default
			fmt.Printf("API key generated successfully:\n")
			fmt.Printf("Key ID: %s\n", apiKey.ID)
			fmt.Printf("Name: %s\n", apiKey.Name)
			if apiKey.Description != nil {
				fmt.Printf("Description: %s\n", *apiKey.Description)
			}
			fmt.Printf("Expires at: %s\n", apiKey.ExpiresAt.ToTime().Format(time.RFC3339))
			if showToken {
				fmt.Printf("\nAPI Token:\n%s\n", token)
				fmt.Println("\nWarning: Save this token securely.")
			} else {
				fmt.Println("\nAPI token generated but not shown (use --show-token to display)")
			}
			return nil
		}
	},
}

// apiKeyCreateCmd creates an API key via API (requires running server)
var apiKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key via API",
	Long:  `Create a new API key for the authenticated user via the Pocket ID API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		expiresIn, _ := cmd.Flags().GetString("expires-in")

		// Validate name
		if name == "" {
			return fmt.Errorf("API key name is required")
		}

		// Parse expiration duration
		var expiresAt time.Time
		if expiresIn != "" {
			duration, err := time.ParseDuration(expiresIn)
			if err != nil {
				return fmt.Errorf("invalid duration format for --expires-in: %w", err)
			}
			expiresAt = time.Now().Add(duration)
		} else {
			// Default: 1 year
			expiresAt = time.Now().Add(365 * 24 * time.Hour)
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		if globalFlags.apiKey == "" {
			return fmt.Errorf("API key is required. Set via --api-key flag or POCKET_ID_API_KEY environment variable")
		}

		// Prepare request body
		expiresAtDateTime := datatype.DateTime(expiresAt)
		createDto := dto.ApiKeyCreateDto{
			Name:      name,
			ExpiresAt: expiresAtDateTime,
		}

		// Set description if provided
		if description != "" {
			createDto.Description = &description
		}

		// Make request
		var response struct {
			ApiKey dto.ApiKeyDto `json:"apiKey"`
			Token  string        `json:"token"`
		}
		if err := client.Post(ctx, "/api/api-keys", createDto, &response); err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}

		// Output result based on format
		switch strings.ToLower(globalFlags.format) {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(map[string]interface{}{
				"apiKey": response.ApiKey,
				"token":  response.Token,
			})
		case "yaml":
			// For now, fall back to JSON if YAML is not available
			yamlData, err := json.Marshal(map[string]interface{}{
				"apiKey": response.ApiKey,
				"token":  response.Token,
			})
			if err != nil {
				return err
			}
			fmt.Println(string(yamlData))
			return nil
		default: // table or default
			fmt.Printf("API key created successfully:\n")
			fmt.Printf("Key ID: %s\n", response.ApiKey.ID)
			fmt.Printf("Name: %s\n", response.ApiKey.Name)
			if response.ApiKey.Description != nil {
				fmt.Printf("Description: %s\n", *response.ApiKey.Description)
			}
			fmt.Printf("Expires at: %s\n", response.ApiKey.ExpiresAt.ToTime().Format(time.RFC3339))
			fmt.Printf("\nAPI Token (save this now, it won't be shown again):\n%s\n", response.Token)
			fmt.Println("\nWarning: This token will not be displayed again. Save it securely.")
			return nil
		}
	},
}

// apiKeyListCmd lists API keys via API
var apiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Long:  `List API keys for the authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		if globalFlags.apiKey == "" {
			return fmt.Errorf("API key is required. Set via --api-key flag or POCKET_ID_API_KEY environment variable")
		}

		// Prepare query parameters
		query := ListRequestOptions{
			Page:  page,
			Limit: limit,
		}.ToQuery()

		// Make request
		var response PaginatedResponse[dto.ApiKeyDto]
		if err := client.Get(ctx, "/api/api-keys", &response, query); err != nil {
			return fmt.Errorf("failed to list API keys: %w", err)
		}

		// Output result
		return outputResult(response, globalFlags.format)
	},
}

// apiKeyRevokeCmd revokes an API key via API
var apiKeyRevokeCmd = &cobra.Command{
	Use:   "revoke [api-key-id]",
	Short: "Revoke an API key",
	Long:  `Revoke (delete) an API key by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		apiKeyID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		if globalFlags.apiKey == "" {
			return fmt.Errorf("API key is required. Set via --api-key flag or POCKET_ID_API_KEY environment variable")
		}

		// Make request
		if err := client.Delete(ctx, "/api/api-keys/"+apiKeyID); err != nil {
			return fmt.Errorf("failed to revoke API key: %w", err)
		}

		fmt.Printf("API key %s revoked successfully\n", apiKeyID)
		return nil
	},
}

func init() {
	// Add flags to apiKeyGenerateCmd
	apiKeyGenerateCmd.Flags().String("name", "", "Name for the API key (required)")
	apiKeyGenerateCmd.Flags().String("description", "", "Description for the API key")
	apiKeyGenerateCmd.Flags().String("expires-in", "", "Expiration duration (e.g., 720h, 30d, 1y). Default: 1 year")
	apiKeyGenerateCmd.Flags().Bool("show-token", false, "Show the API token (otherwise only metadata is shown)")
	apiKeyGenerateCmd.MarkFlagRequired("name")

	// Add flags to apiKeyCreateCmd
	apiKeyCreateCmd.Flags().String("name", "", "Name for the API key (required)")
	apiKeyCreateCmd.Flags().String("description", "", "Description for the API key")
	apiKeyCreateCmd.Flags().String("expires-in", "", "Expiration duration (e.g., 720h, 30d, 1y). Default: 1 year")
	apiKeyCreateCmd.MarkFlagRequired("name")

	// Add flags to apiKeyListCmd
	apiKeyListCmd.Flags().Int("page", 1, "Page number")
	apiKeyListCmd.Flags().Int("limit", 20, "Items per page")
}
