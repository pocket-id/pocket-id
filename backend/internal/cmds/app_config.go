package cmds

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// appConfigCmd represents the app-config command
var appConfigCmd = &cobra.Command{
	Use:   "app-config",
	Short: "Manage application configuration",
	Long:  `View and update application configuration settings.`,
}

func init() {
	// Add app-config command to root
	rootCmd.AddCommand(appConfigCmd)

	// Add subcommands
	appConfigCmd.AddCommand(appConfigGetCmd)
	appConfigCmd.AddCommand(appConfigGetAllCmd)
	appConfigCmd.AddCommand(appConfigUpdateCmd)
	appConfigCmd.AddCommand(appConfigTestEmailCmd)
	appConfigCmd.AddCommand(appConfigSyncLdapCmd)
}

// appConfigGetCmd represents the get app config command
var appConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get public application configuration",
	Long:  `Get public application configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var configs []dto.PublicAppConfigVariableDto
		if err := client.Get(ctx, "/api/application-configuration", &configs, nil); err != nil {
			return fmt.Errorf("failed to get application configuration: %w", err)
		}

		// Output result
		return outputResult(configs, globalFlags.format)
	},
}

// appConfigGetAllCmd represents the get all app config command
var appConfigGetAllCmd = &cobra.Command{
	Use:   "get-all",
	Short: "Get all application configuration (including private)",
	Long:  `Get all application configuration settings including private ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var configs []dto.AppConfigVariableDto
		if err := client.Get(ctx, "/api/application-configuration/all", &configs, nil); err != nil {
			return fmt.Errorf("failed to get all application configuration: %w", err)
		}

		// Output result
		return outputResult(configs, globalFlags.format)
	},
}

// appConfigUpdateCmd represents the update app config command
var appConfigUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update application configuration",
	Long:  `Update application configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		filePath, _ := cmd.Flags().GetString("file")
		configJSON, _ := cmd.Flags().GetString("config")

		var input dto.AppConfigUpdateDto

		// Read from file if provided
		if filePath != "" {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			if err := json.Unmarshal(fileData, &input); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}
		} else if configJSON != "" {
			// Parse JSON string
			if err := json.Unmarshal([]byte(configJSON), &input); err != nil {
				return fmt.Errorf("failed to parse config JSON: %w", err)
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
				return fmt.Errorf("either --file or --config flag must be provided")
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var configs []dto.AppConfigVariableDto
		if err := client.Put(ctx, "/api/application-configuration", &configs, input); err != nil {
			return fmt.Errorf("failed to update application configuration: %w", err)
		}

		// Output result
		return outputResult(configs, globalFlags.format)
	},
}

// appConfigTestEmailCmd represents the test email command
var appConfigTestEmailCmd = &cobra.Command{
	Use:   "test-email",
	Short: "Send test email",
	Long:  `Send a test email to verify email configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Post(ctx, "/api/application-configuration/test-email", nil, nil); err != nil {
			return fmt.Errorf("failed to send test email: %w", err)
		}

		fmt.Println("Test email sent successfully")
		return nil
	},
}

// appConfigSyncLdapCmd represents the sync LDAP command
var appConfigSyncLdapCmd = &cobra.Command{
	Use:   "sync-ldap",
	Short: "Synchronize LDAP",
	Long:  `Manually trigger LDAP synchronization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Post(ctx, "/api/application-configuration/sync-ldap", nil, nil); err != nil {
			return fmt.Errorf("failed to sync LDAP: %w", err)
		}

		fmt.Println("LDAP synchronization triggered successfully")
		return nil
	},
}

func init() {
	// Add flags to update command
	appConfigUpdateCmd.Flags().String("file", "", "Path to JSON file containing configuration")
	appConfigUpdateCmd.Flags().StringP("config", "c", "", "JSON string containing configuration")
}
