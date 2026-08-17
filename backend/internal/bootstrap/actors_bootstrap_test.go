package bootstrap

import (
	"bytes"
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	francishost "github.com/italypaleale/francis/host"
	"github.com/italypaleale/francis/host/local"
	"github.com/italypaleale/francis/host/remote"
	"github.com/libtnb/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestNewActorsOptsGetPSKUsesStableValue(t *testing.T) {
	opts := NewActorsOpts{
		EnvConfig: &common.EnvConfigSchema{
			EncryptionKey: []byte("test-encryption-key"),
		},
		// Constant value for this test
		InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
	}

	expectedHex := "db09067fa194c3731bf77b6415a1c5d903f03d4557605ba3236b31f6eddfc8d7"
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	actual, err := opts.getPSK()
	require.NoError(t, err)
	require.Equalf(t, expected, actual, "actual result: %s", actual)
}

// TestNewActorsSelectsTopology covers the branch that FRANCIS_HOST drives: with no standalone runtime configured Pocket ID starts an embedded one, and otherwise it connects to the addresses it was given.
func TestNewActorsSelectsTopology(t *testing.T) {
	// The actor host is created but never run, so the database only has to exist
	newDB := func(t *testing.T) *gorm.DB {
		t.Helper()

		dbPath := filepath.Join(t.TempDir(), "pocket-id.db")
		db, err := gorm.Open(sqlite.Open("file:"+dbPath+"?_pragma=foreign_keys(1)"), &gorm.Config{})
		require.NoError(t, err)
		t.Cleanup(func() {
			sqlDB, dbErr := db.DB()
			if dbErr == nil {
				_ = sqlDB.Close()
			}
		})

		return db
	}

	// registerCronJobs reads the app environment off the global config and skips every job in test mode, which keeps each case down to the host itself and the rate limiters
	baseConfig := func(t *testing.T) *common.EnvConfigSchema {
		t.Helper()

		originalAppEnv := common.EnvConfig.AppEnv
		common.EnvConfig.AppEnv = common.AppEnvTest
		t.Cleanup(func() {
			common.EnvConfig.AppEnv = originalAppEnv
		})

		return &common.EnvConfigSchema{
			AppEnv:        common.AppEnvTest,
			EncryptionKey: []byte("test-encryption-key"),
			ActorsHost:    "127.0.0.1",
			ActorsPort:    "1414",
		}
	}

	t.Run("embedded runtime by default", func(t *testing.T) {
		cfg := baseConfig(t)

		h, rateLimitServices, err := NewActors(NewActorsOpts{
			EnvConfig:  cfg,
			InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
			DB:         newDB(t),
		})
		require.NoError(t, err)
		require.IsType(t, &local.Host{}, h)
		require.NotEmpty(t, rateLimitServices)
	})

	t.Run("remote runtime when addresses are configured", func(t *testing.T) {
		cfg := baseConfig(t)
		cfg.FrancisAddresses = []string{"runtime-1.example.com:8443", "runtime-2.example.com:8443"}
		cfg.FrancisHostPSK = []byte("bootstrap-psk-that-is-long-enough")

		// No database is passed, since a standalone runtime owns the actor data and the remote host must not reach for Pocket ID's own database
		h, rateLimitServices, err := NewActors(NewActorsOpts{
			EnvConfig:  cfg,
			InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
		})
		require.NoError(t, err)
		require.IsType(t, &remote.Host{}, h)
		require.NotEmpty(t, rateLimitServices)
	})

	// Each bootstrap method has to produce a host Francis accepts, which is the only part of the remote wiring that can be checked without a runtime to connect to
	t.Run("every bootstrap method builds a valid remote host", func(t *testing.T) {
		jwtFile := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(jwtFile, []byte("header.payload.signature"), 0600))

		for name, apply := range map[string]func(cfg *common.EnvConfigSchema){
			"PSK":      func(cfg *common.EnvConfigSchema) { cfg.FrancisHostPSK = []byte("bootstrap-psk-that-is-long-enough") },
			"JWT":      func(cfg *common.EnvConfigSchema) { cfg.FrancisHostJWT = "header.payload.signature" },
			"JWT file": func(cfg *common.EnvConfigSchema) { cfg.FrancisHostJWTFile = jwtFile },
		} {
			t.Run(name, func(t *testing.T) {
				cfg := baseConfig(t)
				cfg.FrancisAddresses = []string{"runtime-1.example.com:8443"}
				apply(cfg)

				opts := NewActorsOpts{EnvConfig: cfg, InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876"}
				h, err := opts.newRemoteHost(slog.New(slog.DiscardHandler))
				require.NoError(t, err)
				require.NotNil(t, h)
			})
		}
	})

	t.Run("no bootstrap method is rejected by Francis", func(t *testing.T) {
		cfg := baseConfig(t)
		cfg.FrancisAddresses = []string{"runtime-1.example.com:8443"}

		opts := NewActorsOpts{EnvConfig: cfg, InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876"}
		_, err := opts.newRemoteHost(slog.New(slog.DiscardHandler))
		require.Error(t, err)
	})

	t.Run("the actor client requires a standalone runtime", func(t *testing.T) {
		cfg := baseConfig(t)

		err := WithActorClient(t.Context(), cfg, func(context.Context, francishost.Host) error {
			t.Fatal("the callback must not run without a standalone runtime")
			return nil
		})
		require.ErrorIs(t, err, ErrEmbeddedFrancisRuntime)
	})

	t.Run("state store is unavailable with a remote runtime", func(t *testing.T) {
		cfg := baseConfig(t)
		cfg.FrancisAddresses = []string{"runtime-1.example.com:8443"}
		cfg.FrancisHostPSK = []byte("bootstrap-psk-that-is-long-enough")

		_, err := NewActorStateStore(NewActorsOpts{
			EnvConfig:  cfg,
			InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
			DB:         newDB(t),
		})
		require.ErrorIs(t, err, ErrRemoteFrancisRuntime)
	})
}

// TestNewActorsBackupProvider covers the provider the export and import use to back up and restore the actor host's data.
// It builds the provider from the same options the actor host uses, so a mismatch between those options and the concrete provider would otherwise only surface at runtime, when an export or import is attempted.
func TestNewActorsBackupProvider(t *testing.T) {
	// Foreign keys must be enabled, which the provider validates on init and which the application enables on every connection
	dbPath := filepath.Join(t.TempDir(), "pocket-id.db")
	dsn := "file:" + dbPath + "?_txlock=immediate&_pragma=busy_timeout(2500)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	providerOpts, err := ActorsProviderOptions(db, nil)
	require.NoError(t, err)

	provider, err := NewActorsBackupProvider(t.Context(), providerOpts)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = provider.Close()
	})

	// Init applies the actor host's schema migrations, so an export also works against a database the actor host has never run against
	var tables int64
	err = db.Raw(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name LIKE 'francis_%'`).Scan(&tables).Error
	require.NoError(t, err)
	require.Positive(t, tables, "the provider must create the actor host's own tables")

	// Even an empty cluster produces a valid backup stream, which an import can restore
	buf := &bytes.Buffer{}
	err = provider.Backup(t.Context(), buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "francis-backup")

	err = provider.Restore(t.Context(), bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
}
