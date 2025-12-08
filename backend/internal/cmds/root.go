package cmds

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/utils/signals"
)

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
	// Get a context that is canceled when the application is stopping
	ctx := signals.SignalContext(context.Background())

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
