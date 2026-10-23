package managementcatalog

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/viant/forge/backend/types"
)

func TestCompiledWindowBundlesCloseAllManagementDatasources(t *testing.T) {
	config := Config{Namespace: "datastudio", ProviderIdentity: "datastudio-management", MCPService: "mcp://datastudio/admin", PublicURL: "http://127.0.0.1:8095"}
	bundles, err := CompiledWindowBundles(config)
	if err != nil || len(bundles) < 10 {
		t.Fatalf("compile management catalog: count=%d err=%v", len(bundles), err)
	}
	seen := map[string]bool{}
	protocolTools := 0
	for _, bundle := range bundles {
		if seen[bundle.URI] {
			t.Fatalf("duplicate published root %s", bundle.URI)
		}
		seen[bundle.URI] = true
		var envelope types.WindowResourceEnvelope
		if json.Unmarshal(bundle.JSON, &envelope) != nil || envelope.Validate() != nil || envelope.SchemaVersion != 2 || envelope.Format != types.WindowBundleFormat {
			t.Fatalf("invalid compiled management window %s", bundle.URI)
		}
		for _, variant := range envelope.Variants {
			if len(variant.DataSourceResources) != 0 || len(variant.DataSources) == 0 {
				t.Fatalf("window %s retained child refs or lost descriptors", bundle.URI)
			}
			for _, raw := range variant.DataSources {
				var descriptor struct {
					Backend struct {
						Kind    string `json:"kind"`
						Service string `json:"service"`
						Method  string `json:"method"`
					} `json:"backend"`
				}
				if json.Unmarshal(raw, &descriptor) != nil || descriptor.Backend.Kind != "mcp_tool" && descriptor.Backend.Kind != "inline" || descriptor.Backend.Kind == "mcp_tool" && descriptor.Backend.Service != config.ProviderIdentity {
					t.Fatal("published datasource did not bind the configured provider")
				}
				if descriptor.Backend.Method == "studio.sdk.catalog.tools.list" {
					protocolTools++
				}
			}
		}
	}
	if protocolTools != 2 {
		t.Fatalf("runtime protocol tool catalog projections=%d, want 2", protocolTools)
	}
	again, err := CompiledWindowBundles(config)
	if err != nil || len(again) != len(bundles) {
		t.Fatalf("recompile management catalog: %v", err)
	}
	for i := range bundles {
		if bundles[i].URI != again[i].URI || !bytes.Equal(bundles[i].JSON, again[i].JSON) {
			t.Fatal("compiled management output is nondeterministic")
		}
	}
}

func TestCompileWindowBundleRejectsMissingAndDriftedChild(t *testing.T) {
	config := Config{Namespace: "datastudio", ProviderIdentity: "datastudio-management", MCPService: "mcp://datastudio/admin"}
	definitions, err := managementWindow(config, "connectors")
	if err != nil {
		t.Fatal(err)
	}
	byURI := map[string]Definition{}
	for _, definition := range definitions {
		byURI[definition.URI] = definition
	}
	if _, err := CompileWindowBundle(definitions[0], byURI); err != nil {
		t.Fatalf("valid closure denied: %v", err)
	}
	childURI := definitions[1].URI
	child := byURI[childURI]
	delete(byURI, childURI)
	if _, err := CompileWindowBundle(definitions[0], byURI); err == nil {
		t.Fatal("missing declared child accepted")
	}
	child.JSON = json.RawMessage(`{"schemaVersion":1,"dataSource":{"id":"changed"}}`)
	byURI[childURI] = child
	if _, err := CompileWindowBundle(definitions[0], byURI); err == nil {
		t.Fatal("drifted declared child accepted")
	}
}
