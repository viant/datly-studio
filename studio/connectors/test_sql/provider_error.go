package test_sql

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/api/googleapi"
)

const maxProviderMessage = 12 << 10
const maxProviderInput = 64 << 10

// ProviderError is returned only after connector SQL-editor authorization. It
// contains native provider fields, never wrapper errors, DSNs or HTTP bodies.
type ProviderError struct {
	Provider  string `json:"provider"`
	Code      string `json:"code,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Location  string `json:"location,omitempty"`
	SQLState  string `json:"sqlState,omitempty"`
	Message   string `json:"message"`
	Truncated bool   `json:"truncated,omitempty"`
}

type terminalBigQueryError interface {
	BigQueryReason() string
	BigQueryMessage() string
	BigQueryLocation() string
}

func providerExecutionError(err error) *ProviderError {
	var result ProviderError
	var job terminalBigQueryError
	var api *googleapi.Error
	var my *mysql.MySQLError
	switch {
	case errors.As(err, &job):
		result = ProviderError{Provider: "BigQuery", Code: job.BigQueryReason(), Reason: job.BigQueryReason(), Location: job.BigQueryLocation(), Message: job.BigQueryMessage()}
	case errors.As(err, &api):
		result = ProviderError{Provider: "Google API", Code: strconv.Itoa(api.Code), Message: api.Message}
		messages := []string{api.Message}
		for _, item := range api.Errors {
			if result.Reason == "" {
				result.Reason = item.Reason
			}
			messages = append(messages, item.Message)
		}
		// GoogleAPI's ErrorItem omits location. Read only the documented provider
		// error fields from a bounded body; headers and arbitrary details stay private.
		if len(api.Body) <= maxProviderInput {
			var body struct {
				Error struct {
					Errors []struct{ Message, Reason, Location string }
				}
			}
			if json.Unmarshal([]byte(api.Body), &body) == nil {
				for _, item := range body.Error.Errors {
					if result.Location == "" {
						result.Location = item.Location
					}
					if result.Reason == "" {
						result.Reason = item.Reason
					}
					messages = append(messages, item.Message)
				}
			}
		}
		result.Message = uniqueProviderMessages(messages)
	case errors.As(err, &my):
		result = ProviderError{Provider: "MySQL", Code: strconv.Itoa(int(my.Number)), Message: my.Message, SQLState: strings.Trim(string(my.SQLState[:]), "\x00")}
	default:
		return nil
	}
	if strings.TrimSpace(result.Message) == "" {
		return nil
	}
	result.Message, result.Truncated = redactProviderText(result.Message, maxProviderMessage)
	result.Code, _ = redactProviderText(result.Code, 128)
	result.Reason, _ = redactProviderText(result.Reason, 256)
	result.Location, _ = redactProviderText(result.Location, 1024)
	result.SQLState, _ = redactProviderText(result.SQLState, 16)
	return &result
}

func uniqueProviderMessages(values []string) string {
	seen := map[string]bool{}
	var result []string
	size := 0
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
		size += len(value)
		if size > maxProviderInput {
			break
		}
	}
	return strings.Join(result, "\n")
}

func (e *ProviderError) summary() string {
	label := e.Provider + " error"
	if e.Code != "" {
		label += " " + e.Code
	}
	detail := e.Reason
	if e.SQLState != "" {
		detail = e.SQLState
	}
	if detail != "" && detail != e.Code {
		label += " (" + detail + ")"
	}
	if e.Location != "" {
		label += " at " + e.Location
	}
	return label + ": " + e.Message
}

var providerCredentials = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|access_token|refresh_token|id_token|secret|client_secret|api_key|apikey|private_key|credurl|credentials?)\b["']?\s*[:=]\s*(?:"(?:\\.|[^"\\])*"|'[^']*'|[^\s&;,]+)`)
var providerAuthorization = regexp.MustCompile(`(?i)\b(Bearer|Basic)\s+[A-Za-z0-9._~+/=-]+`)
var providerPrivateKey = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`)
var providerJWT = regexp.MustCompile(`\b[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{8,}\b`)
var providerURL = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s"'<>]+`)
var providerMySQLDSN = regexp.MustCompile(`[^\s"':]+:[^\s"'@]+@(?:tcp|unix)\([^)]*\)[^\s"']*`)

func redactProviderText(value string, limit int) (string, bool) {
	truncated := len(value) > maxProviderInput
	if truncated {
		value = value[:maxProviderInput]
	}
	value = providerPrivateKey.ReplaceAllString(value, "[redacted private key]")
	value = providerAuthorization.ReplaceAllString(value, "$1 [redacted]")
	value = providerJWT.ReplaceAllString(value, "[redacted token]")
	value = providerMySQLDSN.ReplaceAllString(value, "[redacted connection]")
	value = providerURL.ReplaceAllStringFunc(value, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "[redacted URL]"
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "[redacted connection]"
		}
		parsed.User = nil
		query := parsed.Query()
		for key := range query {
			if providerCredentials.MatchString(key + "=x") {
				query.Set(key, "[redacted]")
			}
		}
		parsed.RawQuery = query.Encode()
		return parsed.String()
	})
	value = providerCredentials.ReplaceAllStringFunc(value, func(raw string) string {
		i := strings.IndexAny(raw, ":=")
		return raw[:i+1] + "[redacted]"
	})
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, value)
	if len(value) > limit {
		value = value[:limit]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
		truncated = true
	}
	if truncated {
		value += "\n[provider message truncated]"
	}
	return value, truncated
}
