package cmds

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/italypaleale/francis/clusteradmin"
	"github.com/italypaleale/francis/components"
	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type importFlags struct {
	Path                  string
	Yes                   bool
	ForcefullyAcquireLock bool
}

func init() {
	var flags importFlags

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Imports all data of Pocket ID from a ZIP file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(cmd.Context(), flags)
		},
	}

	importCmd.Flags().StringVarP(&flags.Path, "path", "p", "pocket-id-export.zip", "Path to the ZIP file to import the data from, or '-' to read from stdin")
	importCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Skip confirmation prompts")
	importCmd.Flags().BoolVarP(&flags.ForcefullyAcquireLock, "forcefully-acquire-lock", "", false, "Forcefully acquire exclusive access by terminating any running Pocket ID instance")

	rootCmd.AddCommand(importCmd)
}

// runImport handles the high-level orchestration of the import process
func runImport(ctx context.Context, flags importFlags) error {
	if !flags.Yes {
		ok, err := askForConfirmation()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !ok {
			fmt.Println("Aborted")
			os.Exit(1)
		}
	}

	var (
		zipReader *zip.ReadCloser
		cleanup   func()
		err       error
	)

	if flags.Path == "-" {
		zipReader, cleanup, err = readZipFromStdin()
		defer cleanup()
	} else {
		zipReader, err = zip.OpenReader(flags.Path)
	}
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	// Connect to the database without running migrations: the import re-creates the Pocket ID schema itself
	db, pg, err := bootstrap.ConnectDatabase(ctx)
	if err != nil {
		return err
	}

	// The cluster admin talks to the same database as the actor host, so build its provider options the same way the host does
	var sqliteDB *sql.DB
	if pg == nil {
		sqliteDB, err = db.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB connection: %w", err)
		}
	}
	providerOpts, err := bootstrap.ActorsProviderOptions(pg, sqliteDB)
	if err != nil {
		return err
	}

	// Take exclusive access to the cluster so no Pocket ID replica is running while we overwrite the database
	release, lost, err := acquireExclusiveAccess(ctx, providerOpts, flags.ForcefullyAcquireLock)
	if err != nil {
		return err
	}
	defer release()

	// Abort the import if exclusive access is lost partway through (for example if the lease can no longer be renewed)
	importCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-lost:
			cancel()
		case <-importCtx.Done():
		}
	}()

	storage, err := bootstrap.InitStorage(importCtx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	importService := service.NewImportService(db, storage)
	err = importService.ImportFromZip(importCtx, &zipReader.Reader)
	if err != nil {
		return fmt.Errorf("failed to import data from zip: %w", err)
	}

	fmt.Println("Import completed successfully.")
	return nil
}

// acquireExclusiveAccess takes an exclusive-access lease on the cluster so the import can safely overwrite the database.
//
// It returns a release function that must be called once the import is done, and a channel that is closed if the lease is lost while it is held.
func acquireExclusiveAccess(ctx context.Context, providerOpts components.ProviderOptions, force bool) (release func(), lost <-chan struct{}, err error) {
	// New initializes the provider, applying the actor host's schema migrations, so this also works against a brand-new (empty) database
	admin, err := clusteradmin.New(ctx, providerOpts, clusteradmin.Options{
		// Match the actor host so the admin waits the right amount of time for hosts to drain
		HostHealthCheckDeadline: bootstrap.ActorsHostHealthCheckDeadline(common.EnvConfig.HAEnabled),
		Logger:                  slog.Default(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cluster admin: %w", err)
	}

	lost, err = admin.AcquireExclusive(ctx, clusteradmin.AcquireOptions{Force: force})
	if err != nil {
		_ = admin.Close()
		switch {
		case errors.Is(err, components.ErrHostsConnected):
			//nolint:staticcheck
			return nil, nil, errors.New("Pocket ID must be stopped before importing data; please stop the running instance or run with --forcefully-acquire-lock to terminate the other instance")
		case errors.Is(err, components.ErrExclusiveHeld):
			return nil, nil, errors.New("another exclusive operation, such as another import, is already in progress; please wait for it to complete and try again")
		default:
			return nil, nil, fmt.Errorf("failed to acquire exclusive access: %w", err)
		}
	}

	release = func() {
		// The import preserves the actor host's "francis_" tables, including the lease row, so the lease must be released explicitly
		// Detach from ctx so the release still runs even if the import was canceled
		releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancelRelease()
		if rerr := admin.ReleaseExclusive(releaseCtx); rerr != nil {
			slog.WarnContext(ctx, "Failed to release exclusive access", slog.Any("error", rerr))
		}
		_ = admin.Close()
	}
	return release, lost, nil
}

func askForConfirmation() (bool, error) {
	fmt.Println("WARNING: This feature is experimental and may not work correctly. Please create a backup before proceeding and report any issues you encounter.")
	fmt.Println()
	fmt.Println("WARNING: Import will erase all existing data at the following locations:")
	fmt.Printf("Database:      %s\n", absolutePathOrOriginal(common.EnvConfig.DbConnectionString))
	fmt.Printf("Uploads Path:  %s\n", absolutePathOrOriginal(common.EnvConfig.UploadPath))

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

func readZipFromStdin() (*zip.ReadCloser, func(), error) {
	tmpFile, err := os.CreateTemp("", "pocket-id-import-*.zip")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpFile.Name())
	}

	if _, err := io.Copy(tmpFile, os.Stdin); err != nil {
		tmpFile.Close()
		cleanup()
		return nil, nil, fmt.Errorf("failed to read data from stdin: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	return r, cleanup, nil
}
