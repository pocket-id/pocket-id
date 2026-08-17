package cmds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
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
	// The export includes the actor data, which a standalone Francis runtime keeps in its own store rather than in Pocket ID's database
	// Exporting anyway would silently produce an archive missing that data, so refuse instead
	if !common.EnvConfig.HasEmbeddedFrancisRuntime() {
		return errors.New("exporting is not supported when FRANCIS_HOST points to a standalone Francis runtime: export Pocket ID's data and the runtime's data separately, using the runtime's own backup command for the latter")
	}

	db, pg, err := bootstrap.NewDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	storage, err := bootstrap.InitStorage(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Close filesystem storage handles before the command exits
	defer func() {
		_ = storage.Close()
	}()

	// The actor host's data lives outside of the Pocket ID schema, so it's exported through Francis
	providerOpts, err := bootstrap.ActorsProviderOptions(db, pg)
	if err != nil {
		return err
	}
	actorsProvider, err := bootstrap.NewActorsBackupProvider(ctx, providerOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize the actor host's data provider: %w", err)
	}
	defer func() {
		_ = actorsProvider.Close()
	}()

	exportService := service.NewExportService(db, storage, actorsProvider)

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
