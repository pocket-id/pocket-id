package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

// setupCmd represents the setup command group
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initial setup commands",
	Long:  `Commands for initial Pocket ID setup and administration.`,
}

func init() {
	// Add setup command to root
	rootCmd.AddCommand(setupCmd)

	// Add subcommands
	setupCmd.AddCommand(setupCreateAdminCmd)
}

// setupCreateAdminCmd creates the initial admin user via API
var setupCreateAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create initial admin user",
	Long: `Create the initial admin user for a fresh Pocket ID installation.

This command requires that no users exist in the database yet.
It will create the first user with admin privileges and return an access token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")

		// Validate required fields
		if username == "" {
			return fmt.Errorf("username is required")
		}
		if firstName == "" {
			return fmt.Errorf("first name is required")
		}
		if email == "" {
			return fmt.Errorf("email is required")
		}

		// Create API client (no API key needed for setup)
		client := NewAPIClient(globalFlags.endpoint, "")

		// Prepare request body
		signUpDto := dto.SignUpDto{
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		}

		// Set email (required)
		signUpDto.Email = &email

		// Make request to setup endpoint
		var userDto dto.UserDto

		if err := client.Post(ctx, "/api/signup/setup", signUpDto, &userDto); err != nil {
			// Check if setup was already completed
			if strings.Contains(err.Error(), "Setup already completed") {
				return fmt.Errorf("initial setup already completed - users already exist in the database")
			}
			return fmt.Errorf("failed to create initial admin: %w", err)
		}

		// Output result based on format
		if globalFlags.format == "json" {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(userDto)
		}

		// Default text output
		fmt.Println("✅ Initial admin user created successfully!")
		fmt.Println()
		fmt.Printf("User ID:      %s\n", userDto.ID)
		fmt.Printf("Username:     %s\n", userDto.Username)
		if userDto.Email != nil {
			fmt.Printf("Email:        %s\n", *userDto.Email)
		}
		fmt.Printf("First Name:   %s\n", userDto.FirstName)
		if userDto.LastName != nil {
			fmt.Printf("Last Name:    %s\n", *userDto.LastName)
		}
		fmt.Printf("Display Name: %s\n", userDto.DisplayName)
		fmt.Printf("Admin:        %v\n", userDto.IsAdmin)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Generate an API key for automation:")
		fmt.Printf("   pocket-id api-key generate \"%s\" --name \"CLI Key\" --show-token\n", username)
		fmt.Println("2. Set the API key as environment variable:")
		fmt.Println("   export POCKET_ID_API_KEY=\"your-generated-api-key\"")
		fmt.Println("3. Use the API key for CLI commands:")
		fmt.Println("   pocket-id --api-key \"YOUR_API_KEY\" users list")

		return nil
	},
}

func init() {
	// Add flags to setupCreateAdminCmd
	setupCreateAdminCmd.Flags().String("username", "", "Username for the admin user (required)")
	setupCreateAdminCmd.Flags().String("email", "", "Email address for the admin user (required)")
	setupCreateAdminCmd.Flags().String("first-name", "", "First name (required)")
	setupCreateAdminCmd.Flags().String("last-name", "", "Last name")
	setupCreateAdminCmd.MarkFlagRequired("username")
	setupCreateAdminCmd.MarkFlagRequired("first-name")
	setupCreateAdminCmd.MarkFlagRequired("email")
}
