package managementcatalog

import (
	"encoding/json"
	"github.com/viant/forge/backend/types"
	"testing"
)

func TestComponentEditorTypedWindowAndExactOperationDependencies(t *testing.T) {
	definitions, err := managementWindow(Config{Namespace: "datastudio", ProviderIdentity: "producer", MCPService: "mcp://datastudio/admin"}, "components")
	if err != nil {
		t.Fatal(err)
	}
	var envelope types.WindowResourceEnvelope
	if err = json.Unmarshal(definitions[0].JSON, &envelope); err != nil {
		t.Fatal(err)
	}
	if err = envelope.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, variant := range envelope.Variants {
		if len(variant.Window.Dialogs) != 2 || variant.Window.Actions == nil || len(variant.Window.View.Content.Containers) < 8 {
			t.Fatal("functional editor controls absent")
		}
		for _, id := range []string{"component_versions_apply", "component_versions_validate", "component_preview_execute", "component_publications_publish", "component_resources_upsert_file", "component_versions_builder"} {
			if variant.DataSourceResources[id].Resource.URI != "datasource://datastudio/"+id {
				t.Fatalf("operation dependency %s absent", id)
			}
		}
	}
}
