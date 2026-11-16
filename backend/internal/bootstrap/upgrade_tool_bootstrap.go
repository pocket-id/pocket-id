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
	upgradeToolVersion = "v3.1.0"

	// upgradeToolMaxSize is far above the size of a released file, and a download cut off by it fails the checksum
	upgradeToolMaxSize = 128 << 20

	// legacyMigrationsTable is the version table of golang-migrate, which the upgrade tool drops when it is done
	legacyMigrationsTable = "schema_migrations"
)

// upgradeToolChecksums holds the SHA-256 checksum of each released gzipped binary of upgradeToolVersion, keyed by GOOS/GOARCH
var upgradeToolChecksums = map[string]string{
	"darwin/amd64":  "6836b072aa42f1938eb660d1e2696fc6ff8911216f9f75cc32bba905cef7e8bd",
	"darwin/arm64":  "53bb7c7ea79e512703469253e0f11c18f2464148b7ed03ece75332ff3b6cc2ca",
	"freebsd/amd64": "c22a92da0ca67d4198959a578f445fe369e6745896aa5b83ba90c15390e3913f",
	"freebsd/arm64": "20f5603a1c2f452c9f33e69efc334ac4bc2d22b1e926ec978dc1a5a90d1c8a2d",
	"linux/amd64":   "a04e262a10a854ae7c7c5f7cd62f9cd8188c0f06470bbe7eea90d05f1ac6dec3",
	"linux/arm64":   "1c161806c02d51da6c390090d43e8f61bc13a77aae5223351455c7abd857f072",
	"openbsd/amd64": "a807853aec2259a4088607410f9fb908066844ce99c58a38f9188893c51b6a9e",
	"openbsd/arm64": "9fa78a1d748b5e8b3bf460cbdf568fc3eedeb24cefed8687e5880759810f3ff2",
	"windows/amd64": "e4afbf7ec1648828a0295793104ade172dcd6fa4a2fce5144668e313d7d7a1af",
	"windows/arm64": "5cbcaec83a20af25e4973cfaed7b6082eec9b443edea9c388addd89265b7408d",
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
