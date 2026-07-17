package cmds

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type healthcheckFlags struct {
	Endpoint   string
	UnixSocket string
	Verbose    bool
}

type healthcheckResult struct {
	StatusCode int
	URL        string
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

			if flags.UnixSocket == "" && !cmd.Flags().Changed("endpoint") {
				flags.UnixSocket = common.EnvConfig.UnixSocket
			}

			result, err := healthcheck(ctx, flags)
			if err != nil {
				slog.ErrorContext(ctx,
					"Healthcheck failed",
					"error", err,
					"ms", time.Since(start).Milliseconds(),
				)
				os.Exit(1)
			}

			if flags.Verbose {
				slog.InfoContext(ctx,
					"Healthcheck succeeded",
					"status", result.StatusCode,
					"url", result.URL,
					"unixSocket", flags.UnixSocket,
					"ms", time.Since(start).Milliseconds(),
				)
			}
		},
	}

	healthcheckCmd.Flags().StringVarP(&flags.Endpoint, "endpoint", "e", defaultEndpoint(), "Endpoint for Pocket ID")
	healthcheckCmd.Flags().StringVar(&flags.UnixSocket, "unix-socket", "", "UNIX socket path for Pocket ID")
	healthcheckCmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose mode")

	rootCmd.AddCommand(healthcheckCmd)
}

// The server only serves TLS when both a certificate and a key file are configured
func tlsEnabled() bool {
	return common.EnvConfig.TLSCertFile != "" && common.EnvConfig.TLSKeyFile != ""
}

func defaultEndpoint() string {
	scheme := "http"
	if tlsEnabled() {
		scheme = "https"
	}

	return scheme + "://localhost:" + common.EnvConfig.Port
}

func healthcheck(ctx context.Context, flags healthcheckFlags) (*healthcheckResult, error) {
	url := strings.TrimRight(flags.Endpoint, "/") + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request object for %q: %w", url, err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if flags.UnixSocket != "" {
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "unix", flags.UnixSocket)
		}
	}

	// If TLS cert and key files are provided, there is a high chance that the server is using a self-signed certificate
	if tlsEnabled() {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // no sensitive data is transmitted in healthcheck
		}
	}

	client := &http.Client{Transport: transport}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to %q: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from %q", res.StatusCode, url)
	}

	return &healthcheckResult{
		StatusCode: res.StatusCode,
		URL:        url,
	}, nil
}
