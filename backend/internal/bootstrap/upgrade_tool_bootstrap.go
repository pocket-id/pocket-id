package bootstrap

import (
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
// This file is the only upgrade code in Pocket ID: when it finds such a database it takes that program from a file next to the Pocket ID binary or downloads it, and runs it only if it is the exact binary pinned below

const (
	upgradeToolVersion = "v3.0.0"

	// upgradeToolMaxSize is far above the size of a released binary, and a download cut off by it fails the checksum
	upgradeToolMaxSize = 128 << 20

	// legacyMigrationsTable is the version table of golang-migrate, which the upgrade tool drops when it is done
	legacyMigrationsTable = "schema_migrations"
)

// upgradeToolChecksums holds the SHA-256 checksum of each released binary of upgradeToolVersion, keyed by GOOS/GOARCH
var upgradeToolChecksums = map[string]string{
	"darwin/amd64":  "501d0a3e991d739171bb3e47bdfa1ad3563bf400d955f267c8955cc025125997",
	"darwin/arm64":  "168763064c73715cf958e42ed8bd5c9eecb3aa9963a8607f7b2f35de5688411f",
	"freebsd/amd64": "d6cfd78f12ae502f3d9174f43d17f1bdbba9346724b177b235a4e430953d28d2",
	"freebsd/arm64": "4085d79d41b5e08228cea02334a35649cefa3fe663925f234668c7f0dadb59a8",
	"linux/amd64":   "460ca530b468924afaf9582b04a59508d569f173c45e7526dde93a952bea2ff6",
	"linux/arm64":   "1b5fea8d9e5f9c3f1cd73e294a677149a9c2b44afd39fe8aad7ea276d9836915",
	"openbsd/amd64": "64802a992b2c0ba34bfdafdb747881ad1c7ce74bdb7281eb2f3990dc503536ae",
	"openbsd/arm64": "24fc8b8b85c0a167dd6a8b032bd55b9bd5474fabb7c89dbd568aea76aeb715f3",
	"windows/amd64": "f8d832ae2b20c678804ac219930310c288396e143c3538bbdecc6382eff0a6b4",
	"windows/arm64": "e522d0aedbb797034f7d9b162ca0afcbe18684713846a437d70fab7cb24b7d04",
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
	url := "https://github.com/pocket-id/upgrade/releases/download/" + upgradeToolVersion + "/" + name
	checksum, ok := upgradeToolChecksums[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("the database must be upgraded by the Pocket ID upgrade tool %s, which is not released for %s/%s: build it from https://github.com/pocket-id/upgrade, run it once with the same environment and working directory as Pocket ID, then start Pocket ID again", upgradeToolVersion, runtime.GOOS, runtime.GOARCH)
	}
	// Hosts without internet access provide the tool as a file next to the Pocket ID binary
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the Pocket ID binary: %w", err)
	}
	localPath := filepath.Join(filepath.Dir(executable), name)
	manualRun := fmt.Sprintf("without internet access, download %s, put it at %s and start Pocket ID again", url, localPath)

	// Load the tool into memory from that file or from GitHub, and keep it there unless it is the pinned binary
	binary, err := os.ReadFile(localPath)
	switch {
	case err == nil:
		slog.InfoContext(ctx, "The database was last used by an earlier major version of Pocket ID, using the tool that upgrades it from next to the Pocket ID binary", slog.String("path", localPath))
		err = verifyChecksum(binary, checksum)
		if err != nil {
			// The file was put there on purpose, so it is reported instead of being replaced by a download
			return fmt.Errorf("the Pocket ID upgrade tool at %s is not version %s: %w", localPath, upgradeToolVersion, err)
		}
	case errors.Is(err, fs.ErrNotExist):
		slog.InfoContext(ctx, "The database was last used by an earlier major version of Pocket ID, downloading the tool that upgrades it", slog.String("url", url))
		binary, err = downloadVerified(ctx, url, checksum)
		if err != nil {
			return fmt.Errorf("failed to download the Pocket ID upgrade tool %s: %w; %s", upgradeToolVersion, err, manualRun)
		}
	default:
		return fmt.Errorf("failed to read the Pocket ID upgrade tool at %s: %w", localPath, err)
	}

	// Write the verified bytes to a directory only this user can access, so what runs is what was verified
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

	// Anything but the pinned binary is discarded, whether the download was cut off or the release was replaced
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
