package outbound

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strings"
)

const (
	KeywordPrivate  = "private"
	KeywordLoopback = "loopback"
	KeywordNone     = "none"
)

// privatePrefixes are the ranges the "private" keyword expands to, in addition to LOCAL_IPV6_RANGES
var privatePrefixes = mustParsePrefixes(
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",
	"fc00::/7",
)

// loopbackPrefixes are the ranges the "loopback" keyword expands to
var loopbackPrefixes = mustParsePrefixes(
	"127.0.0.0/8",
	"::1/128",
)

// specialUsePrefixes are reserved ranges that netip's classification methods do not catch
var specialUsePrefixes = mustParsePrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.31.196.0/24",
	"192.52.193.0/24",
	"192.88.99.0/24",
	"192.175.48.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"100:0:0:1::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"2620:4f:8000::/48",
	"3fff::/20",
	"5f00::/16",
	"fec0::/10",
	"ff00::/8",
)

// Policy decides which destinations an outbound request may reach
type Policy struct {
	// localIPv6 holds LOCAL_IPV6_RANGES, which are public IPv6 ranges the operator uses on the local network
	localIPv6 []netip.Prefix
	// allowedPrefixes are the non-public ranges the allowlist opens up
	allowedPrefixes []netip.Prefix
	// allowedHosts are exact hostnames whose resolved addresses are trusted
	allowedHosts []string
	// allowedHostSuffixes come from "*.example.com" entries and include the leading dot
	allowedHostSuffixes []string
}

// ParsePolicy parses a comma-separated allowlist of keywords, IP addresses, CIDR ranges and hostnames
func ParsePolicy(value string, localIPv6 []netip.Prefix) (Policy, error) {
	policy := Policy{localIPv6: localIPv6}

	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}

		var err error
		policy, err = policy.withEntry(entry)
		if err != nil {
			return Policy{}, err
		}
	}

	return policy, nil
}

// withEntry returns a copy of the policy that also allows the given allowlist entry
func (p Policy) withEntry(entry string) (Policy, error) {
	// Keywords expand to fixed sets of ranges
	switch entry {
	case KeywordNone:
		return p, nil
	case KeywordPrivate:
		p.allowedPrefixes = append(p.allowedPrefixes, privatePrefixes...)
		p.allowedPrefixes = append(p.allowedPrefixes, p.localIPv6...)
		return p, nil
	case KeywordLoopback:
		p.allowedPrefixes = append(p.allowedPrefixes, loopbackPrefixes...)
		return p, nil
	}

	// CIDR ranges and single addresses are compared against the IP that is actually dialed
	if strings.Contains(entry, "/") {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return Policy{}, fmt.Errorf("'%s' is not a valid CIDR range", entry)
		}
		p.allowedPrefixes = append(p.allowedPrefixes, prefix.Masked())
		return p, nil
	}
	if addr, err := netip.ParseAddr(entry); err == nil {
		addr = addr.Unmap()
		p.allowedPrefixes = append(p.allowedPrefixes, netip.PrefixFrom(addr, addr.BitLen()))
		return p, nil
	}

	// Anything else must be a hostname, optionally with a leading wildcard label
	if suffix, ok := strings.CutPrefix(entry, "*."); ok {
		if !isValidHostname(suffix) {
			return Policy{}, fmt.Errorf("'%s' is not a valid hostname pattern", entry)
		}
		p.allowedHostSuffixes = append(p.allowedHostSuffixes, "."+suffix)
		return p, nil
	}
	if !isValidHostname(entry) {
		return Policy{}, fmt.Errorf("'%s' is not a valid keyword, IP address, CIDR range or hostname", entry)
	}
	p.allowedHosts = append(p.allowedHosts, entry)
	return p, nil
}

// AllowAddr reports whether a connection to the address is permitted
func (p Policy) AllowAddr(addr netip.Addr) bool {
	// Zoned addresses are never contained in a prefix, and IPv4-mapped IPv6 addresses must match IPv4 rules
	addr = addr.WithZone("").Unmap()
	if !addr.IsValid() {
		return false
	}

	if !p.isBlocked(addr) {
		return true
	}

	for _, prefix := range p.allowedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// AllowURL reports whether the URL's host is trusted without checking the addresses it resolves to
func (p Policy) AllowURL(u *url.URL) bool {
	host := normalizeHostname(u.Hostname())
	if host == "" {
		return false
	}

	if slices.Contains(p.allowedHosts, host) {
		return true
	}
	for _, suffix := range p.allowedHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}

// hasTrustedHosts reports whether some requests may skip the address check
func (p Policy) hasTrustedHosts() bool {
	return len(p.allowedHosts) > 0 || len(p.allowedHostSuffixes) > 0
}

// isBlocked reports whether the address is anything other than a public unicast address
func (p Policy) isBlocked(addr netip.Addr) bool {
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsMulticast() {
		return true
	}

	for _, prefix := range specialUsePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	for _, prefix := range p.localIPv6 {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// normalizeHostname lowercases a hostname and drops the trailing dot of a fully qualified name
func normalizeHostname(host string) string {
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

// isValidHostname accepts DNS names made of letters, digits, hyphens, underscores and dots
func isValidHostname(host string) bool {
	if host == "" || len(host) > 253 || strings.HasPrefix(host, ".") || strings.Contains(host, "..") {
		return false
	}

	for _, r := range host {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '.' && r != '_' {
			return false
		}
	}

	return true
}

// ParseLocalIPv6Ranges parses the comma-separated LOCAL_IPV6_RANGES value
func ParseLocalIPv6Ranges(value string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, err
		}
		if !prefix.Addr().Is6() || prefix.Addr().Is4In6() {
			return nil, errors.New("range '" + entry + "' is not a valid IPv6 range")
		}
		prefixes = append(prefixes, prefix.Masked())
	}

	return prefixes, nil
}

func mustParsePrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, len(values))
	for i, value := range values {
		prefixes[i] = netip.MustParsePrefix(value)
	}
	return prefixes
}
