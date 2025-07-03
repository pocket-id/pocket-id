package cmds

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
)

var rootCmd = &cobra.Command{
	Use:   "pocket-id",
	Short: "A simple and easy-to-use OIDC provider that allows users to authenticate with their passkeys to your services.",
	Long:  "By default, this command starts the pocket-id server.",
	Run: func(cmd *cobra.Command, args []string) {
		// Start the server
		err := bootstrap.Bootstrap()
		if err != nil {
			slog.Error("Failed to run pocket-id", "error", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
