package service

import (
	"context"
	"io"
)

// ActorsBackupFileName is the name of the entry in the export ZIP that contains the actor host's data
// The payload is Francis' own backup stream, which is a binary, versioned, provider-neutral format
const ActorsBackupFileName = "francis.bin"

// ActorsBackupProvider backs up and restores the actor host's data: actor state, alarms, and dead-lettered jobs.
type ActorsBackupProvider interface {
	// Backup writes a snapshot of all the actor host's persistent data to w.
	// It runs against a consistent snapshot without blocking writers, so it's safe to take while Pocket ID is running.
	Backup(ctx context.Context, w io.Writer) error

	// Restore wipes all the actor host's persistent data and loads a snapshot produced by Backup from r.
	// It fails if any host is currently connected, since restoring underneath a running Pocket ID instance would corrupt active actors.
	Restore(ctx context.Context, r io.Reader) error
}
