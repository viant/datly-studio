package bffauth

import (
	"net/url"
	"strings"
	"unicode"
)

// loginReturnTo accepts only an origin-relative browser location. The target
// stays inside the existing authenticated OAuth state, never callback arguments.
func loginReturnTo(values []string) string {
	if len(values) != 1 {
		return "/"
	}
	value := values[0]
	if len(value) > 2048 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Opaque != "" {
		return "/"
	}
	decoded := parsed.Path
	for attempts := 0; attempts < 4; attempts++ {
		if strings.HasPrefix(decoded, "//") || strings.Contains(decoded, "\\") || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
			return "/"
		}
		next, e := url.PathUnescape(decoded)
		if e != nil {
			return "/"
		}
		if next == decoded {
			return value
		}
		decoded = next
	}
	return "/"
}
