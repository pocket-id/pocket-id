package systemd

import (
	"net"
	"os"
)

type SdNotifyState string

const (
	// SdNotifyReady tells the service manager that service startup is finished
	// or the service finished loading its configuration.
	SdNotifyReady SdNotifyState = "READY=1"

	// SdNotifyStopping tells the service manager that the service is beginning
	// its shutdown.
	SdNotifyStopping SdNotifyState = "STOPPING=1"

	// SdNotifyReloading tells the service manager that this service is
	// reloading its configuration. Note that you must call SdNotifyReady when
	// it completed reloading.
	SdNotifyReloading SdNotifyState = "RELOADING=1"

	// SdNotifyWatchdog tells the service manager to update the watchdog
	// timestamp for the service.
	SdNotifyWatchdog SdNotifyState = "WATCHDOG=1"
)

// SdNotify sends a message to the systemd daemon. It is common to ignore the error.
func SdNotify(state SdNotifyState) error {
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

	if _, err = conn.Write([]byte(state)); err != nil {
		return err
	}

	return nil
}
