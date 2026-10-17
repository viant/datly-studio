package host

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly/constant"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

// This SQLite test verifies constant plumbing, not BigQuery dialect parity.
func TestPublishedReaderDeploymentConstantOverridesAuthoredDefault(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := context.Background()
	dql := strings.Replace(unscopedRecordsDQL, "#define", "#define($_ = $SelectedID<int>(const/SelectedID).Value('999'))\n#define", 1)
	dql = strings.Replace(dql, "ORDER BY t.id", "WHERE t.id = $SelectedID ORDER BY t.id", 1)
	if _, err = service.studio.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='records'", dql, dql); err != nil {
		t.Fatal(err)
	}
	service.config.Const, err = constant.New(map[string]string{"SelectedID": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reload(ctx, 0); err != nil {
		t.Fatal(err)
	}
	output, err := service.manager.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/runtime/records", Name: "records"}, Route: spec.RouteRef{Method: "GET", Path: "/records"}}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(output)
	if err != nil || !strings.Contains(string(data), "alpha-101") || strings.Contains(string(data), "beta-102") {
		t.Fatalf("constant-bound output: %s %v", data, err)
	}
	var stored string
	if err = service.studio.QueryRowContext(ctx, "SELECT authored_dql FROM component_versions WHERE report_id='records'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != dql {
		t.Fatal("deployment constants changed authored source")
	}
}
