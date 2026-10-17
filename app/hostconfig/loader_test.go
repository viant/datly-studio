package hostconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRetainsDatlyConfigurationOrigin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host.yaml")
	input := base + "StaticRoot: ui\nJWTValidator:\n  CertURL: https://auth.example.com/certs\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if c.StaticRoot != filepath.Join(dir, "ui") {
		t.Fatalf("static assets were not resolved against the application base: %q", c.StaticRoot)
	}
	if c.URL != path {
		t.Fatalf("origin changed: %q", c.URL)
	}
	if c.NamespaceMCP.Address != "127.0.0.1:8082" || c.Connector != "studio" || c.Connectors[0].Secret.Key != "blowfish://default" {
		t.Fatal("shared/native configuration changed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("loader wrote generated configs: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Load(ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestLoadUsesDatlyConnectorDependencyFile(t *testing.T) {
	dir := t.TempDir()
	connector := `Connectors:
  - Name: studio
    Driver: mysql
    DSN: '${Username}:${Password}@tcp(localhost:3306)/studio'
    Secret:
      URL: file:///config/database.enc
      Key: blowfish://default
`
	if err := os.WriteFile(filepath.Join(dir, "connector.yaml"), []byte(connector), 0600); err != nil {
		t.Fatal(err)
	}
	input := `Endpoint:
  Address: 127.0.0.1:8080
MCP:
  Address: 127.0.0.1:8081
NamespaceMCP:
  Address: 127.0.0.1:8082
Connector: studio
DependencyURL: connector.yaml
`
	path := filepath.Join(dir, "host.yaml")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Connectors) != 1 || c.Connectors[0].Name != "studio" || c.Connectors[0].Secret.URL != "file:///config/database.enc" {
		t.Fatal("Datly connector file was not loaded")
	}
	if c.BaseDir != dir {
		t.Fatalf("relative dependency origin changed: %q", c.BaseDir)
	}
}
