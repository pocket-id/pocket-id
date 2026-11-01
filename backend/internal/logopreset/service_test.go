package logopreset

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const testBaseURL = "https://icons.example.com"

const testIndex = `{
	"nextcloud": {"base": "svg", "aliases": ["Nextcloud"]},
	"nextcloud-talk": {"base": "svg", "aliases": []},
	"ghost": {"base": "png", "aliases": ["blogging"], "colors": {"light": "ghost-light"}},
	"home-assistant": {"base": "svg", "aliases": []},
	"owncloud": {"base": "svg", "aliases": ["ownCloud", "cloud-storage"]},
	"github": {"base": "svg", "aliases": ["GitHub"], "colors": {"dark": "github", "light": "github-light"}},
	"alertmanager": {"base": "svg", "aliases": [], "colors": {"dark": "alertmanager-dark"}},
	"../bad": {"base": "svg", "aliases": []},
	"bad-variant": {"base": "svg", "aliases": [], "colors": {"light": "../bad"}}
}`

// countingRoundTripper counts requests before delegating to the mock, so tests can assert when the index is fetched
type countingRoundTripper struct {
	next  http.RoundTripper
	calls atomic.Int32
}

func (c *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return c.next.RoundTrip(req)
}

func newTestService(t *testing.T, mock *testutils.MockRoundTripper) (*Service, *countingRoundTripper) {
	t.Helper()

	transport := &countingRoundTripper{next: mock}
	return newService(Dependencies{HTTPClient: &http.Client{Transport: transport}, BaseURL: testBaseURL}), transport
}

func indexMock() *testutils.MockRoundTripper {
	return &testutils.MockRoundTripper{
		Responses: map[string]*http.Response{
			testBaseURL + "/metadata.json": testutils.NewMockResponse(http.StatusOK, testIndex), //nolint:bodyclose // mock response, no real body
		},
	}
}

func references(presets []logoPresetDto) []string {
	refs := make([]string, len(presets))
	for i, p := range presets {
		refs[i] = p.Reference
	}
	return refs
}

func TestSearch_Disabled(t *testing.T) {
	svc, transport := newTestService(t, indexMock())
	svc.baseURL = ""

	_, err := svc.Search(t.Context(), "nextcloud")

	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeLogoPresetsDisabled))
	assert.Zero(t, transport.calls.Load(), "a disabled icon library must not be contacted")
}

func TestSearch_Ranking(t *testing.T) {
	svc, _ := newTestService(t, indexMock())

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{name: "exact match ranks before prefix match", query: "nextcloud", want: []string{"nextcloud", "nextcloud-talk"}},
		{name: "substring matches come after prefix matches", query: "cloud", want: []string{"nextcloud", "nextcloud-talk", "owncloud"}},
		{name: "separators and case are ignored", query: "HOME assistant", want: []string{"home-assistant"}},
		{name: "aliases match last", query: "blogging", want: []string{"ghost"}},
		{name: "empty query lists everything alphabetically", query: "", want: []string{"alertmanager", "ghost", "github", "home-assistant", "nextcloud", "nextcloud-talk", "owncloud"}},
		{name: "no match", query: "jellyfin", want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presets, err := svc.Search(t.Context(), tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.want, references(presets))
		})
	}
}

func TestSearch_LimitsResults(t *testing.T) {
	presets := make([]preset, maxResults+10)
	for i := range presets {
		presets[i] = preset{searchReference: "app"}
	}

	assert.Len(t, search(presets, "app", maxResults), maxResults)
}

func TestSearch_BuildsPresets(t *testing.T) {
	svc, _ := newTestService(t, indexMock())

	presets, err := svc.Search(t.Context(), "")
	require.NoError(t, err)

	byReference := make(map[string]logoPresetDto, len(presets))
	for _, p := range presets {
		byReference[p.Reference] = p
	}

	// An icon without color variants is used for both themes
	nextcloud := byReference["nextcloud"]
	assert.Equal(t, testBaseURL+"/svg/nextcloud.svg", nextcloud.LogoURL)
	assert.Nil(t, nextcloud.DarkLogoURL)

	// The light-colored variant is the dark logo
	github := byReference["github"]
	assert.Equal(t, testBaseURL+"/svg/github.svg", github.LogoURL)
	require.NotNil(t, github.DarkLogoURL)
	assert.Equal(t, testBaseURL+"/svg/github-light.svg", *github.DarkLogoURL)

	// The dark-colored variant is the light logo and the base icon is the dark logo
	alertmanager := byReference["alertmanager"]
	assert.Equal(t, testBaseURL+"/svg/alertmanager-dark.svg", alertmanager.LogoURL)
	require.NotNil(t, alertmanager.DarkLogoURL)
	assert.Equal(t, testBaseURL+"/svg/alertmanager.svg", *alertmanager.DarkLogoURL)

	// A PNG-only icon uses PNG for both variants
	ghost := byReference["ghost"]
	assert.Equal(t, testBaseURL+"/png/ghost.png", ghost.LogoURL)
	require.NotNil(t, ghost.DarkLogoURL)
	assert.Equal(t, testBaseURL+"/png/ghost-light.png", *ghost.DarkLogoURL)

	// The name comes from an alias with the same spelling as the reference, or from the reference itself
	assert.Equal(t, "GitHub", github.Name)
	assert.Equal(t, "Home Assistant", byReference["home-assistant"].Name)

	// File names that aren't plain slugs are dropped
	assert.NotContains(t, byReference, "../bad")
	assert.NotContains(t, byReference, "bad-variant")
}

func TestSearch_CachesIndex(t *testing.T) {
	svc, transport := newTestService(t, indexMock())

	for range 3 {
		_, err := svc.Search(t.Context(), "nextcloud")
		require.NoError(t, err)
	}

	assert.Equal(t, int32(1), transport.calls.Load())
}

func TestSearch_UsesStaleIndexWhenRefreshFails(t *testing.T) {
	mock := indexMock()
	svc, _ := newTestService(t, mock)

	// Expire the cache immediately so the second search has to refetch
	svc.cache = utils.New[[]preset](time.Nanosecond)

	_, err := svc.Search(t.Context(), "nextcloud")
	require.NoError(t, err)

	mock.Err = errors.New("CDN unreachable")

	presets, err := svc.Search(t.Context(), "nextcloud")
	require.NoError(t, err)
	assert.Equal(t, []string{"nextcloud", "nextcloud-talk"}, references(presets))
}

func TestSearch_IndexUnavailable(t *testing.T) {
	mock := &testutils.MockRoundTripper{
		Responses: map[string]*http.Response{
			testBaseURL + "/metadata.json": testutils.NewMockResponse(http.StatusInternalServerError, ""), //nolint:bodyclose // mock response, no real body
		},
	}
	svc, _ := newTestService(t, mock)

	_, err := svc.Search(t.Context(), "nextcloud")

	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeLogoPresetsUnavailable))
}
