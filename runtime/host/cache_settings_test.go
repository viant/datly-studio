package host

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

func TestPublishedReaderResolvesDeploymentCache(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := context.Background()
	dql := strings.Replace(unscopedRecordsDQL, "#define", "#setting($_ = $cache('local'))\n#define", 1)
	if _, err = service.studio.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='records'", dql, dql); err != nil {
		t.Fatal(err)
	}
	if err = service.Reload(ctx, 0); err == nil {
		t.Fatal("unconfigured cache was accepted")
	}
	service.config.ReaderCaches = map[string]*spec.CacheSettings{"local": {Name: "local", Enabled: true, Provider: "afs", Location: filepath.Join(t.TempDir(), "cache"), TTL: "1m"}}
	if err = service.Reload(ctx, 0); err != nil {
		t.Fatal(err)
	}
	output, err := service.manager.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/runtime/records", Name: "records"}, Route: spec.RouteRef{Method: "GET", Path: "/records"}}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(output)
	if err != nil || !strings.Contains(string(data), "alpha-101") {
		t.Fatalf("cached reader: %s %v", data, err)
	}
}
