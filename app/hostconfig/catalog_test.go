package hostconfig

import (
	"context"
	"encoding/json"
	"github.com/viant/scy"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSingleCatalogUsesDatlyConnection(t *testing.T) {
	c, err := Decode([]byte(base))
	if err != nil {
		t.Fatal(err)
	}
	c.Store.Name = ""
	c.Store.Driver = "sqlite"
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "synthetic-secret.json")
	secret, err := json.Marshal(map[string]string{"Username": filepath.Join(dir, "catalog.db")})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, secret, 0600); err != nil {
		t.Fatal(err)
	}
	c.Store.DSN = "${Username}"
	c.Store.Secret = &scy.Resource{URL: secretPath}
	selected, err := c.CatalogConnector()
	if err != nil || selected.Name != "studio" {
		t.Fatalf("default connector: %v", err)
	}
	if c.Store.Name != "" {
		t.Fatal("selection mutated config")
	}
	set, err := c.OpenCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer set.Close()
	if _, err := os.Stat(filepath.Join(dir, "catalog.db")); err != nil {
		t.Fatalf("SCY DSN was not expanded: %v", err)
	}
	db, err := set.ResolveDB(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	named, err := set.ResolveDB(context.Background(), "studio")
	if err != nil || named != db {
		t.Fatal("default catalog does not share named handle")
	}
	if _, err = db.Exec("CREATE TABLE synthetic_probe(id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err = set.Close(); err != nil {
		t.Fatal(err)
	}
	if err = db.Ping(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("catalog ownership: %v", err)
	}
}

func TestSingleCatalogRejectsAmbiguityWithoutExposingDSN(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Store = nil },
		func(c *Config) { c.Store.Driver = "" },
		func(c *Config) { c.Store.DSN = "" },
		func(c *Config) { c.Store.AliasOf = "missing" },
	} {
		c, err := Decode([]byte(base))
		if err != nil {
			t.Fatal(err)
		}
		mutate(c)
		if _, err := c.CatalogConnector(); err == nil {
			t.Fatal("ambiguous catalog accepted")
		} else if strings.Contains(err.Error(), "Password") {
			t.Fatal("DSN leaked")
		}
	}
}
