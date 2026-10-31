package channels_test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/studio/artifact_release/channels"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
	"github.com/viant/x"
)

// STUDIO_RELEASE_MYSQL_DSN must name an empty, disposable MySQL test database.
func TestNativeChannelsMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_RELEASE_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STUDIO_RELEASE_MYSQL_DSN to a disposable MySQL database")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = schema.Initialize(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	for _, scope := range []string{"selected", "other"} {
		if _, err = db.ExecContext(ctx, `INSERT INTO artifact_release_channels(scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,revision,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,'description','[]',0,?,'actor',?,'actor')`, scope, "tenant", "account", "workspace", "namespace", "qa", now, now); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM artifact_release_channels WHERE scope_id IN ('selected','other')")
	})
	registry := x.NewRegistry()
	for _, typ := range []reflect.Type{reflect.TypeFor[channels.Input](), reflect.TypeFor[channels.InputHas](), reflect.TypeFor[channels.Output](), reflect.TypeFor[channels.ReleaseChannel](), reflect.TypeFor[channels.ChannelsComponent]()} {
		registry.Register(x.NewType(typ))
	}
	pkg := reflect.TypeFor[channels.ChannelsComponent]().PkgPath()
	server, err := standalone.New(ctx, standalone.Options{Config: &config.Config{Connector: "studio", GoBootstrap: &config.Packages{Packages: []string{pkg}, LinkedOnly: true}, Connectors: []connector.Config{{Name: "studio", Driver: "mysql", DSN: dsn}}}, Registry: registry, Holders: []any{channels.ChannelsComponent{}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown(ctx) })
	if err = server.Reload(ctx, 1); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"selected", "absent"} {
		input := &channels.Input{}
		input.SetScopeId(scope)
		value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "channels"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/artifact-release-store/channels"}}, Input: input})
		if err != nil {
			t.Fatal(err)
		}
		output := value.(*channels.Output)
		expected := 0
		if scope == "selected" {
			expected = 1
		}
		if len(output.Data) != expected {
			t.Fatalf("scope %q: rows=%d want=%d", scope, len(output.Data), expected)
		}
		if expected == 1 && (output.Data[0].ScopeId == nil || *output.Data[0].ScopeId != scope) {
			t.Fatalf("scope predicate leaked foreign row: %+v", output.Data[0])
		}
	}
}
