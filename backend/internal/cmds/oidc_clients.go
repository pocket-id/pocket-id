package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// oidcClientsCmd represents the oidc-clients command
var oidcClientsCmd = &cobra.Command{
	Use:   "oidc-clients",
	Short: "Manage OIDC clients",
	Long:  `Create, list, update, and delete OIDC clients.`,
}

func init() {
	// Add oidc-clients command to root
	rootCmd.AddCommand(oidcClientsCmd)

	// Add subcommands
	oidcClientsCmd.AddCommand(oidcClientsListCmd)
	oidcClientsCmd.AddCommand(oidcClientsGetCmd)
	oidcClientsCmd.AddCommand(oidcClientsCreateCmd)
	oidcClientsCmd.AddCommand(oidcClientsUpdateCmd)
	oidcClientsCmd.AddCommand(oidcClientsDeleteCmd)
	oidcClientsCmd.AddCommand(oidcClientsCreateSecretCmd)
	oidcClientsCmd.AddCommand(oidcClientsUpdateAllowedGroupsCmd)
}

// oidcClientsListCmd represents the list OIDC clients command
var oidcClientsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OIDC clients",
	Long:  `Get a paginated list of OIDC clients.`,
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
		var response PaginatedResponse[dto.OidcClientWithAllowedGroupsCountDto]
		if err := client.Get(ctx, "/api/oidc/clients", &response, query); err != nil {
			return fmt.Errorf("failed to list OIDC clients: %w", err)
		}

		// Output result
		return outputOidcClientsResult(response, globalFlags.format)
	},
}

// oidcClientsGetCmd represents the get OIDC client command
var oidcClientsGetCmd = &cobra.Command{
	Use:   "get [client-id]",
	Short: "Get an OIDC client by ID",
	Long:  `Get detailed information about a specific OIDC client.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var clientDto dto.OidcClientWithAllowedUserGroupsDto
		if err := client.Get(ctx, "/api/oidc/clients/"+clientID, &clientDto, nil); err != nil {
			return fmt.Errorf("failed to get OIDC client: %w", err)
		}

		// Output result
		return outputOidcClientResult(clientDto, globalFlags.format)
	},
}

// oidcClientsCreateCmd represents the create OIDC client command
var oidcClientsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new OIDC client",
	Long:  `Create a new OIDC client with the specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Parse flags
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		callbackURLs, _ := cmd.Flags().GetStringSlice("callback-urls")
		logoutCallbackURLs, _ := cmd.Flags().GetStringSlice("logout-callback-urls")
		isPublic, _ := cmd.Flags().GetBool("public")
		pkceEnabled, _ := cmd.Flags().GetBool("pkce-enabled")
		requiresReauth, _ := cmd.Flags().GetBool("requires-reauth")
		launchURL, _ := cmd.Flags().GetString("launch-url")
		isGroupRestricted, _ := cmd.Flags().GetBool("group-restricted")

		// Validate required fields
		if name == "" {
			return fmt.Errorf("name is required")
		}
		if len(callbackURLs) == 0 {
			return fmt.Errorf("at least one callback URL is required")
		}

		// Prepare request body
		createDto := dto.OidcClientCreateDto{
			OidcClientUpdateDto: dto.OidcClientUpdateDto{
				Name:                     name,
				CallbackURLs:             callbackURLs,
				LogoutCallbackURLs:       logoutCallbackURLs,
				IsPublic:                 isPublic,
				PkceEnabled:              pkceEnabled,
				RequiresReauthentication: requiresReauth,
				IsGroupRestricted:        isGroupRestricted,
			},
		}

		// Set optional fields
		if id != "" {
			createDto.ID = id
		}
		if launchURL != "" {
			createDto.LaunchURL = &launchURL
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var clientDto dto.OidcClientWithAllowedUserGroupsDto
		if err := client.Post(ctx, "/api/oidc/clients", createDto, &clientDto); err != nil {
			return fmt.Errorf("failed to create OIDC client: %w", err)
		}

		// Output result
		fmt.Printf("OIDC client created successfully:\n")
		return outputOidcClientResult(clientDto, globalFlags.format)
	},
}

// oidcClientsUpdateCmd represents the update OIDC client command
var oidcClientsUpdateCmd = &cobra.Command{
	Use:   "update [client-id]",
	Short: "Update an OIDC client",
	Long:  `Update an existing OIDC client's details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientID := args[0]

		// Parse flags
		name, _ := cmd.Flags().GetString("name")
		callbackURLs, _ := cmd.Flags().GetStringSlice("callback-urls")
		logoutCallbackURLs, _ := cmd.Flags().GetStringSlice("logout-callback-urls")
		isPublic, _ := cmd.Flags().GetBool("public")
		pkceEnabled, _ := cmd.Flags().GetBool("pkce-enabled")
		requiresReauth, _ := cmd.Flags().GetBool("requires-reauth")
		launchURL, _ := cmd.Flags().GetString("launch-url")
		isGroupRestricted, _ := cmd.Flags().GetBool("group-restricted")

		// Get existing client first
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)
		var existingClient dto.OidcClientWithAllowedUserGroupsDto
		if err := client.Get(ctx, "/api/oidc/clients/"+clientID, &existingClient, nil); err != nil {
			return fmt.Errorf("failed to get existing OIDC client: %w", err)
		}

		// Prepare update body (only include fields that were explicitly set)
		updateBody := make(map[string]interface{})

		if cmd.Flags().Changed("name") {
			updateBody["name"] = name
		}
		if cmd.Flags().Changed("callback-urls") {
			updateBody["callbackURLs"] = callbackURLs
		}
		if cmd.Flags().Changed("logout-callback-urls") {
			updateBody["logoutCallbackURLs"] = logoutCallbackURLs
		}
		if cmd.Flags().Changed("public") {
			updateBody["isPublic"] = isPublic
		}
		if cmd.Flags().Changed("pkce-enabled") {
			updateBody["pkceEnabled"] = pkceEnabled
		}
		if cmd.Flags().Changed("requires-reauth") {
			updateBody["requiresReauthentication"] = requiresReauth
		}
		if cmd.Flags().Changed("launch-url") {
			if launchURL == "" {
				updateBody["launchURL"] = nil
			} else {
				updateBody["launchURL"] = launchURL
			}
		}
		if cmd.Flags().Changed("group-restricted") {
			updateBody["isGroupRestricted"] = isGroupRestricted
		}

		// If no fields were updated, return
		if len(updateBody) == 0 {
			fmt.Println("No fields to update")
			return nil
		}

		// Make update request
		var updatedClient dto.OidcClientWithAllowedUserGroupsDto
		if err := client.Put(ctx, "/api/oidc/clients/"+clientID, updateBody, &updatedClient); err != nil {
			return fmt.Errorf("failed to update OIDC client: %w", err)
		}

		// Output result
		fmt.Printf("OIDC client updated successfully:\n")
		return outputOidcClientResult(updatedClient, globalFlags.format)
	},
}

// oidcClientsDeleteCmd represents the delete OIDC client command
var oidcClientsDeleteCmd = &cobra.Command{
	Use:   "delete [client-id]",
	Short: "Delete an OIDC client",
	Long:  `Delete an OIDC client by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientID := args[0]

		// Confirm deletion
		if !forceDelete {
			fmt.Printf("Are you sure you want to delete OIDC client %s? (y/N): ", clientID)
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
		if err := client.Delete(ctx, "/api/oidc/clients/"+clientID); err != nil {
			return fmt.Errorf("failed to delete OIDC client: %w", err)
		}

		fmt.Printf("OIDC client %s deleted successfully\n", clientID)
		return nil
	},
}

// oidcClientsCreateSecretCmd represents the create client secret command
var oidcClientsCreateSecretCmd = &cobra.Command{
	Use:   "create-secret [client-id]",
	Short: "Create a new secret for an OIDC client",
	Long:  `Generate a new secret for an OIDC client.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientID := args[0]

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var response struct {
			Secret string `json:"secret"`
		}
		if err := client.Post(ctx, "/api/oidc/clients/"+clientID+"/secret", nil, &response); err != nil {
			return fmt.Errorf("failed to create client secret: %w", err)
		}

		// Output result
		switch strings.ToLower(globalFlags.format) {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(response)
		case "yaml":
			yamlData, err := json.Marshal(response)
			if err != nil {
				return err
			}
			fmt.Println(string(yamlData))
			return nil
		case "table":
			fmt.Printf("Client ID: %s\n", clientID)
			fmt.Printf("New Secret: %s\n", response.Secret)
			fmt.Println("\nIMPORTANT: Save this secret securely. It will not be shown again.")
			return nil
		default:
			return fmt.Errorf("unsupported format: %s", globalFlags.format)
		}
	},
}

// oidcClientsUpdateAllowedGroupsCmd represents the update allowed groups command
var oidcClientsUpdateAllowedGroupsCmd = &cobra.Command{
	Use:   "update-allowed-groups [client-id]",
	Short: "Update allowed user groups for an OIDC client",
	Long:  `Update the user groups allowed to access an OIDC client.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientID := args[0]

		// Parse flags
		groupIDs, _ := cmd.Flags().GetStringSlice("group-ids")

		// Prepare request body
		updateDto := dto.OidcUpdateAllowedUserGroupsDto{
			UserGroupIDs: groupIDs,
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var clientDto dto.OidcClientWithAllowedUserGroupsDto
		if err := client.Put(ctx, "/api/oidc/clients/"+clientID+"/allowed-user-groups", updateDto, &clientDto); err != nil {
			return fmt.Errorf("failed to update allowed groups: %w", err)
		}

		// Output result
		fmt.Printf("Allowed groups updated successfully:\n")
		return outputOidcClientResult(clientDto, globalFlags.format)
	},
}

// Global flag for force deletion is declared in users.go and shared across commands

func init() {
	// Add flags to subcommands
	oidcClientsListCmd.Flags().Int("page", 1, "Page number")
	oidcClientsListCmd.Flags().Int("limit", 20, "Items per page")
	oidcClientsListCmd.Flags().String("sort-by", "", "Sort by column")
	oidcClientsListCmd.Flags().String("sort-order", "asc", "Sort order (asc/desc)")
	oidcClientsListCmd.Flags().String("search", "", "Search term")

	oidcClientsCreateCmd.Flags().String("id", "", "Client ID (optional, auto-generated if not provided)")
	oidcClientsCreateCmd.Flags().String("name", "", "Client name (required)")
	oidcClientsCreateCmd.Flags().StringSlice("callback-urls", []string{}, "Callback URLs (required, comma-separated)")
	oidcClientsCreateCmd.Flags().StringSlice("logout-callback-urls", []string{}, "Logout callback URLs (comma-separated)")
	oidcClientsCreateCmd.Flags().Bool("public", false, "Make client public (no authentication required)")
	oidcClientsCreateCmd.Flags().Bool("pkce-enabled", true, "Enable PKCE (Proof Key for Code Exchange)")
	oidcClientsCreateCmd.Flags().Bool("requires-reauth", false, "Require reauthentication for each authorization")
	oidcClientsCreateCmd.Flags().String("launch-url", "", "Launch URL for the client")
	oidcClientsCreateCmd.Flags().Bool("group-restricted", false, "Restrict access to specific user groups")
	oidcClientsCreateCmd.MarkFlagRequired("name")
	oidcClientsCreateCmd.MarkFlagRequired("callback-urls")

	oidcClientsUpdateCmd.Flags().String("name", "", "Client name")
	oidcClientsUpdateCmd.Flags().StringSlice("callback-urls", []string{}, "Callback URLs (comma-separated)")
	oidcClientsUpdateCmd.Flags().StringSlice("logout-callback-urls", []string{}, "Logout callback URLs (comma-separated)")
	oidcClientsUpdateCmd.Flags().Bool("public", false, "Make client public (no authentication required)")
	oidcClientsUpdateCmd.Flags().Bool("pkce-enabled", true, "Enable PKCE (Proof Key for Code Exchange)")
	oidcClientsUpdateCmd.Flags().Bool("requires-reauth", false, "Require reauthentication for each authorization")
	oidcClientsUpdateCmd.Flags().String("launch-url", "", "Launch URL for the client")
	oidcClientsUpdateCmd.Flags().Bool("group-restricted", false, "Restrict access to specific user groups")

	oidcClientsDeleteCmd.Flags().BoolVar(&forceDelete, "force", false, "Force deletion without confirmation")

	oidcClientsUpdateAllowedGroupsCmd.Flags().StringSlice("group-ids", []string{}, "User group IDs allowed to access this client (comma-separated)")
	oidcClientsUpdateAllowedGroupsCmd.MarkFlagRequired("group-ids")
}

// outputOidcClientsResult outputs OIDC clients data in the specified format
func outputOidcClientsResult(data PaginatedResponse[dto.OidcClientWithAllowedGroupsCountDto], format string) error {
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
		fmt.Printf("%-36s %-30s %-20s %-10s %-10s %-15s\n", "ID", "Name", "Callback URLs", "Public", "PKCE", "Allowed Groups")
		fmt.Println(strings.Repeat("-", 140))
		for _, client := range data.Data {
			callbackURLs := "None"
			if len(client.CallbackURLs) > 0 {
				callbackURLs = truncate(client.CallbackURLs[0], 18)
				if len(client.CallbackURLs) > 1 {
					callbackURLs += fmt.Sprintf(" (+%d more)", len(client.CallbackURLs)-1)
				}
			}
			publicStatus := "No"
			if client.IsPublic {
				publicStatus = "Yes"
			}
			pkceStatus := "No"
			if client.PkceEnabled {
				pkceStatus = "Yes"
			}
			fmt.Printf("%-36s %-30s %-20s %-10s %-10s %-15d\n",
				client.ID,
				truncate(client.Name, 28),
				callbackURLs,
				publicStatus,
				pkceStatus,
				client.AllowedUserGroupsCount)
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

// outputOidcClientResult outputs OIDC client data in the specified format
func outputOidcClientResult(client dto.OidcClientWithAllowedUserGroupsDto, format string) error {
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(client)
	case "yaml":
		yamlData, err := json.Marshal(client)
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
		return nil
	case "table":
		fmt.Printf("ID:                           %s\n", client.ID)
		fmt.Printf("Name:                         %s\n", client.Name)
		fmt.Printf("Public:                       %v\n", client.IsPublic)
		fmt.Printf("PKCE Enabled:                 %v\n", client.PkceEnabled)
		fmt.Printf("Requires Reauthentication:    %v\n", client.RequiresReauthentication)
		fmt.Printf("Group Restricted:             %v\n", client.IsGroupRestricted)

		if client.LaunchURL != nil {
			fmt.Printf("Launch URL:                   %s\n", *client.LaunchURL)
		}

		fmt.Printf("\nCallback URLs:\n")
		for i, url := range client.CallbackURLs {
			fmt.Printf("  %d. %s\n", i+1, url)
		}

		if len(client.LogoutCallbackURLs) > 0 {
			fmt.Printf("\nLogout Callback URLs:\n")
			for i, url := range client.LogoutCallbackURLs {
				fmt.Printf("  %d. %s\n", i+1, url)
			}
		}

		if len(client.AllowedUserGroups) > 0 {
			fmt.Printf("\nAllowed User Groups (%d):\n", len(client.AllowedUserGroups))
			for i, group := range client.AllowedUserGroups {
				fmt.Printf("  %d. %s (%s)\n", i+1, group.Name, group.ID)
			}
		} else {
			fmt.Printf("\nAllowed User Groups: None\n")
		}

		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
