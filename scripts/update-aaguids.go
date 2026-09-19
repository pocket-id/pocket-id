// Refreshes the AAGUID metadata and authenticator icons from the community AAGUID list
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	sourceURL       = "https://raw.githubusercontent.com/pocket-id/passkey-aaguids/refs/heads/main/combined_aaguid.json"
	metadataFile    = "backend/resources/aaguids.json"
	iconDir         = "backend/resources/aaguid-icons"
	svgDataPrefix   = "data:image/svg+xml;base64,"
	iconFilePattern = "*.svg"
	iconDigestBytes = 8
)

// authenticator is the subset of an upstream entry this script cares about
// The icons are data URIs rather than links, so nothing beyond the combined JSON has to be fetched
type authenticator struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
}

// authenticatorMetadata is the generated runtime representation of an authenticator
// Icon file names contain a 64-bit SHA-256 prefix of their decoded SVG so identical upstream icons share one asset
type authenticatorMetadata struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light,omitempty"`
	IconDark  string `json:"icon_dark,omitempty"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalln("error:", err)
	}
}

// run writes the metadata manifest and its content-addressed icons from one upstream snapshot so their references stay consistent
func run() error {
	authenticators, err := fetchAuthenticators(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", sourceURL, err)
	}
	if len(authenticators) == 0 {
		return errors.New("upstream AAGUID list is empty")
	}

	metadata, icons, err := buildResources(authenticators)
	if err != nil {
		return fmt.Errorf("failed to build authenticator resources: %w", err)
	}

	if err := writeIcons(icons); err != nil {
		return fmt.Errorf("failed to write icon files: %w", err)
	}

	if err := writeAuthenticatorMetadata(metadata); err != nil {
		return fmt.Errorf("failed to write authenticator metadata: %w", err)
	}

	return nil
}

// fetchAuthenticators downloads the upstream authenticator list keyed by AAGUID
func fetchAuthenticators(url string) (authenticators map[string]authenticator, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticator JSON: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status 200, got %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&authenticators); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return authenticators, nil
}

// buildResources converts upstream data URIs into metadata references and a set of unique decoded SVGs
func buildResources(authenticators map[string]authenticator) (map[string]authenticatorMetadata, map[string][]byte, error) {
	metadata := make(map[string]authenticatorMetadata, len(authenticators))
	icons := make(map[string][]byte)

	// The AAGUIDs are sorted so failures are deterministic when the upstream data contains an invalid icon
	for _, aaguid := range slices.Sorted(maps.Keys(authenticators)) {
		a := authenticators[aaguid]
		entry := authenticatorMetadata{Name: a.Name}

		// Entries without a usable light SVG retain their display names but do not reference any icon
		lightName, err := addIcon(icons, a.IconLight)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process light SVG for %s: %w", aaguid, err)
		}
		if lightName == "" {
			metadata[aaguid] = entry
			continue
		}
		entry.IconLight = lightName

		// A missing or identical dark icon falls back to the light asset at runtime
		darkName, err := addIcon(icons, a.IconDark)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process dark SVG for %s: %w", aaguid, err)
		}
		if darkName != "" && darkName != lightName {
			entry.IconDark = darkName
		}

		metadata[aaguid] = entry
	}

	return metadata, icons, nil
}

// addIcon decodes one SVG data URI and returns the content-addressed file name stored in icons
func addIcon(icons map[string][]byte, dataURI string) (string, error) {
	payload, ok := strings.CutPrefix(dataURI, svgDataPrefix)
	if !ok {
		return "", nil
	}

	svg, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	digest := sha256.Sum256(svg)
	name := fmt.Sprintf("%x.svg", digest[:iconDigestBytes])
	if existing, exists := icons[name]; exists && !bytes.Equal(existing, svg) {
		return "", fmt.Errorf("SHA-256 prefix collision for %s", name)
	}
	icons[name] = svg

	return name, nil
}

// writeIcons replaces the icon directory contents so assets dropped upstream do not linger
func writeIcons(icons map[string][]byte) error {
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return fmt.Errorf("failed to create authenticator icons directory: %w", err)
	}

	existing, err := filepath.Glob(filepath.Join(iconDir, iconFilePattern))
	if err != nil {
		return fmt.Errorf("error gathering existing SVGs: %w", err)
	}
	for _, path := range existing {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("error clearing existing SVG: %w", err)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(icons)) {
		if err := os.WriteFile(filepath.Join(iconDir, name), icons[name], 0o644); err != nil {
			return fmt.Errorf("failed to write SVG %s: %w", name, err)
		}
	}

	return nil
}

// writeAuthenticatorMetadata writes the AAGUID manifest consumed by the backend
func writeAuthenticatorMetadata(metadata map[string]authenticatorMetadata) (err error) {
	if err := os.MkdirAll(filepath.Dir(metadataFile), 0o755); err != nil {
		return fmt.Errorf("failed to create authenticator metadata directory: %w", err)
	}

	file, err := os.Create(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to create authenticator metadata file: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metadata)
}
