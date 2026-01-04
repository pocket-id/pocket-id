package utils

import (
	"net"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// GetCallbackURLFromList returns the first callback URL that matches the input callback URL
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
			u.Host = host
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

	// Strip fragment part
	// The endpoint URI MUST NOT include a fragment component.
	// https://datatracker.ietf.org/doc/html/rfc6749#section-3.1.2
	pattern, _, _ = strings.Cut(pattern, "#")
	inputCallbackURL, _, _ = strings.Cut(inputCallbackURL, "#")

	// Store and strip query part
	var patternQuery url.Values
	if i := strings.Index(pattern, "?"); i >= 0 {
		patternQuery, err = url.ParseQuery(pattern[i+1:])
		if err != nil {
			return false, err
		}
		pattern = pattern[:i]
	}
	var inputQuery url.Values
	if i := strings.Index(inputCallbackURL, "?"); i >= 0 {
		inputQuery, err = url.ParseQuery(inputCallbackURL[i+1:])
		if err != nil {
			return false, err
		}
		inputCallbackURL = inputCallbackURL[:i]
	}

	// Split both pattern and input parts
	patternParts, patternPath := splitParts(pattern)
	inputParts, inputPath := splitParts(inputCallbackURL)

	// Verify everything except the path and query parameters
	if len(patternParts) != len(inputParts) {
		return false, nil
	}

	for i, patternPart := range patternParts {
		matched, err := path.Match(patternPart, inputParts[i])
		if err != nil || !matched {
			return false, err
		}
	}

	// Verify path with wildcard support
	matched, err := matchPath(patternPath, inputPath)
	if err != nil || !matched {
		return false, err
	}

	// Verify query parameters
	if len(patternQuery) != len(inputQuery) {
		return false, nil
	}

	for patternKey, patternValues := range patternQuery {
		inputValues, exists := inputQuery[patternKey]
		if !exists {
			return false, nil
		}

		if len(patternValues) != len(inputValues) {
			return false, nil
		}

		for i := range patternValues {
			matched, err := path.Match(patternValues[i], inputValues[i])
			if err != nil || !matched {
				return false, err
			}
		}
	}

	return true, nil
}

// matchPath matches the input path against the pattern with wildcard support
// Supported wildcards:
//
//	'*'  matches any sequence of characters except '/'
//	'**' matches any sequence of characters including '/'
func matchPath(pattern string, input string) (matches bool, err error) {
	var regexPattern strings.Builder
	regexPattern.WriteString("^")

	runes := []rune(pattern)
	n := len(runes)

	for i := 0; i < n; {
		switch runes[i] {
		case '*':
			// Check if it's a ** (globstar)
			if i+1 < n && runes[i+1] == '*' {
				// globstar = .* (match slashes too)
				regexPattern.WriteString(".*")
				i += 2
			} else {
				// single * = [^/]* (no slash)
				regexPattern.WriteString(`[^/]*`)
				i++
			}
		default:
			regexPattern.WriteString(regexp.QuoteMeta(string(runes[i])))
			i++
		}
	}

	regexPattern.WriteString("$")

	matched, err := regexp.MatchString(regexPattern.String(), input)
	return matched, err
}

// splitParts splits the URL into parts by special characters and returns the path separately
func splitParts(s string) (parts []string, path string) {
	split := func(r rune) bool {
		return r == ':' || r == '/' || r == '[' || r == ']' || r == '@' || r == '.'
	}

	pathStart := -1

	// Look for scheme:// first
	if i := strings.Index(s, "://"); i >= 0 {
		// Look for the next slash after scheme://
		rest := s[i+3:]
		if j := strings.IndexRune(rest, '/'); j >= 0 {
			pathStart = i + 3 + j
		}
	} else {
		// Otherwise, first slash is path start
		pathStart = strings.IndexRune(s, '/')
	}

	if pathStart >= 0 {
		path = s[pathStart:]
		base := s[:pathStart]
		parts = strings.FieldsFunc(base, split)
	} else {
		parts = strings.FieldsFunc(s, split)
		path = ""
	}

	return parts, path
}
