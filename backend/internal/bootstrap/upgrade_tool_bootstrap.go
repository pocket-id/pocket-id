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
	upgradeToolVersion = "v3.2.0"

	// upgradeToolMaxSize is far above the size of a released file, and a download cut off by it fails the checksum
	upgradeToolMaxSize = 128 << 20

	// legacyMigrationsTable is the version table of golang-migrate, which the upgrade tool drops when it is done
	legacyMigrationsTable = "schema_migrations"
)

// upgradeToolChecksums holds the SHA-256 checksum of each released gzipped binary of upgradeToolVersion, keyed by GOOS/GOARCH
var upgradeToolChecksums = map[string]string{
	"darwin/amd64":  "08dd022d30914c75e5e21237d462db4c240a011ab025e695e1b00503643905c2",
	"darwin/arm64":  "80a4b93f978cdebfd7906363088ac862825e2333a514b924eb0716821e3f945d",
	"freebsd/amd64": "d20926c7979b0c8afaab47e639606de7f748f5190cc41b5129752bb75b9e20f5",
	"freebsd/arm64": "c8a8c910da412dd81bc835b40b1e63fa8949265eb1b5a856639494eb656dfcaf",
	"linux/amd64":   "09d732917e0998a4246ed47ea27a0c8f5998ab5fcde67a7e81b0b32e8d4f1a6f",
	"linux/arm64":   "587e73deab2820c1189031fa57c445036d0a7b6e1b5d58b7e6f892dcc8787e92",
	"openbsd/amd64": "ad031bce20eca91f5ee9bf16ae6d2d67e38fbe0e621a4b5b868f3f41d3719e66",
	"openbsd/arm64": "1cf0c38fe94e7d707569d03096defab2ae1004cbfd98c1910755eff9e6960224",
	"windows/amd64": "a69052d886e4e8b5f58e54744dd5dadb68d2ed3cb244d1c37a676b2c246dd98a",
	"windows/arm64": "0ed19b6e3af5a772858fdc2810f273dd6b04c685fb2337f6cc32ab4d7c500aae",
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
