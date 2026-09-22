package sqltransport

import (
	"reflect"
	"testing"

	"github.com/viant/datly/spec"
)

func TestMCPToolFamilyReservesDerivedCanonicalNames(t *testing.T) {
	enabled := true
	settings := &spec.ReportSettings{Enabled: true, Compose: &spec.CubeComposeSettings{Enabled: true, MCPTool: &enabled}}
	want := []string{"owner.vendor.read", "owner.vendor.readCube", "owner.vendor.readCubeCompose"}
	if actual := mcpToolFamily("owner.vendor.read", settings); !reflect.DeepEqual(actual, want) {
		t.Fatalf("tool family=%v want=%v", actual, want)
	}
	for _, name := range want {
		if !canonicalMCPToolName(name) {
			t.Fatalf("generated tool name %q is not canonical", name)
		}
	}
}
