package cmds

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// scimCmd represents the scim command
var scimCmd = &cobra.Command{
	Use:   "scim",
	Short: "Manage SCIM service providers",
	Long:  `Create, update, delete, and sync SCIM service providers.`,
}

func init() {
	// Add scim command to root
	rootCmd.AddCommand(scimCmd)

	// Add subcommands
	scimCmd.AddCommand(scimCreateCmd)
	scimCmd.AddCommand(scimUpdateCmd)
	scimCmd.AddCommand(scimDeleteCmd)
	scimCmd.AddCommand(scimSyncCmd)
}

// scimCreateCmd represents the create SCIM service provider command
var scimCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a SCIM service provider",
	Long:  `Create a new SCIM service provider.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		endpoint, _ := cmd.Flags().GetString("endpoint")
		token, _ := cmd.Flags().GetString("token")
		oidcClientID, _ := cmd.Flags().GetString("oidc-client-id")
		filePath, _ := cmd.Flags().GetString("file")

		var input dto.ScimServiceProviderCreateDTO

		// Read from file if provided
		if filePath != "" {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read SCIM config file: %w", err)
			}

			if err := json.Unmarshal(fileData, &input); err != nil {
				return fmt.Errorf("failed to parse SCIM config file: %w", err)
			}
		} else {
			// Validate required flags
			if endpoint == "" {
				return fmt.Errorf("--endpoint flag is required")
			}
			if oidcClientID == "" {
				return fmt.Errorf("--oidc-client-id flag is required")
			}

			input = dto.ScimServiceProviderCreateDTO{
				Endpoint:     endpoint,
				Token:        token,
				OidcClientID: oidcClientID,
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var provider dto.ScimServiceProviderDTO
		if err := client.Post(ctx, "/api/scim/service-provider", &provider, input); err != nil {
			return fmt.Errorf("failed to create SCIM service provider: %w", err)
		}

		// Output result
		return outputResult(provider, globalFlags.format)
	},
}

// scimUpdateCmd represents the update SCIM service provider command
var scimUpdateCmd = &cobra.Command{
	Use:   "update [provider-id]",
	Short: "Update a SCIM service provider",
	Long:  `Update an existing SCIM service provider.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		providerID := args[0]

		// Parse flags
		endpoint, _ := cmd.Flags().GetString("endpoint")
		token, _ := cmd.Flags().GetString("token")
		oidcClientID, _ := cmd.Flags().GetString("oidc-client-id")
		filePath, _ := cmd.Flags().GetString("file")

		var input dto.ScimServiceProviderCreateDTO

		// Read from file if provided
		if filePath != "" {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read SCIM config file: %w", err)
			}

			if err := json.Unmarshal(fileData, &input); err != nil {
				return fmt.Errorf("failed to parse SCIM config file: %w", err)
			}
		} else {
			// Try to read from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				// stdin has data
				stdinData, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read from stdin: %w", err)
				}
				if err := json.Unmarshal(stdinData, &input); err != nil {
					return fmt.Errorf("failed to parse stdin data: %w", err)
				}
			} else {
				// Use flag values
				input = dto.ScimServiceProviderCreateDTO{
					Endpoint:     endpoint,
					Token:        token,
					OidcClientID: oidcClientID,
				}
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var provider dto.ScimServiceProviderDTO
		if err := client.Put(ctx, "/api/scim/service-provider/"+providerID, &provider, input); err != nil {
			return fmt.Errorf("failed to update SCIM service provider: %w", err)
		}

		// Output result
		return outputResult(provider, globalFlags.format)
	},
}

// scimDeleteCmd represents the delete SCIM service provider command
var scimDeleteCmd = &cobra.Command{
	Use:   "delete [provider-id]",
	Short: "Delete a SCIM service provider",
	Long:  `Delete a SCIM service provider by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		providerID := args[0]

		// Parse flags
		force, _ := cmd.Flags().GetBool("force")

		// Confirm deletion unless force flag is used
		if !force {
			if !confirmDeletion(fmt.Sprintf("Are you sure you want to delete SCIM service provider %s?", providerID)) {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Delete(ctx, "/api/scim/service-provider/"+providerID); err != nil {
			return fmt.Errorf("failed to delete SCIM service provider: %w", err)
		}

		fmt.Printf("SCIM service provider %s deleted successfully\n", providerID)
		return nil
	},
}

// scimSyncCmd represents the sync SCIM service provider command
var scimSyncCmd = &cobra.Command{
	Use:   "sync [provider-id]",
	Short: "Sync a SCIM service provider",
	Long:  `Trigger synchronization for a SCIM service provider.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		providerID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Post(ctx, "/api/scim/service-provider/"+providerID+"/sync", nil, nil); err != nil {
			return fmt.Errorf("failed to sync SCIM service provider: %w", err)
		}

		fmt.Printf("SCIM service provider %s synchronization triggered successfully\n", providerID)
		return nil
	},
}

// Helper function to confirm deletion
func confirmDeletion(message string) bool {
	fmt.Printf("%s (y/N): ", message)

	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y"
}

func init() {
	// Add flags to create command
	scimCreateCmd.Flags().String("endpoint", "", "SCIM endpoint URL (required)")
	scimCreateCmd.Flags().String("token", "", "SCIM authentication token")
	scimCreateCmd.Flags().String("oidc-client-id", "", "OIDC client ID (required)")
	scimCreateCmd.Flags().String("file", "", "Path to JSON file containing SCIM configuration")

	// Add flags to update command
	scimUpdateCmd.Flags().String("endpoint", "", "SCIM endpoint URL")
	scimUpdateCmd.Flags().String("token", "", "SCIM authentication token")
	scimUpdateCmd.Flags().String("oidc-client-id", "", "OIDC client ID")
	scimUpdateCmd.Flags().String("file", "", "Path to JSON file containing SCIM configuration")

	// Add flags to delete command
	scimDeleteCmd.Flags().Bool("force", false, "Force deletion without confirmation")
}
