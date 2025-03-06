package systemd

import (
	"net"
	"os"
)

// SdNotifyReady sends a message to the systemd daemon to notify that service is ready to operate.
// It is common to ignore the error.
func SdNotifyReady() error {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	if socketAddr.Name == "" {
		return nil
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	if _, err = conn.Write([]byte("READY=1")); err != nil {
		return err
	}

	return nil
}
