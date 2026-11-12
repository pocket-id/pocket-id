package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	db, pg, err := bootstrap.NewDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Close the database last, since the storage and the actor provider still use it, and Windows can't delete a SQLite file that's still open
	defer func() {
		_ = sqlDB.Close()
	}()

	storage, err := bootstrap.InitStorage(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Close filesystem storage handles before the command exits
	defer func() {
		_ = storage.Close()
	}()

	// The actor data lives outside of the Pocket ID schema, so it's exported through Francis
	// A standalone runtime keeps it in its own store, out of reach from here, and the export service leaves the entry out of the archive when no provider is passed
	var actorsProvider service.ActorsBackupProvider
	if common.EnvConfig.HasEmbeddedFrancisRuntime() {
		providerOpts, err := bootstrap.ActorsProviderOptions(db, pg)
		if err != nil {
			return err
		}

		provider, err := bootstrap.NewActorsBackupProvider(ctx, providerOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize the actor host's data provider: %w", err)
		}
		defer func() {
			_ = provider.Close()
		}()

		actorsProvider = provider
	} else {
		printRemoteActorDataNotice("The actor data is NOT included in this export", "back it up separately with: francis runtime backup -f actors.bin")
	}

	exportService := service.NewExportService(db, storage, actorsProvider)

	if flags.Path == "-" {
		err = exportService.ExportToZip(ctx, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to export data: %w", err)
		}
		return nil
	}

	// The archive is written to a temporary file that replaces the target only once it's complete, so a failed export keeps an earlier archive intact
	file, err := os.CreateTemp(filepath.Dir(flags.Path), ".pocket-id-export-*")
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	err = exportService.ExportToZip(ctx, file)
	if err != nil {
		return fmt.Errorf("failed to export data: %w", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	err = os.Rename(file.Name(), flags.Path)
	if err != nil {
		return fmt.Errorf("failed to move export file into place: %w", err)
	}

	fmt.Printf("Exported data to %s\n", flags.Path)
	return nil
}
