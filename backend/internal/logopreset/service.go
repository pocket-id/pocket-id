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

// referencePattern matches the slugs selfh.st uses as file names, so a reference can never escape its directory in an icon URL
var referencePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// nonAlphanumeric is stripped from names and queries so "home assistant" matches "Home-Assistant"
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// indexEntry is one icon in selfh.st's index.json, whose availability flags are the strings "Yes" or "No"
type indexEntry struct {
	Name      string `json:"Name"`
	Reference string `json:"Reference"`
	SVG       string `json:"SVG"`
	Light     string `json:"Light"`
	Tags      string `json:"Tags"`
}

// preset is an index entry with its search keys normalized once when the index is loaded
type preset struct {
	indexEntry

	searchName      string
	searchReference string
	searchTags      string
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
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, s.baseURL+"/index.json", nil)
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

	var entries []indexEntry
	err = json.NewDecoder(utils.NewLimitReader(resp.Body, maxIndexSize)).Decode(&entries)
	if err != nil {
		return nil, fmt.Errorf("decode icon index: %w", err)
	}

	// Skip entries whose reference can't safely be used as a file name in an icon URL
	presets := make([]preset, 0, len(entries))
	for _, e := range entries {
		if !referencePattern.MatchString(e.Reference) {
			continue
		}

		presets = append(presets, preset{
			indexEntry:      e,
			searchName:      normalize(e.Name),
			searchReference: normalize(e.Reference),
			searchTags:      strings.ToLower(e.Tags),
		})
	}

	return presets, nil
}

// search returns up to limit presets matching the query, ordered by match quality and then by name
func search(presets []preset, query string, limit int) []preset {
	normalizedQuery := normalize(query)
	tagQuery := strings.ToLower(strings.TrimSpace(query))

	type match struct {
		preset preset
		rank   int
	}

	// Rank every preset that matches the query at all
	matches := make([]match, 0, limit)
	for _, p := range presets {
		rank, ok := matchRank(p, normalizedQuery, tagQuery)
		if ok {
			matches = append(matches, match{preset: p, rank: rank})
		}
	}

	// Lower ranks are better matches, and ties are broken alphabetically so results are stable
	slices.SortFunc(matches, func(a, b match) int {
		return cmp.Or(
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(a.preset.searchName, b.preset.searchName),
		)
	})

	result := make([]preset, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		result = append(result, m.preset)
	}

	return result
}

// matchRank reports how well a preset matches the query, where 0 is an exact match
func matchRank(p preset, normalizedQuery, tagQuery string) (int, bool) {
	switch {
	case normalizedQuery == "":
		return 0, true
	case p.searchName == normalizedQuery || p.searchReference == normalizedQuery:
		return 0, true
	case strings.HasPrefix(p.searchName, normalizedQuery) || strings.HasPrefix(p.searchReference, normalizedQuery):
		return 1, true
	case strings.Contains(p.searchName, normalizedQuery) || strings.Contains(p.searchReference, normalizedQuery):
		return 2, true
	case tagQuery != "" && strings.Contains(p.searchTags, tagQuery):
		return 3, true
	default:
		return 0, false
	}
}

func (s *Service) toDto(p preset) logoPresetDto {
	format := "png"
	if p.SVG == "Yes" {
		format = "svg"
	}

	dto := logoPresetDto{
		Name:      p.Name,
		Reference: p.Reference,
		LogoURL:   s.iconURL(format, p.Reference),
	}

	// The white variant keeps monochrome logos readable on dark backgrounds
	if p.Light == "Yes" {
		darkLogoURL := s.iconURL(format, p.Reference+"-light")
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
