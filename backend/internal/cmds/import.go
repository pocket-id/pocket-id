package cmds

import (
	"archive/zip"
	"fmt"
	"path/filepath"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/spf13/cobra"
)

type importFlags struct {
	Path string
	Yes  bool
}

func init() {
	var flags importFlags

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Imports all data of Pocket ID from a ZIP file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(flags)
		},
	}

	importCmd.Flags().StringVarP(&flags.Path, "path", "p", "./pocket-id-export.zip", "Path to the ZIP file to import the data from")
	importCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Skip confirmation prompts")

	rootCmd.AddCommand(importCmd)
}

// runImport handles the high-level orchestration of the import process
func runImport(flags importFlags) error {
	if !flags.Yes {
		ok, err := askForConfirmation()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !ok {
			fmt.Println("Aborted")
			return nil
		}
	}

	r, err := zip.OpenReader(flags.Path)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	db, err := bootstrap.ConnectDatabase()
	if err != nil {
		return err
	}

	importService := service.NewImportService(db)
	if err := importService.ImportFromZip(&r.Reader); err != nil {
		return err
	}

	fmt.Println("Import completed successfully.")
	return nil
}

func askForConfirmation() (bool, error) {
	fmt.Println("WARNING: This feature is experimental and may not work correctly. Please create a backup before proceeding and report any issues you encounter.")
	fmt.Println()
	fmt.Println("WARNING: Import will erase all existing data at the following locations:")
	fmt.Printf("Database:      %s\n", absolutePathOrOriginal(common.EnvConfig.DbConnectionString))
	fmt.Printf("Uploads Path:  %s\n", absolutePathOrOriginal(common.EnvConfig.UploadPath))
	fmt.Printf("Keys Path:     %s\n", absolutePathOrOriginal(common.EnvConfig.KeysPath))

	ok, err := utils.PromptForConfirmation("Do you want to continue?")
	if err != nil {
		return false, err
	}

	return ok, nil
}

// absolutePathOrOriginal returns the absolute path of the given path, or the original if it fails
func absolutePathOrOriginal(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
