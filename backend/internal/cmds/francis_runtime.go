package cmds

import (
	"archive/zip"
	"fmt"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// printRemoteActorDataNotice warns that the command does not cover the actor data, which a standalone Francis runtime owns rather than Pocket ID
// It goes to stderr so it stays visible when the archive itself is streamed to stdout
func printRemoteActorDataNotice(consequence string, remedy string) {
	fmt.Fprintf(os.Stderr, `WARNING: FRANCIS_HOST points to a standalone Francis runtime.
  %s.
  That data covers the app configuration, signup and one-time access tokens, device
  login requests, LDAP sync state, and the schedule of the background jobs.
  To cover it, %s.

`, consequence, remedy)
}

// ensureNoActorsBackup rejects an archive that carries the actor data when a standalone Francis runtime owns it
// Such an archive comes from a deployment with an embedded runtime, and restoring only its Pocket ID half would leave the runtime holding actor state belonging to a different deployment
func ensureNoActorsBackup(zipReader *zip.Reader) error {
	for _, f := range zipReader.File {
		if f.Name == service.ActorsBackupFileName {
			return fmt.Errorf("this archive contains the actor data (%s) but FRANCIS_HOST points to a standalone Francis runtime, which owns that data instead: restore it into a deployment with an embedded runtime, or load the actor data into the runtime with 'francis runtime restore'", service.ActorsBackupFileName)
		}
	}

	return nil
}
