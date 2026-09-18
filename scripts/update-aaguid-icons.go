// Refreshes the AAGUID name lookup and the authenticator icons from the community AAGUID list.
package main

import (
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
	sourceURL     = "https://raw.githubusercontent.com/pocket-id/passkey-aaguids/refs/heads/main/combined_aaguid.json"
	namesFile     = "backend/resources/aaguids.json"
	iconDir       = "backend/resources/aaguid-icons"
	svgDataPrefix = "data:image/svg+xml;base64,"
)

type authenticator struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalln("error:", err)
	}
}

func run() error {
	authenticators, err := fetchAuthenticators(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", sourceURL, err)
	}

	if err := writeAuthenticatorNames(authenticators); err != nil {
		return fmt.Errorf("failed to write authenticator names: %w", err)
	}

	if err := writeIcons(authenticators); err != nil {
		return fmt.Errorf("failed to write icon files: %w", err)
	}

	return nil
}

func fetchAuthenticators(url string) (map[string]authenticator, error) {
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

	var authenticators map[string]authenticator
	if err := json.NewDecoder(resp.Body).Decode(&authenticators); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return authenticators, nil
}

func writeAuthenticatorNames(authenticators map[string]authenticator) error {
	names := map[string]string{}
	for aaguid, a := range authenticators {
		names[aaguid] = a.Name
	}

	if err := os.MkdirAll(filepath.Dir(namesFile), 0o755); err != nil {
		return fmt.Errorf("failed to create authenticator names directory: %w", err)
	}

	file, err := os.Create(namesFile)
	if err != nil {
		return fmt.Errorf("failed to create authenticator names file: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return json.NewEncoder(file).Encode(names)
}

// writeIcons replaces the icon directory contents so icons dropped upstream do not linger
func writeIcons(authenticators map[string]authenticator) error {
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return fmt.Errorf("failed to create authenticator icons directory: %w", err)
	}

	existing, err := filepath.Glob(filepath.Join(iconDir, "*.svg"))
	if err != nil {
		return fmt.Errorf("error gathering existing SVGs: %w", err)
	}
	for _, path := range existing {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("error clearing existing SVG: %w", err)
		}
	}

	for _, aaguid := range slices.Sorted(maps.Keys(authenticators)) {
		a := authenticators[aaguid]

		// An entry without a usable light icon is skipped entirely, dark alone is never enough
		lightContent := svgPayload(a.IconLight)
		if lightContent == "" {
			continue
		}

		if err := writeIcon(fmt.Sprintf("%s.svg", aaguid), lightContent); err != nil {
			return fmt.Errorf("failed to write light SVG for %s: %w", aaguid, err)
		}

		darkContent := svgPayload(a.IconDark)
		if darkContent == "" || darkContent == lightContent {
			continue
		}

		if err = writeIcon(fmt.Sprintf("%s.dark.svg", aaguid), darkContent); err != nil {
			return fmt.Errorf("failed to write dark SVG for %s: %w", aaguid, err)
		}
	}

	return nil
}

func writeIcon(name, payload string) error {
	svg, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	if err := os.WriteFile(filepath.Join(iconDir, name), svg, 0o644); err != nil {
		return fmt.Errorf("failed to write SVG to file: %w", err)
	}

	return nil
}

func svgPayload(dataURI string) string {
	payload, ok := strings.CutPrefix(dataURI, svgDataPrefix)
	if !ok {
		return ""
	}
	return payload
}
