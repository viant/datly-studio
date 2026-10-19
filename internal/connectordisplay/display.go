// Package connectordisplay formats saved connector configuration for inspection.
// It never resolves SCY resources or substitutes credentials into a template.
package connectordisplay

import (
	"net/url"
	"regexp"
	"strings"
)

const hidden = "[redacted]"

var placeholder = regexp.MustCompile(`^\$(?:[A-Za-z_][A-Za-z0-9_]*|\{[A-Za-z_][A-Za-z0-9_]*\})$`)
var embeddedPlaceholder = regexp.MustCompile(`\$(?:[A-Za-z_][A-Za-z0-9_]*|\{[A-Za-z_][A-Za-z0-9_]*\})`)
var credentialOption = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|access_token|refresh_token|id_token|secret|client_secret|api_key|apikey|private_key)\s*=\s*(?:'[^']*'|"[^"]*"|[^&;\s]*)`)

func mask(value string) string {
	if value == "" || placeholder.MatchString(value) {
		return value
	}
	return hidden
}
func options(value string) string {
	return credentialOption.ReplaceAllStringFunc(value, func(item string) string {
		i := strings.IndexByte(item, '=')
		return item[:i+1] + mask(strings.TrimSpace(item[i+1:]))
	})
}
func DSN(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(value), "inlined:") {
		return hidden
	}
	if strings.Contains(value, "@tcp(") || strings.Contains(value, "@unix(") {
		at := strings.LastIndex(value, "@")
		prefix := value[:at]
		if colon := strings.IndexByte(prefix, ':'); colon >= 0 {
			prefix = prefix[:colon+1] + mask(prefix[colon+1:])
		}
		return options(prefix + value[at:])
	}
	parsed, err := url.Parse(embeddedPlaceholder.ReplaceAllStringFunc(value, url.QueryEscape))
	if err == nil && parsed.Scheme != "" {
		if parsed.Opaque != "" && parsed.Scheme != "file" && parsed.Scheme != "sqlite" {
			return hidden
		}
		query := parsed.Query()
		changed := false
		for key, values := range query {
			if credentialOption.MatchString(key + "=x") {
				for i, value := range values {
					safe := mask(value)
					changed = changed || safe != value
					values[i] = safe
				}
				query[key] = values
			}
		}
		if changed {
			parsed.RawQuery = query.Encode()
			value = parsed.String()
		}

		if parsed.User != nil {
			username := parsed.User.Username()
			password, hasPassword := parsed.User.Password()
			if hasPassword && mask(password) != password {
				parsed.User = url.UserPassword(username, mask(password))
				value = parsed.String()
				value = strings.ReplaceAll(value, "%5Bredacted%5D", hidden)
			}
		}
		return options(value)
	}
	// Keyword DSNs (PostgreSQL/ODBC) are inspectable with credential values masked.
	if strings.Contains(value, "=") {
		return options(value)
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") || strings.HasSuffix(value, ".db") {
		return value
	}
	return hidden
}
func SecretReference(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~/") {
		return options(value)
	}
	parts := strings.SplitN(value, "|", 2)
	parsed, err := url.Parse(parts[0])
	if err != nil || parsed.Scheme == "" || parsed.Scheme == "inlined" || parsed.Scheme == "data" {
		return hidden
	}
	return DSN(value)
}
