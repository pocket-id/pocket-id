package cmds

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// appImagesCmd represents the app-images command
var appImagesCmd = &cobra.Command{
	Use:   "app-images",
	Short: "Manage application images",
	Long:  `Upload, download, and delete application images (logo, background, favicon, etc.).`,
}

func init() {
	// Add app-images command to root
	rootCmd.AddCommand(appImagesCmd)

	// Add subcommands
	appImagesCmd.AddCommand(appImagesUpdateCmd)
	appImagesCmd.AddCommand(appImagesDeleteCmd)
}

// appImagesUpdateCmd represents the update app image command
var appImagesUpdateCmd = &cobra.Command{
	Use:   "update [image-type] [file-path]",
	Short: "Update application image",
	Long: `Update application image by type.

Available image types:
- logo [--light=true|false] - Update logo (light or dark mode)
- email - Update email logo
- background - Update background image
- favicon - Update favicon (.ico file)
- default-profile-picture - Update default profile picture`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		imageType := args[0]
		filePath := args[1]

		// Parse flags
		lightLogo, _ := cmd.Flags().GetBool("light")

		// Open file
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		defer file.Close()

		// Get file info (not used but kept for future validation)
		_, err = file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", filePath)
		}

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create form file field
		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			return fmt.Errorf("failed to create form file: %w", err)
		}

		// Copy file content
		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		// Close writer
		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close multipart writer: %w", err)
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Determine endpoint based on image type
		endpoint, queryParams := getImageEndpoint(imageType, lightLogo)

		// Make request
		_, err = client.Do(ctx, RequestOptions{
			Method: http.MethodPut,
			Path:   endpoint,
			Query:  queryParams,
			Body:   body,
			Headers: map[string]string{
				"Content-Type": writer.FormDataContentType(),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update %s image: %w", imageType, err)
		}

		fmt.Printf("%s image updated successfully\n", imageType)
		return nil
	},
}

// appImagesDeleteCmd represents the delete app image command
var appImagesDeleteCmd = &cobra.Command{
	Use:   "delete [image-type]",
	Short: "Delete application image",
	Long: `Delete application image by type.

Currently only supports:
- default-profile-picture - Delete default profile picture`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		imageType := args[0]

		// Only default-profile-picture can be deleted
		if imageType != "default-profile-picture" {
			return fmt.Errorf("only 'default-profile-picture' can be deleted. Other images can be updated but not deleted.")
		}

		// Create API client
		client := NewAPIClient(globalFlags.endpoint, globalFlags.apiKey)

		// Make request
		if err := client.Delete(ctx, "/api/application-images/default-profile-picture"); err != nil {
			return fmt.Errorf("failed to delete %s image: %w", imageType, err)
		}

		fmt.Printf("%s image deleted successfully\n", imageType)
		return nil
	},
}

// Helper function to get endpoint and query parameters for image type
func getImageEndpoint(imageType string, lightLogo bool) (string, map[string]string) {
	var endpoint string
	queryParams := make(map[string]string)

	switch imageType {
	case "logo":
		endpoint = "/api/application-images/logo"
		if !lightLogo {
			queryParams["light"] = "false"
		}
	case "email":
		endpoint = "/api/application-images/email"
	case "background":
		endpoint = "/api/application-images/background"
	case "favicon":
		endpoint = "/api/application-images/favicon"
	case "default-profile-picture":
		endpoint = "/api/application-images/default-profile-picture"
	default:
		endpoint = "/api/application-images/logo"
	}

	return endpoint, queryParams
}

func init() {
	// Add flags to update command
	appImagesUpdateCmd.Flags().Bool("light", true, "Light mode logo (true) or dark mode logo (false)")
}
