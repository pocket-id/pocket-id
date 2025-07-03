package cmds

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type healthcheckFlags struct {
	Endpoint string
	Verbose  bool
}

func init() {
	var flags healthcheckFlags

	healthcheckCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Performs a healthcheck of a running Pocket ID instance",
		Run: func(cmd *cobra.Command, args []string) {
			start := time.Now()

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			url := flags.Endpoint + "/healthz"
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				slog.ErrorContext(ctx,
					"Failed to create request object",
					"error", err,
					"url", url,
					"ms", time.Since(start).Milliseconds(),
				)
				os.Exit(1)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				slog.ErrorContext(ctx,
					"Failed to perform request",
					"error", err,
					"url", url,
					"ms", time.Since(start).Milliseconds(),
				)
				os.Exit(1)
			}
			defer res.Body.Close()

			if res.StatusCode < 200 || res.StatusCode >= 300 {
				if err != nil {
					slog.ErrorContext(ctx,
						"Healthcheck failed",
						"status", res.StatusCode,
						"url", url,
						"ms", time.Since(start).Milliseconds(),
					)
					os.Exit(1)
				}
			}

			if flags.Verbose {
				slog.InfoContext(ctx,
					"Healthcheck succeeded",
					"status", res.StatusCode,
					"url", url,
					"ms", time.Since(start).Milliseconds(),
				)
			}
		},
	}

	healthcheckCmd.Flags().StringVarP(&flags.Endpoint, "endpoint", "e", "http://localhost:"+common.EnvConfig.Port, "Endpoint for Pocket ID")
	healthcheckCmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose mode")

	rootCmd.AddCommand(healthcheckCmd)
}
