package mcp

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	// docsIndexURL is published by the docs website and lists every page as Markdown
	docsIndexURL     = "https://pocket-id.org/docs-index.json"
	docsSiteURL      = "https://pocket-id.org/docs/"
	docsIndexTTL     = 24 * time.Hour
	docsFetchTimeout = 10 * time.Second
	maxDocsIndexSize = 10 << 20

	defaultDocsResults = 5
	maxDocsResults     = 10
	maxSectionLength   = 2000

	// BM25 parameters with their usual defaults
	bm25K1 = 1.2
	bm25B  = 0.75

	// Matches in a page title or section heading say more about a section than matches in its text
	titleWeight   = 3
	headingWeight = 2
)

// sectionHeading matches the headings a page is split at, which are the levels Starlight shows in its table of contents
var sectionHeading = regexp.MustCompile(`^#{2,3}\s+(.+?)\s*$`)

var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// docsPage is one page of the docs index
type docsPage struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

// docsSection is the part of a page below one heading, which is what a search returns
type docsSection struct {
	Title   string
	Heading string
	URL     string
	Content string

	terms  map[string]int
	length int
}

// docsCorpus holds the sections together with the statistics BM25 needs, computed once per download
type docsCorpus struct {
	sections          []docsSection
	documentFrequency map[string]int
	averageLength     float64
}

type docsIndex struct {
	httpClient *http.Client
	url        string
	cache      *utils.Cache[*docsCorpus]
}

func newDocsIndex(httpClient *http.Client, url string) *docsIndex {
	return &docsIndex{
		httpClient: httpClient,
		url:        url,
		cache:      utils.New[*docsCorpus](docsIndexTTL),
	}
}

type searchDocsInput struct {
	Query string `json:"query" binding:"required" jsonschema:"What to look for, such as a setting, an error message or the name of an app to connect to Pocket ID"`
	Limit int    `json:"limit,omitempty" jsonschema:"Number of sections to return, at most 10, 5 by default"`
}

type docsResult struct {
	Page    string `json:"page"`
	Section string `json:"section,omitempty"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type searchDocsOutput struct {
	Results []docsResult `json:"results"`
}

func registerDocsTool(t *Tools, index *docsIndex) {
	AddPublicTool(t, Tool{
		Name:        "search_docs",
		Description: "Search the Pocket ID documentation, including the guides for connecting apps such as Immich or Nextcloud. Returns the best matching sections with a link to each.",
		Effect:      ReadOnly,
	}, index.search)
}

func (d *docsIndex) search(ctx context.Context, _ Call, input searchDocsInput) (any, error) {
	if err := httpserver.Validate(&input); err != nil {
		return nil, err
	}

	corpus, err := d.cache.GetOrFetch(ctx, d.fetch)
	// A stale index is still good enough to search while the website is unreachable
	if staleErr, ok := errors.AsType[*utils.ErrStale](err); ok {
		slog.WarnContext(ctx, "Failed to refresh the docs index, using stale cache", slog.Any("error", staleErr.Err))
	} else if err != nil {
		slog.WarnContext(ctx, "Failed to load the docs index", slog.Any("error", err))
		return nil, apperror.DocsUnavailable(err)
	}

	limit := input.Limit
	if limit < 1 {
		limit = defaultDocsResults
	}

	sections := corpus.search(input.Query, min(limit, maxDocsResults))
	results := make([]docsResult, len(sections))
	for i, section := range sections {
		results[i] = docsResult{
			Page:    section.Title,
			Section: section.Heading,
			URL:     section.URL,
			Content: truncate(section.Content, maxSectionLength),
		}
	}
	return searchDocsOutput{Results: results}, nil
}

func (d *docsIndex) fetch(ctx context.Context) (*docsCorpus, error) {
	reqCtx, cancel := context.WithTimeout(ctx, docsFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, d.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create docs index request: %w", err)
	}
	req.Header.Set("User-Agent", "pocket-id/docs-search")
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch docs index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docs index returned status %d", resp.StatusCode)
	}

	var index struct {
		Pages []docsPage `json:"pages"`
	}
	err = json.NewDecoder(utils.NewLimitReader(resp.Body, maxDocsIndexSize)).Decode(&index)
	if err != nil {
		return nil, fmt.Errorf("decode docs index: %w", err)
	}

	return newDocsCorpus(index.Pages), nil
}

func newDocsCorpus(pages []docsPage) *docsCorpus {
	corpus := &docsCorpus{documentFrequency: make(map[string]int)}

	totalLength := 0
	for _, page := range pages {
		for _, section := range splitSections(page) {
			section.terms = make(map[string]int)
			addTerms(section.terms, page.Title, titleWeight)
			addTerms(section.terms, section.Heading, headingWeight)
			addTerms(section.terms, section.Content, 1)

			for term, frequency := range section.terms {
				section.length += frequency
				corpus.documentFrequency[term]++
			}
			totalLength += section.length
			corpus.sections = append(corpus.sections, section)
		}
	}

	if len(corpus.sections) > 0 {
		corpus.averageLength = float64(totalLength) / float64(len(corpus.sections))
	}
	return corpus
}

// splitSections splits a page at its headings, keeping the text before the first heading together with the page description as the page's introduction
func splitSections(page docsPage) []docsSection {
	pageURL := docsSiteURL + strings.Trim(page.Slug, "/")

	var sections []docsSection
	var heading, anchor string
	var content strings.Builder
	content.WriteString(page.Description + "\n\n")

	flush := func() {
		text := strings.TrimSpace(content.String())
		if text == "" && heading == "" {
			return
		}

		url := pageURL
		if anchor != "" {
			url += "#" + anchor
		}
		sections = append(sections, docsSection{Title: page.Title, Heading: heading, URL: url, Content: text})
		content.Reset()
	}

	// Headings repeat on pages like client examples, so later ones get the numeric suffix Starlight gives them
	seenAnchors := make(map[string]int)
	inCodeBlock := false
	for line := range strings.Lines(page.Body) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
		}

		// A line starting with # inside a code block is a shell comment, not a heading
		if match := sectionHeading.FindStringSubmatch(trimmed); match != nil && !inCodeBlock {
			flush()
			heading = plainHeading(match[1])
			anchor = slugify(heading)
			if count := seenAnchors[anchor]; count > 0 {
				seenAnchors[anchor]++
				anchor += "-" + strconv.Itoa(count)
			} else {
				seenAnchors[anchor] = 1
			}
			continue
		}

		content.WriteString(line)
	}
	flush()

	return sections
}

// plainHeading strips the inline Markdown that doesn't appear in the rendered heading
func plainHeading(heading string) string {
	heading = markdownLink.ReplaceAllString(heading, "$1")
	return strings.NewReplacer("`", "", "**", "", "*", "").Replace(heading)
}

// slugify builds a heading's anchor the way github-slugger does for Starlight
func slugify(heading string) string {
	var slug strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsMark(r), r == '-', r == '_':
			slug.WriteRune(r)
		case r == ' ':
			slug.WriteRune('-')
		}
	}
	return slug.String()
}

func addTerms(terms map[string]int, text string, weight int) {
	for _, term := range tokenize(text) {
		terms[term] += weight
	}
}

func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// search ranks the sections with BM25 and returns up to limit sections that match at least one query term
func (c *docsCorpus) search(query string, limit int) []docsSection {
	queryTerms := slices.Compact(slices.Sorted(slices.Values(tokenize(query))))

	type match struct {
		section docsSection
		score   float64
	}

	sectionCount := float64(len(c.sections))
	var matches []match
	for _, section := range c.sections {
		score := 0.0
		for _, term := range queryTerms {
			frequency := float64(section.terms[term])
			if frequency == 0 {
				continue
			}

			documentFrequency := float64(c.documentFrequency[term])
			idf := math.Log(1 + (sectionCount-documentFrequency+0.5)/(documentFrequency+0.5))
			lengthNorm := 1 - bm25B + bm25B*float64(section.length)/c.averageLength
			score += idf * frequency * (bm25K1 + 1) / (frequency + bm25K1*lengthNorm)
		}
		if score > 0 {
			matches = append(matches, match{section: section, score: score})
		}
	}

	// Ties are broken by URL so results are stable
	slices.SortFunc(matches, func(a, b match) int {
		return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.section.URL, b.section.URL))
	})

	result := make([]docsSection, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		result = append(result, m.section)
	}
	return result
}

func truncate(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "…"
}
