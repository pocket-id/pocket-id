package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// userGroupsCmd represents the user-groups command
var userGroupsCmd = &cobra.Command{
	Use:   "user-groups",
	Short: "Manage user groups",
	Long:  `Create, list, update, and delete user groups.`,
}

func init() {
	// Add user-groups command to root
	rootCmd.AddCommand(userGroupsCmd)

	// Add subcommands
	userGroupsCmd.AddCommand(userGroupsListCmd)
	userGroupsCmd.AddCommand(userGroupsGetCmd)
	userGroupsCmd.AddCommand(userGroupsCreateCmd)
	userGroupsCmd.AddCommand(userGroupsUpdateCmd)
	userGroupsCmd.AddCommand(userGroupsDeleteCmd)
	userGroupsCmd.AddCommand(userGroupsUpdateUsersCmd)
	userGroupsCmd.AddCommand(userGroupsUpdateAllowedClientsCmd)
}

// userGroupsListCmd represents the list user groups command
var userGroupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List user groups",
	Long:  `Get a paginated list of user groups.`,
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
		var response PaginatedResponse[dto.UserGroupMinimalDto]
		if err := client.Get(ctx, "/api/user-groups", &response, query); err != nil {
			return fmt.Errorf("failed to list user groups: %w", err)
		}

		// Output result
		return outputUserGroupsResult(response, globalFlags.format)
	},
}

// userGroupsGetCmd represents the get user group command
var userGroupsGetCmd = &cobra.Command{
	Use:   "get [group-id]",
	Short: "Get a user group by ID",
	Long:  `Get detailed information about a specific user group.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		groupID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var group dto.UserGroupDto
		if err := client.Get(ctx, "/api/user-groups/"+groupID, &group, nil); err != nil {
			return fmt.Errorf("failed to get user group: %w", err)
		}

		// Output result
		return outputUserGroupResult(group, globalFlags.format)
	},
}

// userGroupsCreateCmd represents the create user group command
var userGroupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user group",
	Long:  `Create a new user group with the specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		name, _ := cmd.Flags().GetString("name")
		friendlyName, _ := cmd.Flags().GetString("friendly-name")

		// Validate required fields
		if name == "" {
			return fmt.Errorf("name is required")
		}
		if friendlyName == "" {
			return fmt.Errorf("friendly name is required")
		}

		// Prepare request body
		createDto := dto.UserGroupCreateDto{
			Name:         name,
			FriendlyName: friendlyName,
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var group dto.UserGroupDto
		if err := client.Post(ctx, "/api/user-groups", createDto, &group); err != nil {
			return fmt.Errorf("failed to create user group: %w", err)
		}

		// Output result
		fmt.Printf("User group created successfully:\n")
		return outputUserGroupResult(group, globalFlags.format)
	},
}

// userGroupsUpdateCmd represents the update user group command
var userGroupsUpdateCmd = &cobra.Command{
	Use:   "update [group-id]",
	Short: "Update a user group",
	Long:  `Update an existing user group's details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		groupID := args[0]

		// Parse flags
		name, _ := cmd.Flags().GetString("name")
		friendlyName, _ := cmd.Flags().GetString("friendly-name")

		// Get existing group first
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		var existingGroup dto.UserGroupDto
		if err := client.Get(ctx, "/api/user-groups/"+groupID, &existingGroup, nil); err != nil {
			return fmt.Errorf("failed to get existing user group: %w", err)
		}

		// Prepare update body (only include fields that were explicitly set)
		updateBody := make(map[string]interface{})

		if cmd.Flags().Changed("name") {
			updateBody["name"] = name
		}
		if cmd.Flags().Changed("friendly-name") {
			updateBody["friendlyName"] = friendlyName
		}

		// If no fields were updated, return
		if len(updateBody) == 0 {
			fmt.Println("No fields to update")
			return nil
		}

		// Make update request
		var updatedGroup dto.UserGroupDto
		if err := client.Put(ctx, "/api/user-groups/"+groupID, updateBody, &updatedGroup); err != nil {
			return fmt.Errorf("failed to update user group: %w", err)
		}

		// Output result
		fmt.Printf("User group updated successfully:\n")
		return outputUserGroupResult(updatedGroup, globalFlags.format)
	},
}

// userGroupsDeleteCmd represents the delete user group command
var userGroupsDeleteCmd = &cobra.Command{
	Use:   "delete [group-id]",
	Short: "Delete a user group",
	Long:  `Delete a user group by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		groupID := args[0]

		// Confirm deletion
		if !forceDelete {
			fmt.Printf("Are you sure you want to delete user group %s? (y/N): ", groupID)
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
		if err := client.Delete(ctx, "/api/user-groups/"+groupID); err != nil {
			return fmt.Errorf("failed to delete user group: %w", err)
		}

		fmt.Printf("User group %s deleted successfully\n", groupID)
		return nil
	},
}

// userGroupsUpdateUsersCmd represents the update users in group command
var userGroupsUpdateUsersCmd = &cobra.Command{
	Use:   "update-users [group-id]",
	Short: "Update users in a group",
	Long:  `Update the list of users belonging to a specific user group.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		groupID := args[0]

		// Parse flags
		userIDs, _ := cmd.Flags().GetStringSlice("user-ids")

		// Prepare request body
		updateDto := dto.UserGroupUpdateUsersDto{
			UserIDs: userIDs,
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var group dto.UserGroupDto
		if err := client.Put(ctx, "/api/user-groups/"+groupID+"/users", updateDto, &group); err != nil {
			return fmt.Errorf("failed to update users in group: %w", err)
		}

		// Output result
		fmt.Printf("Users updated successfully in group:\n")
		return outputUserGroupResult(group, globalFlags.format)
	},
}

// userGroupsUpdateAllowedClientsCmd represents the update allowed OIDC clients command
var userGroupsUpdateAllowedClientsCmd = &cobra.Command{
	Use:   "update-allowed-clients [group-id]",
	Short: "Update allowed OIDC clients for a group",
	Long:  `Update the OIDC clients allowed for a specific user group.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		groupID := args[0]

		// Parse flags
		clientIDs, _ := cmd.Flags().GetStringSlice("client-ids")

		// Prepare request body
		updateDto := dto.UserGroupUpdateAllowedOidcClientsDto{
			OidcClientIDs: clientIDs,
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var group dto.UserGroupDto
		if err := client.Put(ctx, "/api/user-groups/"+groupID+"/allowed-oidc-clients", updateDto, &group); err != nil {
			return fmt.Errorf("failed to update allowed OIDC clients: %w", err)
		}

		// Output result
		fmt.Printf("Allowed OIDC clients updated successfully for group:\n")
		return outputUserGroupResult(group, globalFlags.format)
	},
}

func init() {
	// Add flags to subcommands
	userGroupsListCmd.Flags().Int("page", 1, "Page number")
	userGroupsListCmd.Flags().Int("limit", 20, "Items per page")
	userGroupsListCmd.Flags().String("sort-by", "", "Sort by column")
	userGroupsListCmd.Flags().String("sort-order", "asc", "Sort order (asc/desc)")
	userGroupsListCmd.Flags().String("search", "", "Search term")

	userGroupsCreateCmd.Flags().String("name", "", "Group name (required, 2-255 chars)")
	userGroupsCreateCmd.Flags().String("friendly-name", "", "Friendly name (required, 2-50 chars)")
	userGroupsCreateCmd.MarkFlagRequired("name")
	userGroupsCreateCmd.MarkFlagRequired("friendly-name")

	userGroupsUpdateCmd.Flags().String("name", "", "Group name (2-255 chars)")
	userGroupsUpdateCmd.Flags().String("friendly-name", "", "Friendly name (2-50 chars)")

	userGroupsDeleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force deletion without confirmation")

	userGroupsUpdateUsersCmd.Flags().StringSlice("user-ids", []string{}, "User IDs to assign to this group (comma-separated)")
	userGroupsUpdateUsersCmd.MarkFlagRequired("user-ids")

	userGroupsUpdateAllowedClientsCmd.Flags().StringSlice("client-ids", []string{}, "OIDC client IDs allowed for this group (comma-separated)")
	userGroupsUpdateAllowedClientsCmd.MarkFlagRequired("client-ids")
}

// outputUserGroupsResult outputs user groups data in the specified format
func outputUserGroupsResult(data PaginatedResponse[dto.UserGroupMinimalDto], format string) error {
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "yaml":
		yamlData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
		return nil
	case "table":
		fmt.Printf("%-36s %-30s %-30s %-10s\n", "ID", "Name", "Friendly Name", "Users")
		fmt.Println(strings.Repeat("-", 110))
		for _, group := range data.Data {
			fmt.Printf("%-36s %-30s %-30s %-10d\n",
				group.ID,
				truncate(group.Name, 28),
				truncate(group.FriendlyName, 28),
				group.UserCount)
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

// outputUserGroupResult outputs user group data in the specified format
func outputUserGroupResult(group dto.UserGroupDto, format string) error {
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(group)
	case "yaml":
		yamlData, err := json.Marshal(group)
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
		return nil
	case "table":
		fmt.Printf("ID:                 %s\n", group.ID)
		fmt.Printf("Name:               %s\n", group.Name)
		fmt.Printf("Friendly Name:      %s\n", group.FriendlyName)
		fmt.Printf("Created At:         %s\n", group.CreatedAt)

		if group.LdapID != nil {
			fmt.Printf("LDAP ID:            %s\n", *group.LdapID)
		}

		if len(group.CustomClaims) > 0 {
			fmt.Printf("\nCustom Claims (%d):\n", len(group.CustomClaims))
			for i, claim := range group.CustomClaims {
				fmt.Printf("  %d. %s: %v\n", i+1, claim.Key, claim.Value)
			}
		}

		if len(group.Users) > 0 {
			fmt.Printf("\nUsers (%d):\n", len(group.Users))
			for i, user := range group.Users {
				fmt.Printf("  %d. %s (%s)\n", i+1, user.DisplayName, user.Username)
			}
		} else {
			fmt.Printf("\nUsers: None\n")
		}

		if len(group.AllowedOidcClients) > 0 {
			fmt.Printf("\nAllowed OIDC Clients (%d):\n", len(group.AllowedOidcClients))
			for i, client := range group.AllowedOidcClients {
				fmt.Printf("  %d. %s (%s)\n", i+1, client.Name, client.ID)
			}
		} else {
			fmt.Printf("\nAllowed OIDC Clients: None\n")
		}

		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
