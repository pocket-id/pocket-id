package cmds

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

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

	importCmd.Flags().StringVarP(&flags.Path, "path", "p", "pocket-id-export.zip", "Path to the ZIP file to import the data from")
	importCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Skip confirmation prompts")
	importCmd.Flags().BoolVarP(&flags.ForcefullyAcquireLock, "forcefully-acquire-lock", "", false, "Forcefully acquire the application lock by terminating the Pocket ID instance")

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

	err = acquireImportLock(ctx, db, flags.ForcefullyAcquireLock)
	if err != nil {
		return err
	}

	storage, err := bootstrap.InitStorage(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	importService := service.NewImportService(db, storage)
	err = importService.ImportFromZip(ctx, &r.Reader)
	if err != nil {
		return fmt.Errorf("failed to import data from zip: %w", err)
	}

	fmt.Println("Import completed successfully.")
	return nil
}

func acquireImportLock(ctx context.Context, db *gorm.DB, force bool) error {
	// Check if the kv table exists, in case we are starting from an empty database
	exists, err := utils.DBTableExists(db, "kv")
	if err != nil {
		return fmt.Errorf("failed to check if kv table exists: %w", err)
	}
	if !exists {
		// This either means the database is empty, or the import is into an old version of PocketID that doesn't support locks
		// In either case, there's no lock to acquire
		fmt.Println("Could not acquire a lock because the 'kv' table does not exist. This is fine if you're importing into a new database, but make sure that there isn't an instance of Pocket ID currently running and using the same database.")
		return nil
	}

	// Note that we do not call a deferred Release if the data was imported
	// This is because we are overriding the contents of the database, so the lock is automatically lost
	appLockService := service.NewAppLockService(db)

	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	waitUntil, err := appLockService.Acquire(opCtx, force)
	if err != nil {
		if errors.Is(err, service.ErrLockUnavailable) {
			//nolint:staticcheck
			return errors.New("Pocket ID must be stopped before importing data; please stop the running instance or run with --forcefully-acquire-lock to terminate the other instance")
		}
		return fmt.Errorf("failed to acquire application lock: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(waitUntil)):
	}

	return nil
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
