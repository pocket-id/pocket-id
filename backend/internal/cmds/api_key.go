package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// apiKeyCmd represents the api-key command group
var apiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Manage API keys",
	Long:  `Create, list, and revoke API keys.`,
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

// apiKeyGenerateCmd generates an API key directly (database access)
var apiKeyGenerateCmd = &cobra.Command{
	Use:   "generate [username or email]",
	Short: "Generate a new API key for a user",
	Long:  `Generate a new API key for the specified user with direct database access.`,
	Args:  cobra.ExactArgs(1),
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

			// Create API key service
			apiKeyService := service.NewApiKeyService(tx, nil) // No email service needed for CLI

			// Create API key DTO
			expiresAtDateTime := datatype.DateTime(expiresAt)
			createDto := dto.ApiKeyCreateDto{
				Name:      name,
				ExpiresAt: expiresAtDateTime,
			}

			// Set description if provided
			if description != "" {
				createDto.Description = &description
			}

			// Create the API key
			createdApiKey, createdToken, txErr := apiKeyService.CreateApiKey(queryCtx, user.ID, createDto)
			if txErr != nil {
				return fmt.Errorf("failed to generate API key: %w", txErr)
			}

			apiKey = &createdApiKey
			token = createdToken
			return nil
		})
		if err != nil {
			return err
		}

		// Print the result
		fmt.Printf("API key created successfully for user: %s\n", userArg)
		fmt.Printf("Key ID: %s\n", apiKey.ID)
		fmt.Printf("Name: %s\n", apiKey.Name)
		if apiKey.Description != nil {
			fmt.Printf("Description: %s\n", *apiKey.Description)
		}
		fmt.Printf("Expires at: %s\n", apiKey.ExpiresAt.ToTime().Format(time.RFC3339))

		if showToken {
			fmt.Printf("\nAPI Token (save this now, it won't be shown again):\n%s\n", token)
			fmt.Println("\nWarning: This token will not be displayed again. Save it securely.")
		} else {
			fmt.Println("\nNote: Use --show-token flag to display the API token.")
		}

		return nil
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

		// Output result
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
	},
}

// apiKeyListCmd lists API keys via API
var apiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys via API",
	Long:  `List API keys for the authenticated user via the Pocket ID API.`,
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
		return outputApiKeys(response, globalFlags.format)
	},
}

// apiKeyRevokeCmd revokes an API key via API
var apiKeyRevokeCmd = &cobra.Command{
	Use:   "revoke [key-id]",
	Short: "Revoke an API key via API",
	Long:  `Revoke (delete) an API key by ID via the Pocket ID API.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		keyID := args[0]

		// Confirm revocation
		if !forceDelete {
			fmt.Printf("Are you sure you want to revoke API key %s? (y/N): ", keyID)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("Revocation cancelled")
				return nil
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		if globalFlags.apiKey == "" {
			return fmt.Errorf("API key is required. Set via --api-key flag or POCKET_ID_API_KEY environment variable")
		}

		// Make request
		if err := client.Delete(ctx, "/api/api-keys/"+keyID); err != nil {
			return fmt.Errorf("failed to revoke API key: %w", err)
		}

		fmt.Printf("API key %s revoked successfully\n", keyID)
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

	// Add flags to apiKeyRevokeCmd
	apiKeyRevokeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force revocation without confirmation")
}

// outputApiKeys outputs API keys in the specified format
func outputApiKeys(data PaginatedResponse[dto.ApiKeyDto], format string) error {
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "yaml":
		// For now, fall back to JSON if YAML is not available
		yamlData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
		return nil
	case "table":
		fmt.Printf("%-36s %-20s %-30s %-25s %-25s\n", "ID", "Name", "Description", "Expires At", "Last Used")
		fmt.Println(strings.Repeat("-", 140))
		for _, key := range data.Data {
			desc := ""
			if key.Description != nil {
				desc = *key.Description
			}
			expiresAt := key.ExpiresAt.ToTime().Format("2006-01-02 15:04:05")
			lastUsed := ""
			if key.LastUsedAt != nil {
				lastUsed = key.LastUsedAt.ToTime().Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-36s %-20s %-30s %-25s %-25s\n",
				key.ID,
				truncate(key.Name, 18),
				truncate(desc, 28),
				expiresAt,
				lastUsed)
		}
		fmt.Printf("\nPage %d of %d (Total: %d items)\n",
			data.Pagination.Page,
			data.Pagination.TotalPages,
			data.Pagination.TotalItems)
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
