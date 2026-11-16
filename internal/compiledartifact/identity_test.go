package compiledartifact

import (
	"strings"
	"testing"

	"github.com/viant/datly/typecatalog"
)

func TestIdentitySeparatesPreviewFromPublicationAndResolvedDestinations(t *testing.T) {
	source := Source{ReportID: "reader", VersionNo: 1, DQL: "SELECT * FROM records", Scope: "example.com/proof/records", Name: "reader", Connector: "main", Driver: "sqlite"}
	connectors := map[string]Connector{"main": {Driver: "sqlite", DSN: "file:owned-first.db?password=credential-canary"}}
	published, _, err := Identity("namespace", "example.com/proof", ".", source, "resources", "constants", connectors, typecatalog.NewCatalog())
	if err != nil {
		t.Fatal(err)
	}
	source.Preview = true
	preview, _, err := Identity("namespace", "example.com/proof", ".", source, "resources", "constants", connectors, typecatalog.NewCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if published.Key == preview.Key || published.Path == preview.Path {
		t.Fatal("preview can overwrite published snapshot")
	}
	connectors["main"] = Connector{Driver: "sqlite", DSN: "file:owned-second.db?password=credential-canary"}
	moved, _, err := Identity("namespace", "example.com/proof", ".", source, "resources", "constants", connectors, typecatalog.NewCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if preview.Key == moved.Key {
		t.Fatal("changed resolved destination retained artifact key")
	}
	if strings.Contains(preview.URI, "credential-canary") || strings.Contains(moved.URI, "credential-canary") {
		t.Fatal("identity disclosed connection material")
	}
}
