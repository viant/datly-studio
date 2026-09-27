package namespacevalidation

import "strings"

// ValidName enforces the Studio business-namespace grammar.
func ValidName(value string) bool {
	segments := strings.Split(strings.TrimSpace(value), ".")
	for _, segment := range segments {
		if segment == "" || len(segment) > 64 || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for _, character := range segment[1:] {
			if character != '_' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return false
			}
		}
	}
	return len(value) <= 200
}
