package cmds

import (
	"context"
	"fmt"
	"io"
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
			return runExport(cmd.Context(), flags)
		},
	}

	exportCmd.Flags().StringVarP(&flags.Path, "path", "p", "pocket-id-export.zip", "Path to the ZIP file to export the data to, or '-' to write to stdout")

	rootCmd.AddCommand(exportCmd)
}

// runExport orchestrates the export flow
func runExport(ctx context.Context, flags exportFlags) error {
	db, err := bootstrap.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	storage, err := bootstrap.InitStorage(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	exportService := service.NewExportService(db, storage)

	var w io.Writer
	if flags.Path == "-" {
		w = os.Stdout
	} else {
		file, err := os.Create(flags.Path)
		if err != nil {
			return fmt.Errorf("failed to create export file: %w", err)
		}
		defer file.Close()

		w = file
	}

	if err := exportService.ExportToZip(ctx, w); err != nil {
		return fmt.Errorf("failed to export data: %w", err)
	}

	if flags.Path != "-" {
		fmt.Printf("Exported data to %s\n", flags.Path)
	}

	return nil
}
