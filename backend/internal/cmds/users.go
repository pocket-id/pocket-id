package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// usersCmd represents the users command
var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long:  `Create, list, update, and delete users.`,
}

func init() {
	// Add users command to root
	rootCmd.AddCommand(usersCmd)

	// Add subcommands
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersGetCmd)
	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersUpdateCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.AddCommand(usersGetMeCmd)
	usersCmd.AddCommand(usersUpdateMeCmd)
}

// usersListCmd represents the list users command
var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  `Get a paginated list of users.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		sortOrder, _ := cmd.Flags().GetString("sort-order")
		search, _ := cmd.Flags().GetString("search")

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Prepare query parameters
		query := ListRequestOptions{
			Page:      page,
			Limit:     limit,
			SortBy:    sortBy,
			SortOrder: sortOrder,
			Search:    search,
		}.ToQuery()

		// Make request
		var response PaginatedResponse[dto.UserDto]
		if err := client.Get(ctx, "/api/users", &response, query); err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		// Output result
		return outputResult(response, globalFlags.format)
	},
}

// usersGetCmd represents the get user command
var usersGetCmd = &cobra.Command{
	Use:   "get [user-id]",
	Short: "Get a user by ID",
	Long:  `Get detailed information about a specific user.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var user dto.UserDto
		if err := client.Get(ctx, "/api/users/"+userID, &user, nil); err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Output result
		return outputResult(user, globalFlags.format)
	},
}

// usersCreateCmd represents the create user command
var usersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  `Create a new user with the specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		displayName, _ := cmd.Flags().GetString("display-name")
		isAdmin, _ := cmd.Flags().GetBool("admin")
		locale, _ := cmd.Flags().GetString("locale")
		disabled, _ := cmd.Flags().GetBool("disabled")

		// Validate required fields
		if username == "" {
			return fmt.Errorf("username is required")
		}
		if firstName == "" {
			return fmt.Errorf("first name is required")
		}
		// Generate display name from first and last name if not provided
		if displayName == "" {
			if lastName != "" {
				displayName = firstName + " " + lastName
			} else {
				displayName = firstName
			}
		}

		// Prepare request body
		createDto := dto.UserCreateDto{
			Username:    username,
			FirstName:   firstName,
			LastName:    lastName,
			DisplayName: displayName,
			IsAdmin:     isAdmin,
			Disabled:    disabled,
		}

		// Set optional fields
		if email != "" {
			createDto.Email = &email
		}
		if locale != "" {
			createDto.Locale = &locale
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var user dto.UserDto
		if err := client.Post(ctx, "/api/users", createDto, &user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Output result
		fmt.Printf("User created successfully:\n")
		return outputResult(user, globalFlags.format)
	},
}

// usersUpdateCmd represents the update user command
var usersUpdateCmd = &cobra.Command{
	Use:   "update [user-id]",
	Short: "Update a user",
	Long:  `Update an existing user's details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userID := args[0]

		// Parse flags
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		displayName, _ := cmd.Flags().GetString("display-name")
		isAdmin, _ := cmd.Flags().GetBool("admin")
		locale, _ := cmd.Flags().GetString("locale")
		disabled, _ := cmd.Flags().GetBool("disabled")

		// Get existing user first
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		var existingUser dto.UserDto
		if err := client.Get(ctx, "/api/users/"+userID, &existingUser, nil); err != nil {
			return fmt.Errorf("failed to get existing user: %w", err)
		}

		// Prepare update body (only include fields that were explicitly set)
		updateBody := make(map[string]interface{})

		if cmd.Flags().Changed("username") {
			updateBody["username"] = username
		}
		if cmd.Flags().Changed("email") {
			if email == "" {
				updateBody["email"] = nil
			} else {
				updateBody["email"] = email
			}
		}
		if cmd.Flags().Changed("first-name") {
			updateBody["firstName"] = firstName
		}
		if cmd.Flags().Changed("last-name") {
			updateBody["lastName"] = lastName
		}
		if cmd.Flags().Changed("display-name") {
			updateBody["displayName"] = displayName
		}
		if cmd.Flags().Changed("admin") {
			updateBody["isAdmin"] = isAdmin
		}
		if cmd.Flags().Changed("locale") {
			if locale == "" {
				updateBody["locale"] = nil
			} else {
				updateBody["locale"] = locale
			}
		}
		if cmd.Flags().Changed("disabled") {
			updateBody["disabled"] = disabled
		}

		// If no fields were updated, return
		if len(updateBody) == 0 {
			fmt.Println("No fields to update")
			return nil
		}

		// Make update request
		var updatedUser dto.UserDto
		if err := client.Put(ctx, "/api/users/"+userID, updateBody, &updatedUser); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		// Output result
		fmt.Printf("User updated successfully:\n")
		return outputResult(updatedUser, globalFlags.format)
	},
}

// usersDeleteCmd represents the delete user command
var usersDeleteCmd = &cobra.Command{
	Use:   "delete [user-id]",
	Short: "Delete a user",
	Long:  `Delete a user by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userID := args[0]

		// Confirm deletion
		if !forceDelete {
			fmt.Printf("Are you sure you want to delete user %s? (y/N): ", userID)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("Deletion cancelled")
				return nil
			}
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Delete(ctx, "/api/users/"+userID); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		fmt.Printf("User %s deleted successfully\n", userID)
		return nil
	},
}

// usersGetMeCmd represents the get current user command
var usersGetMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Get current user information",
	Long:  `Get information about the currently authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var user dto.UserDto
		if err := client.Get(ctx, "/api/users/me", &user, nil); err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		// Output result
		return outputResult(user, globalFlags.format)
	},
}

// usersUpdateMeCmd represents the update current user command
var usersUpdateMeCmd = &cobra.Command{
	Use:   "update-me",
	Short: "Update current user",
	Long:  `Update the currently authenticated user's details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		email, _ := cmd.Flags().GetString("email")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		displayName, _ := cmd.Flags().GetString("display-name")
		locale, _ := cmd.Flags().GetString("locale")

		// Get existing user first
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		var existingUser dto.UserDto
		if err := client.Get(ctx, "/api/users/me", &existingUser, nil); err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		// Prepare update body (only include fields that were explicitly set)
		updateBody := make(map[string]interface{})

		if cmd.Flags().Changed("email") {
			if email == "" {
				updateBody["email"] = nil
			} else {
				updateBody["email"] = email
			}
		}
		if cmd.Flags().Changed("first-name") {
			updateBody["firstName"] = firstName
		}
		if cmd.Flags().Changed("last-name") {
			updateBody["lastName"] = lastName
		}
		if cmd.Flags().Changed("display-name") {
			updateBody["displayName"] = displayName
		}
		if cmd.Flags().Changed("locale") {
			if locale == "" {
				updateBody["locale"] = nil
			} else {
				updateBody["locale"] = locale
			}
		}

		// If no fields were updated, return
		if len(updateBody) == 0 {
			fmt.Println("No fields to update")
			return nil
		}

		// Make update request
		var updatedUser dto.UserDto
		if err := client.Put(ctx, "/api/users/me", updateBody, &updatedUser); err != nil {
			return fmt.Errorf("failed to update current user: %w", err)
		}

		// Output result
		fmt.Printf("Current user updated successfully:\n")
		return outputResult(updatedUser, globalFlags.format)
	},
}

// Global flag for force deletion
var forceDelete bool

func init() {
	// Add flags to subcommands
	usersListCmd.Flags().Int("page", 1, "Page number")
	usersListCmd.Flags().Int("limit", 20, "Items per page")
	usersListCmd.Flags().String("sort-by", "", "Sort by column")
	usersListCmd.Flags().String("sort-order", "asc", "Sort order (asc/desc)")
	usersListCmd.Flags().String("search", "", "Search term")

	usersCreateCmd.Flags().String("username", "", "Username (required)")
	usersCreateCmd.Flags().String("email", "", "Email address")
	usersCreateCmd.Flags().String("first-name", "", "First name (required)")
	usersCreateCmd.Flags().String("last-name", "", "Last name")
	usersCreateCmd.Flags().String("display-name", "", "Display name (generated from first and last name if not provided)")
	usersCreateCmd.Flags().Bool("admin", false, "Make user an admin")
	usersCreateCmd.Flags().String("locale", "", "Locale (e.g., en-US)")
	usersCreateCmd.Flags().Bool("disabled", false, "Disable the user")
	usersCreateCmd.MarkFlagRequired("username")
	usersCreateCmd.MarkFlagRequired("first-name")

	usersUpdateCmd.Flags().String("username", "", "Username")
	usersUpdateCmd.Flags().String("email", "", "Email address")
	usersUpdateCmd.Flags().String("first-name", "", "First name")
	usersUpdateCmd.Flags().String("last-name", "", "Last name")
	usersUpdateCmd.Flags().String("display-name", "", "Display name")
	usersUpdateCmd.Flags().Bool("admin", false, "Make user an admin")
	usersUpdateCmd.Flags().String("locale", "", "Locale (e.g., en-US)")
	usersUpdateCmd.Flags().Bool("disabled", false, "Disable the user")

	usersDeleteCmd.Flags().BoolVar(&forceDelete, "force", false, "Force deletion without confirmation")

	usersUpdateMeCmd.Flags().String("email", "", "Email address")
	usersUpdateMeCmd.Flags().String("first-name", "", "First name")
	usersUpdateMeCmd.Flags().String("last-name", "", "Last name")
	usersUpdateMeCmd.Flags().String("display-name", "", "Display name")
	usersUpdateMeCmd.Flags().String("locale", "", "Locale (e.g., en-US)")
}

// outputResult outputs data in the specified format
func outputResult(data interface{}, format string) error {
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
		// Simple table output for lists
		switch v := data.(type) {
		case PaginatedResponse[dto.UserDto]:
			fmt.Printf("%-36s %-20s %-30s %-15s %-10s\n", "ID", "Username", "Email", "Display Name", "Admin")
			fmt.Println(strings.Repeat("-", 120))
			for _, user := range v.Data {
				email := ""
				if user.Email != nil {
					email = *user.Email
				}
				adminStatus := "No"
				if user.IsAdmin {
					adminStatus = "Yes"
				}
				fmt.Printf("%-36s %-20s %-30s %-15s %-10s\n",
					user.ID,
					user.Username,
					truncate(email, 28),
					truncate(user.DisplayName, 13),
					adminStatus)
			}
			fmt.Printf("\nPage %d of %d (Total: %d items)\n",
				v.Pagination.Page,
				v.Pagination.TotalPages,
				v.Pagination.TotalItems)
		case dto.UserDto:
			fmt.Printf("ID:           %s\n", v.ID)
			fmt.Printf("Username:     %s\n", v.Username)
			if v.Email != nil {
				fmt.Printf("Email:        %s\n", *v.Email)
			} else {
				fmt.Printf("Email:        (not set)\n")
			}
			fmt.Printf("First Name:   %s\n", v.FirstName)
			if v.LastName != nil {
				fmt.Printf("Last Name:    %s\n", *v.LastName)
			}
			fmt.Printf("Display Name: %s\n", v.DisplayName)
			fmt.Printf("Admin:        %v\n", v.IsAdmin)
			if v.Locale != nil {
				fmt.Printf("Locale:       %s\n", *v.Locale)
			}
			fmt.Printf("Disabled:     %v\n", v.Disabled)
			if len(v.UserGroups) > 0 {
				fmt.Printf("User Groups:  ")
				groupNames := make([]string, len(v.UserGroups))
				for i, group := range v.UserGroups {
					groupNames[i] = group.Name
				}
				fmt.Println(strings.Join(groupNames, ", "))
			}
		default:
			// Fall back to JSON for other types
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(data)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// truncate truncates a string to the specified length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
