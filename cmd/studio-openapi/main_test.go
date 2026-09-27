package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedSDKOpenAPIIsCurrent(t *testing.T) {
	generated := filepath.Join(t.TempDir(), "studio.json")
	if err := run(context.Background(), generated); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "sdk", "openapi", "studio.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("Studio SDK OpenAPI is stale; run: GOWORK=off go run ./cmd/studio-openapi")
	}
	var document struct {
		Paths map[string]struct {
			Post struct {
				Responses map[string]json.RawMessage `json:"responses"`
			} `json:"post"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(got, &document); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/studio/sdk/acl.delete", "/v1/studio/sdk/authorization_predicates.delete", "/v1/studio/sdk/connectors.delete", "/v1/studio/sdk/namespaces.delete"} {
		responses := document.Paths[path].Post.Responses
		if len(responses) != 1 || len(responses["204"]) == 0 {
			t.Fatalf("%s OpenAPI responses=%v, want only 204", path, responses)
		}
	}
}
