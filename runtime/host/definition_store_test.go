package host

import (
	"strings"
	"testing"
)

func TestMapPublishedDefinitionUsesGeneratedDQLThenAuthoredFallback(t *testing.T) {
	dsn := "file:studio.db"
	for _, test := range []struct {
		name, generated, authored, want string
	}{
		{name: "generated", generated: "SELECT 2", authored: "SELECT 1", want: "SELECT 2"},
		{name: "authored", generated: "   ", authored: "SELECT 1", want: "SELECT 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := mapPublishedDefinition("report", 3, "example.com/report", "report", "studio", "sqlite", &dsn, "secret", test.generated, test.authored)
			if err != nil {
				t.Fatal(err)
			}
			if actual.dql != test.want || actual.versionNo != 3 || actual.dsn != dsn || actual.secretRef != "secret" {
				t.Fatalf("mapped definition=%+v", actual)
			}
		})
	}
}

func TestMapPublishedDefinitionFailsClosed(t *testing.T) {
	dsn := "file:studio.db"
	if _, err := mapPublishedDefinition("report", 1, "scope", "name", "studio", "sqlite", &dsn, "", " ", ""); err == nil || !strings.Contains(err.Error(), "no DQL") {
		t.Fatalf("missing DQL: %v", err)
	}
	if _, err := mapPublishedDefinition("report", 1, "scope", "name", "studio", "sqlite", nil, "", "SELECT 1", ""); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("missing connector DSN: %v", err)
	}
}
