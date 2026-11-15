package bootstrap

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Everything a database of an earlier major version needs before this one can use it lives in a separate program, github.com/pocket-id/upgrade
// This file is the only upgrade code in Pocket ID: when it finds such a database it takes that program from a file next to the Pocket ID binary or downloads it, and runs it only if it is the exact file pinned below

const (
	upgradeToolVersion = "v3.0.0"

	// upgradeToolMaxSize is far above the size of a released file, and a download cut off by it fails the checksum
	upgradeToolMaxSize = 128 << 20

	// legacyMigrationsTable is the version table of golang-migrate, which the upgrade tool drops when it is done
	legacyMigrationsTable = "schema_migrations"
)

// upgradeToolChecksums holds the SHA-256 checksum of each released gzipped binary of upgradeToolVersion, keyed by GOOS/GOARCH
var upgradeToolChecksums = map[string]string{
	"darwin/amd64":  "9245a382fd018278fc57eb301be164bb5bcd743fe15f280f7236f293315560a5",
	"darwin/arm64":  "78bb470b6fc7a4501bcb82d3bb417a23604d42809666ca8d98ec4d67b8f722e4",
	"freebsd/amd64": "678ca25a66746f3d79cfcfe12f099c817e81354688248734f76cc73cc14777fe",
	"freebsd/arm64": "7f0dc1ce9ce848aaed92518bf0f08484dc7b9e38bf936c221cbf8d489f9fbcc9",
	"linux/amd64":   "b7dab47ede9d3e46d65eca4a7865cc5d2f1727352432064d4b9141232bf44c4b",
	"linux/arm64":   "c5e46dab3aa172520b1fb1a98a4a2dfa24704b7718145f0d6f89c2021b7d9bfa",
	"openbsd/amd64": "b712fe5a882c84d5963e47daa6e1b7914c6a020ea927f676ef15239c909107bd",
	"openbsd/arm64": "9320aef0c4cde0fdad135cc2fec4a52c2099e9056f1583fbb1ce87f6f3d2eed9",
	"windows/amd64": "e276b2be084600e9cfdfb12bac08d78e4444d8d5e8fb0d65a118401699583316",
	"windows/arm64": "913808014a67b033d8b7736247422b56e0581ca857a6d64afbd358ee690f1adf",
}

// upgradeLegacyDatabase runs the upgrade tool if the database was last used by Pocket ID v1 or v2
func upgradeLegacyDatabase(ctx context.Context, sqlDb *sql.DB) error {
	// A database created by v3 never had the table, so this is the only thing a regular start does here
	legacy, err := utils.TableExists(ctx, sqlDb, common.EnvConfig.DbProvider, legacyMigrationsTable)
	if err != nil || !legacy {
		return err
	}

	// Build the download URL from compile-time values only, so nothing at runtime can point it elsewhere
	name := "pocket-id-upgrade_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	url := "https://github.com/pocket-id/upgrade/releases/download/" + upgradeToolVersion + "/" + name + ".gz"
	checksum, ok := upgradeToolChecksums[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("the database must be upgraded by the Pocket ID upgrade tool %s, which is not released for %s/%s: build it from https://github.com/pocket-id/upgrade, run it once with the same environment and working directory as Pocket ID, then start Pocket ID again", upgradeToolVersion, runtime.GOOS, runtime.GOARCH)
	}
	// Hosts without internet access provide the tool as a file next to the Pocket ID binary
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the Pocket ID binary: %w", err)
	}
	localPath := filepath.Join(filepath.Dir(executable), name+".gz")
	manualRun := fmt.Sprintf("without internet access, download %s, put it at %s and start Pocket ID again", url, localPath)

	// Load the tool into memory from that file or from GitHub, and keep it there unless it is the pinned file
	compressed, err := os.ReadFile(localPath)
	switch {
	case err == nil:
		slog.InfoContext(ctx, "The database was last used by an earlier major version of Pocket ID, using the tool that upgrades it from next to the Pocket ID binary", slog.String("path", localPath))
		err = verifyChecksum(compressed, checksum)
		if err != nil {
			// The file was put there on purpose, so it is reported instead of being replaced by a download
			return fmt.Errorf("the Pocket ID upgrade tool at %s is not version %s: %w", localPath, upgradeToolVersion, err)
		}
	case errors.Is(err, fs.ErrNotExist):
		slog.InfoContext(ctx, "The database was last used by an earlier major version of Pocket ID, downloading the tool that upgrades it", slog.String("url", url))
		compressed, err = downloadVerified(ctx, url, checksum)
		if err != nil {
			return fmt.Errorf("failed to download the Pocket ID upgrade tool %s: %w; %s", upgradeToolVersion, err, manualRun)
		}
	default:
		return fmt.Errorf("failed to read the Pocket ID upgrade tool at %s: %w", localPath, err)
	}

	// Decompress only after the checksum matched, so no untrusted data reaches the decompressor
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to decompress the Pocket ID upgrade tool %s: %w", upgradeToolVersion, err)
	}
	binary, err := io.ReadAll(gz)
	if err != nil {
		return fmt.Errorf("failed to decompress the Pocket ID upgrade tool %s: %w", upgradeToolVersion, err)
	}

	// Write the verified binary to a directory only this user can access, so what runs is what was verified
	dir, err := os.MkdirTemp("", "pocket-id-upgrade-")
	if err != nil {
		return fmt.Errorf("failed to store the Pocket ID upgrade tool %s: %w; %s", upgradeToolVersion, err, manualRun)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, name)
	err = os.WriteFile(path, binary, 0o700) //nolint:gosec // G306: the file must be executable
	if err != nil {
		return fmt.Errorf("failed to store the Pocket ID upgrade tool %s: %w; %s", upgradeToolVersion, err, manualRun)
	}

	// Run the tool with the environment and working directory of this process, which is where it reads the same configuration from
	cmd := exec.CommandContext(ctx, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// The tool ran and reported what went wrong, so running it by hand would not help
		return fmt.Errorf("the Pocket ID upgrade tool %s failed: %w", upgradeToolVersion, err)
	} else if err != nil {
		return fmt.Errorf("failed to run the Pocket ID upgrade tool %s: %w; %s", upgradeToolVersion, err, manualRun)
	}

	return nil
}

// downloadVerified returns the content at the URL, but only if its SHA-256 checksum is the expected one
func downloadVerified(ctx context.Context, url string, checksum string) ([]byte, error) {
	// The URL is a constant, so this needs none of the guards the clients for admin-controlled URLs have
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %q", resp.Status)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, upgradeToolMaxSize))
	if err != nil {
		return nil, err
	}

	// Anything but the pinned file is discarded, whether the download was cut off or the release was replaced
	err = verifyChecksum(content, checksum)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func verifyChecksum(content []byte, checksum string) error {
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != checksum {
		return fmt.Errorf("it has the SHA-256 checksum %x instead of %s, and was not run", sum, checksum)
	}
	return nil
}
