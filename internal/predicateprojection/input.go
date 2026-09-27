package predicateprojection

import (
	"encoding/json"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

func ValidName(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for index, char := range part {
			if index == 0 && (char < 'a' || char > 'z') {
				return false
			}
			if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' {
				continue
			}
			return false
		}
	}
	return true
}

func ScopeJSON(alias string, columns []string) (string, error) {
	alias = strings.TrimSpace(alias)
	clean := make([]string, 0, len(columns))
	for _, column := range columns {
		if column = strings.TrimSpace(column); column != "" {
			clean = append(clean, column)
		}
	}
	if alias == "" && len(clean) == 0 {
		return "", nil
	}
	payload, err := json.Marshal(sdk.AuthorizationPredicateSQLMetadata{Alias: alias, Columns: clean})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
