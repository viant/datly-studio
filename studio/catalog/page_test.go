package catalog

import (
	"strings"
	"testing"
)

func TestNamespaceIdentityAndCollectionPaths(t *testing.T) {
	id := strings.Repeat("a", 64)
	if got, err := ParseNamespaceURI("ns://" + id); err != nil || got != id {
		t.Fatalf("namespace URI: %s %v", got, err)
	}
	for _, opaque := range []string{"dstudio", "operations", strings.ToUpper(id)} {
		if got, err := ParseNamespaceURI("ns://" + opaque); err != nil || got != opaque {
			t.Fatalf("opaque namespace identity rejected: %s %v", got, err)
		}
	}
	for _, value := range []string{"ns://" + id + "/", "ns://" + id + "?x=1", "ns://user@" + id, "ns://"} {
		if _, err := ParseNamespaceURI(value); err == nil {
			t.Fatalf("invalid namespace URI accepted: %s", value)
		}
	}
	for _, path := range []string{"/namespaces/" + id, "/namespaces/" + id + "/skill", "/namespaces/" + id + "/mcp"} {
		if got, _, err := ParseNamespacePath(path); err != nil || got != id {
			t.Fatalf("collection path: %s %v", path, err)
		}
	}
	for _, path := range []string{"/namespaces/" + id + "/", "/namespaces/" + id + "/../mcp", "/namespaces/" + id + "/%73kill", "/namespace/" + id} {
		if _, _, err := ParseNamespacePath(path); err == nil {
			t.Fatalf("invalid collection path: %s", path)
		}
	}
}
