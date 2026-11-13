package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	x "github.com/viant/x"
)

type PreviewFilterFixture struct {
	Query string `json:"searchTerm"`
	IDs   []int  `json:"recordIds"`
}

func TestInspectionNativePreviewInput(t *testing.T) {
	f := newImportFixture(t)
	const scope = "example.com/preview/records"
	if _, err := f.db.ExecContext(f.ctx, "UPDATE connectors SET dsn_template=':memory:'; UPDATE components SET component_scope=?,component_name='reader' WHERE id=?", scope, f.report.ID); err != nil {
		t.Fatal(err)
	}
	types := typecatalog.NewCatalog()
	if err := types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeFor[PreviewFilterFixture]())); err != nil {
		t.Fatal(err)
	}
	filterType := reflect.TypeFor[PreviewFilterFixture]()
	source := fmt.Sprintf(`#package('%s')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#define($_ = $PageSize<int>(query/page_size).Optional().Value('25'))
#define($_ = $Enabled<bool>(query/enabled).Optional())
#define($_ = $IDs<[]int>(query/ids).Optional())
#define($_ = $Filter<*%s.%s>(body/filter).Optional())
#define($_ = $Token<string>(header/Authorization).Optional())
#define($_ = $Tenant<int>(const/Tenant).Value('7'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT records.*,type(records,'Row') FROM (SELECT 1 AS ID) records`, scope, filterType.PkgPath(), filterType.Name())
	f.transport.RuntimeTypes = types
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.db.ExecContext(f.ctx, "INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,can_use_dql) VALUES(?,'user','runner',1,1,0),(?,'user','viewer',1,0,0)", f.report.ID, f.report.ID); err != nil {
		t.Fatal(err)
	}
	engine := preview.Dynamic{StudioDB: f.db, ModulePath: "example.com/preview", Types: types, Dependencies: func(context.Context, *spec.Component) (*preview.DependencyRuntime, error) {
		t.Fatal("metadata invoked runtime dependencies")
		return nil, nil
	}}
	f.transport.Preview = engine
	runner := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "runner"})
	inspection, err := f.client.Versions().Inspect(runner, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.PreviewInput == nil {
		t.Fatalf("missing native schema: %+v", inspection.Diagnostics)
	}
	input := inspection.PreviewInput
	if input.ReportID != f.report.ID || input.VersionNo != version.VersionNo || input.SourceRevision != version.SourceRevision || input.Method != "GET" || input.Path != "/records" {
		t.Fatalf("wrong contract identity: %+v", input)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err = json.Unmarshal(input.Schema, &schema); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pageSize", "enabled", "idS", "filter"} {
		if schema.Properties[name] == nil {
			t.Errorf("missing public field %s: %s", name, input.Schema)
		}
	}
	for _, name := range []string{"page_size", "Token", "Authorization", "Tenant", "Rows", "token", "authorization", "tenant", "rows"} {
		if schema.Properties[name] != nil {
			t.Errorf("private or noncanonical field %s", name)
		}
	}
	if !strings.Contains(string(schema.Properties["filter"]), "searchTerm") || !strings.Contains(string(schema.Properties["filter"]), "recordIds") {
		t.Fatalf("nested wire names missing: %s", input.Schema)
	}
	if len(schema.Required) != 0 {
		t.Fatalf("optional arguments changed: %s", input.Schema)
	}
	raw, _ := json.Marshal(inspection)
	if inspection.DQL != "" || strings.Contains(string(raw), "SELECT 1 AS ID") || inspection.Capabilities.CanUseDQL {
		t.Fatalf("DQL redaction bypassed: %s", raw)
	}
	viewer := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "viewer"})
	viewed, err := f.client.Versions().Inspect(viewer, f.report.ID, version.VersionNo)
	if err != nil || viewed.PreviewInput != nil {
		t.Fatalf("view-only metadata: %+v %v", viewed, err)
	}
	t.Run("exact run permission checked", func(t *testing.T) {
		previous := f.transport.Authorizer
		defer func() { f.transport.Authorizer = previous }()
		calls := 0
		f.transport.Authorizer = AuthorizerFunc(func(ctx context.Context, request AuthorizationRequest) error {
			if request.Permission == "run" {
				calls++
				return errors.New("denied")
			}
			return nil
		})
		denied, err := f.client.Versions().Inspect(runner, f.report.ID, version.VersionNo)
		if err != nil || calls != 1 || denied.PreviewInput != nil {
			t.Fatalf("exact preview authorization bypassed: %v %d %+v", err, calls, denied)
		}
	})
	t.Run("empty contract is available", func(t *testing.T) {
		empty := fmt.Sprintf("#package('%s')\n#setting($_ = $connector('source'))\n#setting($_ = $route('/empty','GET'))\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT records.*,type(records,'Row') FROM (SELECT 1 AS ID) records", scope)
		v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: empty})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.client.Versions().Inspect(runner, f.report.ID, v.VersionNo)
		if err != nil || got.PreviewInput == nil {
			t.Fatalf("empty schema became unavailable: %v %+v", err, got)
		}
		var shape map[string]interface{}
		if err = json.Unmarshal(got.PreviewInput.Schema, &shape); err != nil || shape["type"] != "object" {
			t.Fatalf("invalid empty contract %s", got.PreviewInput.Schema)
		}
	})
	t.Run("compilation failure has safe diagnostic", func(t *testing.T) {
		v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT PRIVATE_SOURCE_CANARY FROM MISSING_TABLE_CANARY"})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.client.Versions().Inspect(runner, f.report.ID, v.VersionNo)
		if err != nil {
			t.Fatal(err)
		}
		if got.PreviewInput != nil {
			t.Fatal("failed compilation returned a schema")
		}
		found := false
		for _, d := range got.Diagnostics {
			found = found || d.Code == "preview_input_unavailable"
		}
		raw, _ := json.Marshal(got)
		if !found || strings.Contains(string(raw), "CANARY") {
			t.Fatalf("missing safe unavailable diagnostic: %s", raw)
		}
	})
}
