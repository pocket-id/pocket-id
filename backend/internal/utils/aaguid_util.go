package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"uuid"

	"github.com/pocket-id/pocket-id/backend/resources"
)

const aaguidIconsDir = "aaguid-icons"

// ZeroAAGUID is the AAGUID reported by authenticators that do not want to identify themselves, and it is also the column default for credentials that were registered before AAGUIDs were tracked
var ZeroAAGUID = uuid.Nil().String()

var (
	aaguidMap     map[string]string
	aaguidMapOnce *sync.Once

	// aaguidIcons is an index of the embedded icon directory, keyed by AAGUID and built lazily on first use
	aaguidIcons     map[string]authenticatorIcon
	aaguidIconsOnce *sync.Once
)

// authenticatorIcon records the embedded file names of an authenticator's icon
// darkPath is empty when the authenticator only ships a single icon that is expected to work in both themes
type authenticatorIcon struct {
	lightPath string
	darkPath  string
}

func init() {
	aaguidMapOnce = &sync.Once{}
	aaguidIconsOnce = &sync.Once{}
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

	// Then check JSON-sourced map
	aaguidMapOnce.Do(loadAAGUIDsFromFile)

	if name, ok := aaguidMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	return ""
}

// loadAAGUIDsFromFile loads AAGUID data from the embedded file system
func loadAAGUIDsFromFile() {
	// Read from embedded file system
	data, err := resources.FS.ReadFile("aaguids.json")
	if err != nil {
		slog.Error("Error reading embedded AAGUID file", slog.Any("error", err))
		return
	}

	err = json.Unmarshal(data, &aaguidMap)
	if err != nil {
		slog.Error("Error unmarshalling AAGUID data", slog.Any("error", err))
		return
	}
}

// HasAuthenticatorIcon reports whether an icon is embedded for the given AAGUID
// Callers use this to avoid pointing clients at an icon endpoint that would only answer with a 404
func HasAuthenticatorIcon(aaguid string) bool {
	aaguidIconsOnce.Do(loadAAGUIDIconsFromFS)

	_, ok := aaguidIcons[aaguid]
	return ok
}

// OpenAuthenticatorIcon opens the embedded icon for the given AAGUID and returns it together with its size
// It returns os.ErrNotExist for every AAGUID without an icon, which is also what keeps caller-controlled input from ever reaching the embedded file system
func OpenAuthenticatorIcon(aaguid string, light bool) (fs.File, int64, error) {
	aaguidIconsOnce.Do(loadAAGUIDIconsFromFS)

	// Only AAGUIDs present in the index are served, so the file name below comes from the index and never from the caller
	icon, ok := aaguidIcons[aaguid]
	if !ok {
		return nil, 0, os.ErrNotExist
	}

	// Fall back to the light icon when the authenticator does not ship a dark variant
	name := icon.lightPath
	if !light && icon.darkPath != "" {
		name = icon.darkPath
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

// loadAAGUIDIconsFromFS indexes the embedded icon directory so later lookups do not have to touch the file system
func loadAAGUIDIconsFromFS() {
	entries, err := fs.ReadDir(resources.FS, aaguidIconsDir)
	if err != nil {
		slog.Error("Error reading embedded AAGUID icons", slog.Any("error", err))
		return
	}

	// Icons are named <aaguid>.svg, with an optional <aaguid>.dark.svg companion for authenticators that need a separate dark variant
	aaguidIcons = make(map[string]authenticatorIcon, len(entries))
	for _, entry := range entries {
		name := entry.Name()

		switch {
		case strings.HasSuffix(name, ".dark.svg"):
			aaguid := strings.TrimSuffix(name, ".dark.svg")
			icon := aaguidIcons[aaguid]
			icon.darkPath = name
			aaguidIcons[aaguid] = icon
		case strings.HasSuffix(name, ".svg"):
			aaguid := strings.TrimSuffix(name, ".svg")
			icon := aaguidIcons[aaguid]
			icon.lightPath = name
			aaguidIcons[aaguid] = icon
		}
	}

	// Drop authenticators that only have a dark icon because there would be nothing to serve in light mode
	for aaguid, icon := range aaguidIcons {
		if icon.lightPath == "" {
			delete(aaguidIcons, aaguid)
		}
	}
}
