package utils

import (
	"errors"
	"net"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	patternParseSchemePlaceholder = "https"
	patternParsePortPlaceholder   = "65535"
)

var errInvalidCallbackURLPattern = errors.New("invalid callback URL pattern")

type callbackURLPattern struct {
	SchemePattern   string
	HasUserInfo     bool
	UsernamePattern string
	HasPassword     bool
	PasswordPattern string
	HostnamePattern string
	HasPort         bool
	PortPattern     string
	PathPattern     string
}

type callbackURLValue struct {
	Scheme      string
	HasUserInfo bool
	Username    string
	HasPassword bool
	Password    string
	Hostname    string
	HasPort     bool
	Port        string
	Path        string
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

// ValidateCallbackURLPattern validates callback URL patterns, including wildcard patterns.
func ValidateCallbackURLPattern(raw string) error {
	if raw == "*" {
		return nil
	}

	raw, _, _ = strings.Cut(raw, "#")
	base, rawQuery, hasQuery := strings.Cut(raw, "?")

	if hasQuery {
		query, err := url.ParseQuery(rawQuery)
		if err != nil {
			return err
		}
		for _, values := range query {
			for _, value := range values {
				if err := validateGlobPattern(value); err != nil {
					return err
				}
			}
		}
	}

	pattern, err := parseCallbackURLPattern(base)
	if err != nil {
		return err
	}

	if err := validateGlobPattern(pattern.SchemePattern); err != nil {
		return err
	}
	if pattern.HasUserInfo {
		if err := validateGlobPattern(pattern.UsernamePattern); err != nil {
			return err
		}
	}
	if pattern.HasPassword {
		if err := validateGlobPattern(pattern.PasswordPattern); err != nil {
			return err
		}
	}
	for _, segment := range splitHostLabels(pattern.HostnamePattern) {
		if err := validateGlobPattern(segment); err != nil {
			return err
		}
	}
	if pattern.HasPort {
		if err := validateGlobPattern(pattern.PortPattern); err != nil {
			return err
		}
	}

	return nil
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

	patternBase, rawPatternQuery, patternHasQuery := strings.Cut(pattern, "?")
	inputBase, rawInputQuery, inputHasQuery := strings.Cut(inputCallbackURL, "?")

	// Store and parse query parts.
	var patternQuery url.Values
	if patternHasQuery {
		patternQuery, err = url.ParseQuery(rawPatternQuery)
		if err != nil {
			return false, err
		}
	}
	var inputQuery url.Values
	if inputHasQuery {
		inputQuery, err = url.ParseQuery(rawInputQuery)
		if err != nil {
			return false, err
		}
	}

	patternURL, err := parseCallbackURLPattern(patternBase)
	if err != nil {
		return false, nil
	}

	inputURL, err := parseCallbackURLValue(inputBase)
	if err != nil {
		return false, nil
	}

	// Verify scheme.
	matched, err := path.Match(patternURL.SchemePattern, inputURL.Scheme)
	if err != nil || !matched {
		return false, err
	}

	// Verify userinfo.
	if patternURL.HasUserInfo != inputURL.HasUserInfo {
		return false, nil
	}
	if patternURL.HasUserInfo {
		matched, err = path.Match(patternURL.UsernamePattern, inputURL.Username)
		if err != nil || !matched {
			return false, err
		}

		if patternURL.HasPassword != inputURL.HasPassword {
			return false, nil
		}
		if patternURL.HasPassword {
			matched, err = path.Match(patternURL.PasswordPattern, inputURL.Password)
			if err != nil || !matched {
				return false, err
			}
		}
	}

	// Verify host.
	matched, err = matchHostPattern(patternURL.HostnamePattern, inputURL.Hostname)
	if err != nil || !matched {
		return false, err
	}

	// Verify port.
	if patternURL.HasPort != inputURL.HasPort {
		return false, nil
	}
	if patternURL.HasPort {
		matched, err = path.Match(patternURL.PortPattern, inputURL.Port)
		if err != nil || !matched {
			return false, err
		}
	}

	// Verify path with wildcard support.
	matched, err = matchPath(patternURL.PathPattern, inputURL.Path)
	if err != nil || !matched {
		return false, err
	}

	// Verify query parameters.
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
			matched, err = path.Match(patternValues[i], inputValues[i])
			if err != nil || !matched {
				return false, err
			}
		}
	}

	return true, nil
}

func parseCallbackURLPattern(raw string) (callbackURLPattern, error) {
	schemePattern, rest, hasScheme := strings.Cut(raw, "://")
	if !hasScheme || schemePattern == "" {
		return callbackURLPattern{}, errInvalidCallbackURLPattern
	}

	authority := rest
	pathPattern := ""
	if i := strings.IndexRune(rest, '/'); i >= 0 {
		authority = rest[:i]
		pathPattern = rest[i:]
	}
	if authority == "" {
		return callbackURLPattern{}, errInvalidCallbackURLPattern
	}

	userinfoPattern, hostPort, hasUserinfo := splitAuthority(authority)
	if hostPort == "" {
		return callbackURLPattern{}, errInvalidCallbackURLPattern
	}

	hasPassword := false
	usernamePattern := ""
	passwordPattern := ""
	if hasUserinfo {
		usernamePattern, passwordPattern, hasPassword = strings.Cut(userinfoPattern, ":")
	}

	sanitizedAuthority, wildcardPortPattern, err := sanitizePatternAuthority(authority)
	if err != nil {
		return callbackURLPattern{}, err
	}

	sanitizedScheme := schemePattern
	if strings.ContainsRune(schemePattern, '*') {
		sanitizedScheme = patternParseSchemePlaceholder
	}

	sanitizedURL := sanitizedScheme + "://" + sanitizedAuthority + pathPattern
	u, err := url.Parse(sanitizedURL)
	if err != nil || !u.IsAbs() || u.Hostname() == "" {
		return callbackURLPattern{}, errInvalidCallbackURLPattern
	}

	portPattern := u.Port()
	hasPort := portPattern != ""
	if wildcardPortPattern != "" {
		portPattern = wildcardPortPattern
		hasPort = true
	}

	return callbackURLPattern{
		SchemePattern:   schemePattern,
		HasUserInfo:     hasUserinfo,
		UsernamePattern: usernamePattern,
		HasPassword:     hasPassword,
		PasswordPattern: passwordPattern,
		HostnamePattern: u.Hostname(),
		HasPort:         hasPort,
		PortPattern:     portPattern,
		PathPattern:     pathPattern,
	}, nil
}

func parseCallbackURLValue(raw string) (callbackURLValue, error) {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Hostname() == "" {
		return callbackURLValue{}, errInvalidCallbackURLPattern
	}

	hasUserinfo := u.User != nil
	username := ""
	password := ""
	hasPassword := false
	if hasUserinfo {
		username = u.User.Username()
		password, hasPassword = u.User.Password()
	}

	resolvedPath := u.EscapedPath()
	if resolvedPath == "" {
		resolvedPath = u.Path
	}

	return callbackURLValue{
		Scheme:      u.Scheme,
		HasUserInfo: hasUserinfo,
		Username:    username,
		HasPassword: hasPassword,
		Password:    password,
		Hostname:    u.Hostname(),
		HasPort:     u.Port() != "",
		Port:        u.Port(),
		Path:        resolvedPath,
	}, nil
}

func sanitizePatternAuthority(authority string) (sanitizedAuthority string, wildcardPortPattern string, err error) {
	userinfo, hostPort, hasUserinfo := splitAuthority(authority)
	if hostPort == "" {
		return "", "", errInvalidCallbackURLPattern
	}

	sanitizedHostPort := hostPort

	if strings.HasPrefix(hostPort, "[") {
		end := strings.Index(hostPort, "]")
		if end < 0 {
			return "", "", errInvalidCallbackURLPattern
		}

		rest := hostPort[end+1:]
		if rest != "" {
			if !strings.HasPrefix(rest, ":") {
				return "", "", errInvalidCallbackURLPattern
			}

			port := rest[1:]
			if port == "" {
				return "", "", errInvalidCallbackURLPattern
			}

			if strings.Contains(port, "*") {
				sanitizedHostPort = hostPort[:end+1] + ":" + patternParsePortPlaceholder
				wildcardPortPattern = port
			}
		}
	} else {
		lastColon := strings.LastIndex(hostPort, ":")
		if lastColon >= 0 {
			hostCandidate := hostPort[:lastColon]
			portCandidate := hostPort[lastColon+1:]

			isBareIPv6WithoutPort := strings.Count(hostPort, ":") > 1 && net.ParseIP(hostPort) != nil
			if !isBareIPv6WithoutPort {
				if hostCandidate == "" || portCandidate == "" {
					return "", "", errInvalidCallbackURLPattern
				}

				if strings.Contains(portCandidate, "*") {
					sanitizedHostPort = hostCandidate + ":" + patternParsePortPlaceholder
					wildcardPortPattern = portCandidate
				}
			}
		}
	}

	if hasUserinfo {
		sanitizedAuthority = userinfo + "@" + sanitizedHostPort
	} else {
		sanitizedAuthority = sanitizedHostPort
	}

	return sanitizedAuthority, wildcardPortPattern, nil
}

func splitAuthority(authority string) (userinfo string, hostPort string, hasUserinfo bool) {
	lastAt := strings.LastIndex(authority, "@")
	if lastAt < 0 {
		return "", authority, false
	}

	return authority[:lastAt], authority[lastAt+1:], true
}

func matchHostPattern(patternHost, inputHost string) (bool, error) {
	patternSegments := splitHostLabels(patternHost)
	inputSegments := splitHostLabels(inputHost)

	if len(patternSegments) != len(inputSegments) {
		return false, nil
	}

	for i := range patternSegments {
		matched, err := path.Match(patternSegments[i], inputSegments[i])
		if err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func splitHostLabels(host string) []string {
	if strings.Contains(host, ":") {
		return []string{host}
	}

	return strings.Split(host, ".")
}

func validateGlobPattern(pattern string) error {
	if _, err := path.Match(pattern, ""); err != nil {
		return err
	}

	return nil
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
