package logopreset

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	indexTTL     = 24 * time.Hour
	fetchTimeout = 10 * time.Second
	maxIndexSize = 5 << 20
	maxResults   = 30
)

// referencePattern matches the slugs Dashboard Icons uses as file names, so a reference can never escape its directory in an icon URL
var referencePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// nonAlphanumeric is stripped from names and queries so "home assistant" matches "Home-Assistant"
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// indexEntry is one icon in Dashboard Icons' metadata.json, which is keyed by the icon's reference
type indexEntry struct {
	// Base is the format of the source file, either "svg" or "png"
	Base    string   `json:"base"`
	Aliases []string `json:"aliases"`
	// Colors names the files to use on light and dark backgrounds, and is only set for icons that don't work on both
	// "dark" is the dark-colored icon for light backgrounds and "light" the light-colored icon for dark backgrounds
	Colors struct {
		Dark  string `json:"dark"`
		Light string `json:"light"`
	} `json:"colors"`
}

type preset struct {
	name            string
	reference       string
	format          string
	logoFile        string
	darkLogoFile    string
	searchReference string
	searchAliases   string
}

type Service struct {
	httpClient *http.Client
	baseURL    string
	cache      *utils.Cache[[]preset]
}

func newService(deps Dependencies) *Service {
	return &Service{
		httpClient: deps.HTTPClient,
		baseURL:    deps.BaseURL,
		cache:      utils.New[[]preset](indexTTL),
	}
}

// Search returns the icons that best match the query, ranked from exact to loose matches
func (s *Service) Search(ctx context.Context, query string) ([]logoPresetDto, error) {
	// Operators can turn the icon library off so Pocket ID never contacts it
	if s.baseURL == "" {
		return nil, apperror.LogoPresetsDisabled()
	}

	presets, err := s.getIndex(ctx)
	if err != nil {
		return nil, apperror.LogoPresetsUnavailable(err)
	}

	matches := search(presets, query, maxResults)
	result := make([]logoPresetDto, len(matches))
	for i, p := range matches {
		result[i] = s.toDto(p)
	}

	return result, nil
}

func (s *Service) getIndex(ctx context.Context) ([]preset, error) {
	presets, err := s.cache.GetOrFetch(ctx, s.fetchIndex)

	// A stale index is still good enough to search while the CDN is unreachable
	if staleErr, ok := errors.AsType[*utils.ErrStale](err); ok {
		slog.WarnContext(ctx, "Failed to refresh logo preset index, using stale cache", slog.Any("error", staleErr.Err))
		return presets, nil
	}

	return presets, err
}

func (s *Service) fetchIndex(ctx context.Context) ([]preset, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	// Download the index that lists every icon in the collection
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, s.baseURL+"/metadata.json", nil)
	if err != nil {
		return nil, fmt.Errorf("create icon index request: %w", err)
	}
	req.Header.Set("User-Agent", "pocket-id/logo-presets")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch icon index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("icon index returned status %d", resp.StatusCode)
	}

	var entries map[string]indexEntry
	err = json.NewDecoder(utils.NewLimitReader(resp.Body, maxIndexSize)).Decode(&entries)
	if err != nil {
		return nil, fmt.Errorf("decode icon index: %w", err)
	}

	presets := make([]preset, 0, len(entries))
	for reference, e := range entries {
		// An icon without color variants is meant for both light and dark backgrounds
		logoFile := cmp.Or(e.Colors.Dark, reference)
		darkLogoFile := cmp.Or(e.Colors.Light, reference)

		// Skip entries whose file names can't safely be used in an icon URL
		if !referencePattern.MatchString(reference) || !referencePattern.MatchString(logoFile) || !referencePattern.MatchString(darkLogoFile) {
			continue
		}

		// Every icon is available as PNG, but only icons drawn as SVG are available as SVG
		format := "png"
		if e.Base == "svg" {
			format = "svg"
		}

		presets = append(presets, preset{
			name:            displayName(reference, e.Aliases),
			reference:       reference,
			format:          format,
			logoFile:        logoFile,
			darkLogoFile:    darkLogoFile,
			searchReference: normalize(reference),
			searchAliases:   strings.ToLower(strings.Join(e.Aliases, ",")),
		})
	}

	return presets, nil
}

// displayName prefers an alias that spells the reference with its proper casing, like "GitHub" for "github"
func displayName(reference string, aliases []string) string {
	normalizedReference := normalize(reference)
	for _, alias := range aliases {
		if normalize(alias) == normalizedReference {
			return alias
		}
	}

	words := strings.Fields(strings.ReplaceAll(reference, "-", " "))
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

// search returns up to limit presets matching the query, ordered by match quality and then by reference
func search(presets []preset, query string, limit int) []preset {
	normalizedQuery := normalize(query)
	aliasQuery := strings.ToLower(strings.TrimSpace(query))

	type match struct {
		preset preset
		rank   int
	}

	// Rank every preset that matches the query at all
	matches := make([]match, 0, limit)
	for _, p := range presets {
		rank, ok := matchRank(p, normalizedQuery, aliasQuery)
		if ok {
			matches = append(matches, match{preset: p, rank: rank})
		}
	}

	// Lower ranks are better matches, and ties are broken alphabetically so results are stable
	slices.SortFunc(matches, func(a, b match) int {
		return cmp.Or(
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(a.preset.reference, b.preset.reference),
		)
	})

	result := make([]preset, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		result = append(result, m.preset)
	}

	return result
}

// matchRank reports how well a preset matches the query, where 0 is an exact match
func matchRank(p preset, normalizedQuery, aliasQuery string) (int, bool) {
	switch {
	case normalizedQuery == "":
		return 0, true
	case p.searchReference == normalizedQuery:
		return 0, true
	case strings.HasPrefix(p.searchReference, normalizedQuery):
		return 1, true
	case strings.Contains(p.searchReference, normalizedQuery):
		return 2, true
	case aliasQuery != "" && strings.Contains(p.searchAliases, aliasQuery):
		return 3, true
	default:
		return 0, false
	}
}

func (s *Service) toDto(p preset) logoPresetDto {
	dto := logoPresetDto{
		Name:      p.name,
		Reference: p.reference,
		LogoURL:   s.iconURL(p.format, p.logoFile),
	}

	if p.darkLogoFile != p.logoFile {
		darkLogoURL := s.iconURL(p.format, p.darkLogoFile)
		dto.DarkLogoURL = &darkLogoURL
	}

	return dto
}

func (s *Service) iconURL(format, name string) string {
	return s.baseURL + "/" + format + "/" + name + "." + format
}

func normalize(s string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(s), "")
}
