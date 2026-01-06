package cmds

import (
	"fmt"
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
		var response struct {
			User  dto.UserDto `json:"user"`
			Token string      `json:"token"`
		}

		if err := client.Post(ctx, "/api/users/signup/setup", signUpDto, &response); err != nil {
			// Check if setup was already completed
			if strings.Contains(err.Error(), "Setup already completed") {
				return fmt.Errorf("initial setup already completed - users already exist in the database")
			}
			return fmt.Errorf("failed to create initial admin: %w", err)
		}

		// Output result
		fmt.Println("✅ Initial admin user created successfully!")
		fmt.Println()
		fmt.Printf("User ID:      %s\n", response.User.ID)
		fmt.Printf("Username:     %s\n", response.User.Username)
		if response.User.Email != nil {
			fmt.Printf("Email:        %s\n", *response.User.Email)
		}
		fmt.Printf("First Name:   %s\n", response.User.FirstName)
		if response.User.LastName != nil {
			fmt.Printf("Last Name:    %s\n", *response.User.LastName)
		}
		fmt.Printf("Display Name: %s\n", response.User.DisplayName)
		fmt.Printf("Admin:        %v\n", response.User.IsAdmin)
		fmt.Println()
		fmt.Println("🔑 Access Token (save this now):")
		fmt.Println(response.Token)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Use this token to authenticate in the web interface")
		fmt.Println("2. Generate an API key for automation:")
		fmt.Printf("   pocket-id api-key generate \"%s\" --name \"CLI Key\" --show-token\n", username)
		fmt.Println("3. Set the API key as environment variable:")
		fmt.Println("   export POCKET_ID_API_KEY=\"your-generated-api-key\"")

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
