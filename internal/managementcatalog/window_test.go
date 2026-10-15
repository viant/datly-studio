package managementcatalog

import (
	"bytes"
	"encoding/json"
	"testing"

	ds "github.com/viant/forge/backend/datasource/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
)

func TestNamespaceWindowPortableContract(t *testing.T) {
	for _, namespace := range []string{"dstudio", "operations"} {
		t.Run(namespace, func(t *testing.T) {
			config := Config{Namespace: namespace, ProviderIdentity: "studio-provider", MCPService: "configured-management"}
			definitions, err := NamespaceWindow(config)
			if err != nil {
				t.Fatal(err)
			}
			if len(definitions) != 2 || definitions[0].URI != "window://"+namespace+"/namespaces" || definitions[1].URI != "datasource://"+namespace+"/namespaces" {
				t.Fatalf("unexpected resource identities: %+v", definitions)
			}
			var envelope types.WindowResourceEnvelope
			for _, definition := range definitions {
				var transport map[string]any
				if err := json.Unmarshal(definition.JSON, &transport); err != nil {
					t.Fatal(err)
				}
				roundtrip, err := json.Marshal(transport)
				if err != nil || !bytes.Equal(roundtrip, definition.JSON) {
					t.Fatal("MCP object roundtrip changed resource content")
				}
			}
			if err := json.Unmarshal(definitions[0].JSON, &envelope); err != nil {
				t.Fatal(err)
			}
			if err := envelope.Validate(); err != nil {
				t.Fatal(err)
			}
			var datasource struct {
				SchemaVersion int           `json:"schemaVersion"`
				DataSource    ds.DataSource `json:"dataSource"`
			}
			if err := json.Unmarshal(definitions[1].JSON, &datasource); err != nil {
				t.Fatal(err)
			}
			value := datasource.DataSource
			if datasource.SchemaVersion != 1 || value.ID != "namespaces" || value.Backend.Service != config.MCPService || value.Backend.Kind != ds.BackendMCPTool || value.Backend.Method != "studio.sdk.namespaces.list" || value.Selectors.Data != "items" {
				t.Fatalf("incorrect management datasource: %+v", value)
			}
			if value.Cache == nil || value.Cache.Enabled == nil || *value.Cache.Enabled || value.Service != nil {
				t.Fatal("management datasource must use the MCP backend without caching or a host-specific HTTP route")
			}
			if value.Paging == nil || value.Paging.Parameters.Page != "offset" || value.Paging.Parameters.Size != "limit" || !value.Paging.OpenEnded {
				t.Fatal("management datasource must preserve the list tool pagination contract")
			}
			for _, variant := range envelope.Variants {
				ref := variant.DataSourceResources["namespaces"]
				if ref.Resource.URI != definitions[1].URI || ref.ProviderIdentity != config.ProviderIdentity || ref.ContentFingerprint != identity.ContentFingerprint(definitions[1].JSON) {
					t.Fatal("window does not bind the exact reusable datasource")
				}
				if variant.Window.Namespace != namespace || variant.Window.View.Content.DataSourceRef != "namespaces" || len(variant.Window.View.Content.Table.Columns) != 5 {
					t.Fatal("window lost its namespace or table definition")
				}
			}
			again, err := NamespaceWindow(config)
			if err != nil || !bytes.Equal(again[0].JSON, definitions[0].JSON) || !bytes.Equal(again[1].JSON, definitions[1].JSON) {
				t.Fatal("resource output is nondeterministic")
			}
		})
	}
}

func TestNamespaceWindowRejectsInvalidIdentity(t *testing.T) {
	for _, config := range []Config{
		{},
		{Namespace: "dstudio", MCPService: "management"},
		{Namespace: "dstudio", ProviderIdentity: "provider"},
		{Namespace: "dstudio/foreign", ProviderIdentity: "provider", MCPService: "management"},
	} {
		if _, err := NamespaceWindow(config); err == nil {
			t.Fatalf("invalid configuration accepted: %+v", config)
		}
	}
}
