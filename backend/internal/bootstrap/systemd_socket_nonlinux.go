//go:build !linux

package bootstrap

import "errors"

func systemdSocket() (*socket, error) {
	return nil, errors.New("systemd socket activation is only supported on Linux")
}
