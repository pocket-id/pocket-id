package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderFormPost(t *testing.T, params url.Values) string {
	t.Helper()

	var buf strings.Builder
	err := formPostTemplate.Execute(&buf, struct {
		RedirURL   string
		Parameters url.Values
	}{
		RedirURL:   "https://client.example.com/callback",
		Parameters: params,
	})
	require.NoError(t, err)
	return buf.String()
}

// The form_post page must auto-submit without an inline event handler
// fosite's default template uses <body onload="..."> which our CSP rejects, leaving the user on a blank page with no callback
func TestFormPostTemplateHasNoInlineEventHandler(t *testing.T) {
	html := renderFormPost(t, url.Values{"code": {"the-code"}, "state": {"the-state"}})

	assert.NotContains(t, html, "onload", "form_post page must not rely on an inline onload handler")
	assert.Contains(t, html, `action="https://client.example.com/callback"`)
	assert.Contains(t, html, `name="code"`)
	assert.Contains(t, html, `value="the-code"`)
	assert.Contains(t, html, `name="state"`)
}

// formPostScriptCSPHash must match the inline script the template actually renders
// If the two ever drift, the browser refuses the script and the form never submits, which is the exact regression this guards against
func TestFormPostScriptCSPHashMatchesRenderedScript(t *testing.T) {
	html := renderFormPost(t, url.Values{})

	// Extract the inline script body the browser would compute the hash over
	matches := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindStringSubmatch(html)
	require.Len(t, matches, 2, "rendered form_post page must contain exactly one inline <script> block")
	scriptBody := matches[1]

	sum := sha256.Sum256([]byte(scriptBody))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	assert.Equal(t, want, formPostScriptCSPHash)
}
