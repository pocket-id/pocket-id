package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

func TestSplitSections(t *testing.T) {
	sections := splitSections(docsPage{
		Slug:        "client-examples/immich",
		Title:       "Immich",
		Description: "Sign in to Immich with Pocket ID.",
		Body: "Intro text.\n\n" +
			"## Create the `client`\n\nAdd the callback URL.\n\n" +
			"```sh\n## not a heading\n```\n\n" +
			"### [Mobile](https://immich.app) app\n\nUse the app.\n\n" +
			"## Create the client\n\nAgain.\n",
	})

	require.Equal(t, []docsSection{
		{Title: "Immich", URL: "https://pocket-id.org/docs/client-examples/immich", Content: "Sign in to Immich with Pocket ID.\n\nIntro text."},
		{Title: "Immich", Heading: "Create the client", URL: "https://pocket-id.org/docs/client-examples/immich#create-the-client", Content: "Add the callback URL.\n\n```sh\n## not a heading\n```"},
		{Title: "Immich", Heading: "Mobile app", URL: "https://pocket-id.org/docs/client-examples/immich#mobile-app", Content: "Use the app."},
		{Title: "Immich", Heading: "Create the client", URL: "https://pocket-id.org/docs/client-examples/immich#create-the-client-1", Content: "Again."},
	}, sections)
}

func TestDocsSearchRanksTitlesAndHeadingsFirst(t *testing.T) {
	corpus := newDocsCorpus([]docsPage{
		{Slug: "configuration/environment-variables", Title: "Environment variables", Body: "## Database\n\nSet DB_CONNECTION_STRING. Immich is not configured here.\n"},
		{Slug: "client-examples/immich", Title: "Immich", Body: "## Configure Immich\n\nOpen the OAuth settings in Immich.\n"},
		{Slug: "setup/installation", Title: "Installation", Body: "## Docker\n\nRun the container.\n"},
	})

	results := corpus.search("immich oauth", 5)

	require.Len(t, results, 2)
	require.Equal(t, "https://pocket-id.org/docs/client-examples/immich#configure-immich", results[0].URL)
	require.Equal(t, "https://pocket-id.org/docs/configuration/environment-variables#database", results[1].URL)
	require.Empty(t, corpus.search("kubernetes", 5))
}

func TestSearchDocsTool(t *testing.T) {
	available := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !available {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pages": []docsPage{
			{Slug: "advanced/mcp", Title: "MCP server", Description: "Connect AI assistants to Pocket ID.", Body: "## Set up\n\nAllow the client.\n"},
		}})
	}))
	t.Cleanup(server.Close)

	index := newDocsIndex(server.Client(), server.URL)

	output, err := index.search(t.Context(), Call{}, searchDocsInput{Query: "assistants"})
	require.NoError(t, err)
	require.Equal(t, searchDocsOutput{Results: []docsResult{{
		Page:    "MCP server",
		URL:     "https://pocket-id.org/docs/advanced/mcp",
		Content: "Connect AI assistants to Pocket ID.",
	}}}, output)

	_, err = index.search(t.Context(), Call{}, searchDocsInput{})
	require.Error(t, err, "an empty query is rejected")

	t.Run("reports an unreachable website", func(t *testing.T) {
		available = false
		_, err := newDocsIndex(server.Client(), server.URL).search(t.Context(), Call{}, searchDocsInput{Query: "assistants"})
		require.True(t, apperror.IsCode(err, apperror.CodeDocsUnavailable))
	})
}
