package runtimeadmin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
)

func TestReloadClassifiesMissingNamespaceWithoutMisclassifyingUnscopedOrigin(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := New(server.URL, "deployment-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	selected := sdk.WithNamespaceSelection(context.Background(), strings.Repeat("a", 64))
	if err := client.Reload(selected, 7); !errors.Is(err, ErrNamespaceUnavailable) {
		t.Fatalf("missing namespace: %v", err)
	}
	if err := client.Reload(context.Background(), 7); err == nil || errors.Is(err, ErrNamespaceUnavailable) {
		t.Fatalf("missing admin origin mislabeled as namespace: %v", err)
	}
}
