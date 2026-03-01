package utils

import (
	"log/slog"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/dunglas/go-urlpattern"
)

// ValidateCallbackURLPattern checks if the given callback URL pattern
// is valid according to the rules defined in this package.
func ValidateCallbackURLPattern(pattern string) error {
	if pattern == "*" {
		return nil
	}

	pattern, _, _ = strings.Cut(pattern, "#")
	pattern = normalizeToURLPatternStandard(pattern)

	_, err := urlpattern.New(pattern, "", nil)
	return err
}

// GetCallbackURLFromList returns the first callback URL that matches the input callback URL.
func GetCallbackURLFromList(urls []string, inputCallbackURL string) (callbackURL string, err error) {
	// Special case for Loopback Interface Redirection. Quoting from RFC 8252 section 7.3:
	// https://datatracker.ietf.org/doc/html/rfc8252#section-7.3
	//
	//	 The authorization server MUST allow any port to be specified at the
	//   time of the request for loopback IP redirect URIs, to accommodate
	//   clients that obtain an available ephemeral port from the operating
	//   system at the time of the request.
	loopbackCallbackURLWithoutPort := ""
	u, _ := url.Parse(inputCallbackURL)

	if u != nil && u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host == "localhost" || (ip != nil && ip.IsLoopback()) {
			// For IPv6 loopback hosts, brackets are required when serializing without a port.
			if strings.Contains(host, ":") {
				u.Host = "[" + host + "]"
			} else {
				u.Host = host
			}
			loopbackCallbackURLWithoutPort = u.String()
		}
	}

	for _, pattern := range urls {
		// Try the original callback first
		matches, err := matchCallbackURL(pattern, inputCallbackURL)
		if err != nil {
			return "", err
		}
		if matches {
			return inputCallbackURL, nil
		}

		// If we have a loopback variant, try that too
		if loopbackCallbackURLWithoutPort != "" {
			matches, err = matchCallbackURL(pattern, loopbackCallbackURLWithoutPort)
			if err != nil {
				return "", err
			}
			if matches {
				return inputCallbackURL, nil
			}
		}
	}

	return "", nil
}

// matchCallbackURL checks if the input callback URL matches the given pattern.
// It supports wildcard matching for paths and query parameters.
//
// The base URL (scheme, userinfo, host, port) and query parameters supports single '*' wildcards only,
// while the path supports both single '*' and double '**' wildcards.
func matchCallbackURL(pattern string, inputCallbackURL string) (matches bool, err error) {
	if pattern == inputCallbackURL || pattern == "*" {
		return true, nil
	}

	// Strip fragment part.
	// The endpoint URI MUST NOT include a fragment component.
	// https://datatracker.ietf.org/doc/html/rfc6749#section-3.1.2
	pattern, _, _ = strings.Cut(pattern, "#")
	inputCallbackURL, _, _ = strings.Cut(inputCallbackURL, "#")

	// Store and strip query part
	pattern, patternQuery, err := extractQueryParams(pattern)
	if err != nil {
		return false, err
	}

	inputCallbackURL, inputQuery, err := extractQueryParams(inputCallbackURL)
	if err != nil {
		return false, err
	}

	pattern = normalizeToURLPatternStandard(pattern)

	// Validate query params
	v := validateQueryParams(patternQuery, inputQuery)
	if !v {
		return false, nil
	}

	// Validate the rest of the URL using urlpattern
	p, err := urlpattern.New(pattern, "", nil)
	if err != nil {
		//nolint:nilerr
		slog.Warn("invalid callback URL pattern, skipping", "pattern", pattern, "error", err)
		return false, nil
	}

	return p.Test(inputCallbackURL, ""), nil
}

// normalizeToURLPatternStandard converts patterns with single asterisk wildcards and globstar wildcards
// into a format that can be parsed by the urlpattern package, which uses :param for single segment wildcards
// and ** for multi-segment wildcards.
func normalizeToURLPatternStandard(pattern string) string {
	patternBase, patternPath := extractPath(pattern)

	var result strings.Builder
	for i := 0; i < len(patternPath); i++ {
		if patternPath[i] == '*' {
			// Replace globstar with a single asterisk
			if i+1 < len(patternPath) && patternPath[i+1] == '*' {
				result.WriteString("*")
				i++ // skip next *
			} else {
				// Replace single asterisk with :p{index}
				result.WriteString(":p")
				result.WriteString(strconv.Itoa(i))
			}
		} else {
			result.WriteByte(patternPath[i])
		}
	}
	patternPath = result.String()

	return patternBase + patternPath
}

func extractPath(url string) (base string, path string) {
	pathStart := -1

	// Look for scheme:// first
	if i := strings.Index(url, "://"); i >= 0 {
		// Look for the next slash after scheme://
		rest := url[i+3:]
		if j := strings.IndexByte(rest, '/'); j >= 0 {
			pathStart = i + 3 + j
		}
	} else {
		// Otherwise, first slash is path start
		pathStart = strings.IndexByte(url, '/')
	}

	if pathStart >= 0 {
		path = url[pathStart:]
		base = url[:pathStart]
	} else {
		path = ""
		base = url
	}

	return base, path
}

func extractQueryParams(rawUrl string) (base string, query url.Values, err error) {
	if i := strings.IndexByte(rawUrl, '?'); i >= 0 {
		query, err = url.ParseQuery(rawUrl[i+1:])
		if err != nil {
			return "", nil, err
		}
		rawUrl = rawUrl[:i]
	}

	return rawUrl, query, nil
}

func validateQueryParams(patternQuery, inputQuery url.Values) bool {
	if len(patternQuery) != len(inputQuery) {
		return false
	}

	for patternKey, patternValues := range patternQuery {
		inputValues, exists := inputQuery[patternKey]
		if !exists {
			return false
		}

		if len(patternValues) != len(inputValues) {
			return false
		}

		for i := range patternValues {
			matched, err := path.Match(patternValues[i], inputValues[i])
			if err != nil || !matched {
				return false
			}
		}
	}

	return true
}
