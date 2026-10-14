package sdk_test

import (
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/sdk"
	"testing"
)

func TestNamespaceIDMatchesStorageOwnership(t *testing.T) {
	for _, pair := range [][2]string{{"owner", "ui-first"}, {"other", "ui-first"}, {"owner", "other"}} {
		if got := sdk.NamespaceID(pair[0], pair[1]); got != namespaceaccess.ID(pair[0], pair[1]) {
			t.Fatalf("identity mismatch: %s", got)
		}
	}
	if sdk.NamespaceID("owner", "ui-first") == sdk.NamespaceID("other", "ui-first") {
		t.Fatal("owners share namespace identity")
	}
}
