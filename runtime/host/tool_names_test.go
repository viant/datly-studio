package host

import (
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"testing"
)

func TestToolPrefixPhrasesDoNotChangeOwnershipOrSource(t *testing.T) {
	source := &registry.RegisteredComponent{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example/awitasviantdevtest/component", Name: "reader"}, Routes: []*spec.Route{{Method: "GET", Path: "/reader", MCP: []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "awitasviantdevtest.ssp.read"}, {Kind: spec.MCPExposureTool, Name: "awitasviantdevtest.ssp.readCube"}}}}}}
	settings := MCPToolNames{RemoveSubjectPhrases: []string{"viant", "devtest", "_"}}
	result, err := settings.registrations([]*registry.RegisteredComponent{source})
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Component.Key != source.Component.Key || result[0].Component.Routes[0].Path != "/reader" {
		t.Fatal("tool shortening changed component identity or routing")
	}
	if got := result[0].Component.Routes[0].MCP[0].Name; got != "awitas.ssp.read" {
		t.Fatal(got)
	}
	if got := result[0].Component.Routes[0].MCP[1].Name; got != "awitas.ssp.readCube" {
		t.Fatal(got)
	}
	if got := source.Component.Routes[0].MCP[0].Name; got != "awitasviantdevtest.ssp.read" {
		t.Fatal("mutated authored source")
	}
	if got, err := settings.toolName("awitas_viant_devtest.ssp.read"); err != nil || got != "awitas.ssp.read" {
		t.Fatalf("underscore removal: %q %v", got, err)
	}
}
func TestShortenedToolPrefixCollisionsFailClosed(t *testing.T) {
	makeComponent := func(name string) *registry.RegisteredComponent {
		return &registry.RegisteredComponent{Component: &spec.Component{Routes: []*spec.Route{{MCP: []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: name}}}}}}
	}
	settings := MCPToolNames{RemoveSubjectPhrases: []string{"viant"}}
	if _, err := settings.registrations([]*registry.RegisteredComponent{makeComponent("awitasviant.reader"), makeComponent("awitas.reader")}); err == nil {
		t.Fatal("normalized collision allowed")
	}
	if _, err := settings.toolName("viant.reader"); err == nil {
		t.Fatal("empty owner prefix allowed")
	}
	if got, err := (MCPToolNames{}).toolName("awitasviant.reader"); err != nil || got != "awitasviant.reader" {
		t.Fatal("default configuration changed tool names")
	}
}
