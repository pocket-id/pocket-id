package cmds

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/spf13/cobra"
)

// customClaimsCmd represents the custom-claims command
var customClaimsCmd = &cobra.Command{
	Use:   "custom-claims",
	Short: "Manage custom claims",
	Long:  `Manage custom claims for users and user groups.`,
}

func init() {
	// Add custom-claims command to root
	rootCmd.AddCommand(customClaimsCmd)

	// Add subcommands
	customClaimsCmd.AddCommand(customClaimsSuggestionsCmd)
	customClaimsCmd.AddCommand(customClaimsUpdateUserCmd)
	customClaimsCmd.AddCommand(customClaimsUpdateUserGroupCmd)
}

// customClaimsSuggestionsCmd represents the get suggestions command
var customClaimsSuggestionsCmd = &cobra.Command{
	Use:   "suggestions",
	Short: "Get custom claim suggestions",
	Long:  `Get a list of suggested custom claim names.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var suggestions []string
		if err := client.Get(ctx, "/api/custom-claims/suggestions", &suggestions, nil); err != nil {
			return fmt.Errorf("failed to get custom claim suggestions: %w", err)
		}

		// Output result
		return outputResult(suggestions, globalFlags.format)
	},
}

// customClaimsUpdateUserCmd represents the update custom claims for user command
var customClaimsUpdateUserCmd = &cobra.Command{
	Use:   "update-user [user-id]",
	Short: "Update custom claims for a user",
	Long:  `Update or create custom claims for a specific user.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userID := args[0]

		// Parse flags
		filePath, _ := cmd.Flags().GetString("file")
		claimsJSON, _ := cmd.Flags().GetString("claims")

		var input []dto.CustomClaimCreateDto

		// Read from file if provided
		if filePath != "" {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read claims file: %w", err)
			}

			if err := json.Unmarshal(fileData, &input); err != nil {
				return fmt.Errorf("failed to parse claims file: %w", err)
			}
		} else if claimsJSON != "" {
			// Parse JSON string
			if err := json.Unmarshal([]byte(claimsJSON), &input); err != nil {
				return fmt.Errorf("failed to parse claims JSON: %w", err)
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
				return fmt.Errorf("either --file or --claims flag must be provided")
			}
		}

		// Validate input
		if len(input) == 0 {
			return fmt.Errorf("no custom claims provided")
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var claims []dto.CustomClaimDto
		if err := client.Put(ctx, "/api/custom-claims/user/"+userID, &claims, input); err != nil {
			return fmt.Errorf("failed to update custom claims for user: %w", err)
		}

		// Output result
		return outputResult(claims, globalFlags.format)
	},
}

// customClaimsUpdateUserGroupCmd represents the update custom claims for user group command
var customClaimsUpdateUserGroupCmd = &cobra.Command{
	Use:   "update-user-group [user-group-id]",
	Short: "Update custom claims for a user group",
	Long:  `Update or create custom claims for a specific user group.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userGroupID := args[0]

		// Parse flags
		filePath, _ := cmd.Flags().GetString("file")
		claimsJSON, _ := cmd.Flags().GetString("claims")

		var input []dto.CustomClaimCreateDto

		// Read from file if provided
		if filePath != "" {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read claims file: %w", err)
			}

			if err := json.Unmarshal(fileData, &input); err != nil {
				return fmt.Errorf("failed to parse claims file: %w", err)
			}
		} else if claimsJSON != "" {
			// Parse JSON string
			if err := json.Unmarshal([]byte(claimsJSON), &input); err != nil {
				return fmt.Errorf("failed to parse claims JSON: %w", err)
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
				return fmt.Errorf("either --file or --claims flag must be provided")
			}
		}

		// Validate input
		if len(input) == 0 {
			return fmt.Errorf("no custom claims provided")
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		var claims []dto.CustomClaimDto
		if err := client.Put(ctx, "/api/custom-claims/user-group/"+userGroupID, &claims, input); err != nil {
			return fmt.Errorf("failed to update custom claims for user group: %w", err)
		}

		// Output result
		return outputResult(claims, globalFlags.format)
	},
}

func init() {
	// Add flags to update-user command
	customClaimsUpdateUserCmd.Flags().String("file", "", "Path to JSON file containing custom claims")
	customClaimsUpdateUserCmd.Flags().String("claims", "", "JSON string containing custom claims")

	// Add flags to update-user-group command
	customClaimsUpdateUserGroupCmd.Flags().String("file", "", "Path to JSON file containing custom claims")
	customClaimsUpdateUserGroupCmd.Flags().String("claims", "", "JSON string containing custom claims")
}
