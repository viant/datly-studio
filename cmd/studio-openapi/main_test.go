package main

import (
	"bytes"
	"context"
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
}
