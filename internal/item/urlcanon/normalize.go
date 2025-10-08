package urlcanon

import (
	"net"
	"net/url"
	pathpkg "path"
	"strings"
)

var trackingParameters = map[string]struct{}{
	"fbclid": {},
	"gclid":  {},
	"gclsrc": {},
	"mc_cid": {},
	"mc_eid": {},
}

// Normalize canonicalizes the provided URL string for consistent storage and comparison.
// It trims whitespace, lowercases the scheme and host, strips a leading "www.", removes
// default ports and fragments, eliminates common tracking parameters, and removes trailing
// slashes. Invalid URLs are returned unchanged.
func Normalize(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return raw
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)

	host := strings.ToLower(parsed.Hostname())
	if strings.HasPrefix(host, "www.") && len(host) > len("www.") {
		host = strings.TrimPrefix(host, "www.")
	}

	port := parsed.Port()
	if port != "" {
		if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
			port = ""
		}
	}

	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	} else {
		if strings.Contains(host, ":") {
			parsed.Host = "[" + host + "]"
		} else {
			parsed.Host = host
		}
	}

	if parsed.Path != "" {
		cleaned := pathpkg.Clean(parsed.Path)
		if cleaned == "." {
			cleaned = ""
		}
		if cleaned == "/" {
			cleaned = ""
		}
		parsed.Path = strings.TrimSuffix(cleaned, "/")
	}

	parsed.Fragment = ""

	if parsed.RawQuery != "" {
		query := parsed.Query()
		for key := range query {
			lower := strings.ToLower(key)
			if strings.HasPrefix(lower, "utm_") || isTrackingParameter(lower) {
				query.Del(key)
			}
		}
		if len(query) == 0 {
			parsed.RawQuery = ""
		} else {
			parsed.RawQuery = query.Encode()
		}
	}

	return parsed.String()
}

func isTrackingParameter(name string) bool {
	_, ok := trackingParameters[name]
	return ok
}
