//go:build linux

package bootstrap

import (
	"errors"
	"fmt"

	"github.com/coreos/go-systemd/activation"
)

func systemdSocket() (*socket, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("failed to receive socket from systemd: %w", err)
	}

	if len(listeners) == 0 {
		return nil, errors.New("did not receive any sockets from systemd")
	}

	if len(listeners) > 1 {
		return nil, errors.New("received too many sockets from systemd")
	}

	return &socket{"(systemd)", listeners[0]}, nil
}
