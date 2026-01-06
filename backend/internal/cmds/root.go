package cmds

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils/signals"
)

// Global flags for CLI commands
var globalFlags struct {
	endpoint string
	apiKey   string
	format   string
}

var rootCmd = &cobra.Command{
	Use:          "pocket-id",
	Short:        "A simple and easy-to-use OIDC provider that allows users to authenticate with their passkeys to your services.",
	Long:         "By default, this command starts the pocket-id server.",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Start the server
		err := bootstrap.Bootstrap(cmd.Context())
		if err != nil {
			slog.Error("Failed to run pocket-id", "error", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&globalFlags.endpoint, "endpoint", "e", "", "API endpoint URL (default: http://localhost:"+common.EnvConfig.Port+")")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.apiKey, "api-key", "k", "", "API key for authentication (can also use POCKET_ID_API_KEY env var)")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.format, "format", "f", "json", "Output format (json, yaml, table)")

	// Get a context that is canceled when the application is stopping
	ctx := signals.SignalContext(context.Background())

	// Pre-run function to set API key from environment variable if not provided via flag
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// If API key is not set via flag, check environment variable
		if globalFlags.apiKey == "" {
			if envApiKey := os.Getenv("POCKET_ID_API_KEY"); envApiKey != "" {
				globalFlags.apiKey = envApiKey
			}
		}
	}

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
