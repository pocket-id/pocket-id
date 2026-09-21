package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"
	"uuid"

	"github.com/pocket-id/pocket-id/backend/resources"
)

const (
	aaguidIconsDir             = "aaguid-icons"
	authenticatorIconHashBytes = 8
)

// ZeroAAGUID is the AAGUID reported by authenticators that do not want to identify themselves, and it is also the column default for credentials that were registered before AAGUIDs were tracked
var ZeroAAGUID = uuid.Nil().String()

var (
	aaguidMetadata     map[string]authenticatorMetadata
	aaguidMetadataOnce *sync.Once
)

// authenticatorMetadata records the display name and content-addressed icon files for an AAGUID
// IconDark is empty when the authenticator uses its light icon in both themes
type authenticatorMetadata struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
}

func init() {
	aaguidMetadataOnce = &sync.Once{}
}

// FormatAAGUID converts an AAGUID byte slice to UUID string format
func FormatAAGUID(aaguid []byte) string {
	if len(aaguid) == 0 {
		return ""
	}

	// If exactly 16 bytes, format as UUID
	if len(aaguid) == 16 {
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			aaguid[0:4], aaguid[4:6], aaguid[6:8], aaguid[8:10], aaguid[10:16])
	}

	// Otherwise just return as hex
	return hex.EncodeToString(aaguid)
}

// GetAuthenticatorName returns the name of the authenticator for the given AAGUID
func GetAuthenticatorName(aaguid []byte) string {
	aaguidStr := FormatAAGUID(aaguid)
	if aaguidStr == "" {
		return ""
	}

	// Then check the embedded metadata manifest
	aaguidMetadataOnce.Do(loadAAGUIDMetadataFromFile)

	if metadata, ok := aaguidMetadata[aaguidStr]; ok && metadata.Name != "" {
		return metadata.Name + " Passkey"
	}

	return ""
}

// loadAAGUIDMetadataFromFile loads AAGUID names and icon references from the embedded manifest
func loadAAGUIDMetadataFromFile() {
	// Read from embedded file system
	data, err := resources.FS.ReadFile("aaguids.json")
	if err != nil {
		slog.Error("Error reading embedded AAGUID file", slog.Any("error", err))
		return
	}

	err = json.Unmarshal(data, &aaguidMetadata)
	if err != nil {
		slog.Error("Error unmarshalling AAGUID data", slog.Any("error", err))
		return
	}
}

// HasAuthenticatorIcon reports whether an icon is embedded for the given AAGUID
// Callers use this to avoid pointing clients at an icon endpoint that would only answer with a 404
func HasAuthenticatorIcon(aaguid string) bool {
	aaguidMetadataOnce.Do(loadAAGUIDMetadataFromFile)

	metadata, ok := aaguidMetadata[aaguid]
	return ok && validAuthenticatorIconName(metadata.IconLight)
}

// OpenAuthenticatorIcon opens the embedded icon for the given AAGUID and returns it together with its size
// It returns os.ErrNotExist for every AAGUID without an icon, which is also what keeps caller-controlled input from ever reaching the embedded file system
func OpenAuthenticatorIcon(aaguid string, light bool) (fs.File, int64, error) {
	aaguidMetadataOnce.Do(loadAAGUIDMetadataFromFile)

	// Only AAGUIDs with a valid generated light reference are served, so caller input never becomes an embedded file path
	metadata, ok := aaguidMetadata[aaguid]
	if !ok || !validAuthenticatorIconName(metadata.IconLight) {
		return nil, 0, os.ErrNotExist
	}

	// Fall back to the light icon when the authenticator does not ship a dark variant
	name := metadata.IconLight
	if !light && validAuthenticatorIconName(metadata.IconDark) {
		name = metadata.IconDark
	}

	file, err := resources.FS.Open(path.Join(aaguidIconsDir, name))
	if err != nil {
		return nil, 0, err
	}

	// The size is resolved upfront so the caller can stream the icon with a Content-Length instead of buffering it
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}

	return file, stat.Size(), nil
}

// validAuthenticatorIconName accepts only the truncated SHA-256 file names emitted by the updater
func validAuthenticatorIconName(name string) bool {
	const extension = ".svg"
	if len(name) != authenticatorIconHashBytes*2+len(extension) || name[len(name)-len(extension):] != extension {
		return false
	}

	_, err := hex.DecodeString(name[:authenticatorIconHashBytes*2])
	return err == nil
}
