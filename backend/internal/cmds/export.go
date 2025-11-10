package cmds

import (
	"fmt"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/spf13/cobra"
)

type exportFlags struct {
	Path string
}

func init() {
	var flags exportFlags

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exports all data of Pocket ID into a ZIP file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(flags)
		},
	}

	exportCmd.Flags().StringVarP(&flags.Path, "path", "p", "pocket-id-export.zip", "Path to the ZIP file to export the data to")

	rootCmd.AddCommand(exportCmd)
}

// runExport orchestrates the export flow
func runExport(flags exportFlags) error {
	db, err := bootstrap.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	exportService := service.NewExportService(db)

	var w *os.File
	w, err = os.Create(flags.Path)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer w.Close()

	if err := exportService.ExportToZip(w); err != nil {
		return fmt.Errorf("failed to export data: %w", err)
	}

	fmt.Printf("Exported data to %s\n", flags.Path)
	return nil
}
