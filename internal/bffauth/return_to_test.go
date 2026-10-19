package bffauth

import (
	"strings"
	"testing"
)

func TestLoginReturnToRemainsOnOrigin(t *testing.T) {
	for _, value := range []string{"/", "/?section=components#builder", "/studio/manage?section=window&query=a%20b#editor", "/report?q=https%3A%2F%2Felsewhere.example"} {
		if got := loginReturnTo([]string{value}); got != value {
			t.Fatalf("lost local target %q: %q", value, got)
		}
	}
	for _, value := range []string{"", "https://elsewhere.example/", "https://studio.example/?section=window", "//elsewhere.example/", "/\\elsewhere.example", "/%2felsewhere.example", "/%252felsewhere.example", "/%5cevil.example", "/%255cevil.example", "/\r\nLocation: https://evil.example", "/%0d%0aLocation", "javascript:alert(1)", strings.Repeat("/", 2049)} {
		if got := loginReturnTo([]string{value}); got != "/" {
			t.Fatalf("unsafe target accepted %q: %q", value, got)
		}
	}
	if got := loginReturnTo([]string{"/safe", "//elsewhere.example"}); got != "/" {
		t.Fatal("ambiguous targets accepted")
	}
}
