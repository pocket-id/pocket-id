package bootstrap

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type socket struct {
	addr     string
	listener net.Listener
}

func unixSocket() (*socket, error) {
	addr := common.EnvConfig.UnixSocket
	os.Remove(addr) // remove dangling the socket file to avoid file-exist error

	listener, err := net.Listen("unix", addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to create UNIX socket: %w", err)
	}

	if common.EnvConfig.UnixSocketMode != "" {
		mode, err := strconv.ParseUint(common.EnvConfig.UnixSocketMode, 8, 32)
		if err != nil {
			listener.Close()
			return nil, fmt.Errorf("failed to parse UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}

		if err := os.Chmod(addr, os.FileMode(mode)); err != nil {
			listener.Close()
			return nil, fmt.Errorf("failed to set UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}
	}

	return &socket{addr, listener}, nil
}

func tcpSocket() (*socket, error) {
	addr := net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port)

	listener, err := net.Listen("tcp", addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP socket: %w", err)
	}

	return &socket{addr, listener}, nil
}
