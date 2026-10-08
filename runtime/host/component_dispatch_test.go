package host

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/forge/backend/mcp/portable"
	"github.com/viant/forge/backend/reporting/identity"
	xhandler "github.com/viant/xdatly/handler"
)

func TestComponentDispatchRunsAuthorizedOlderVersionAndRejectsDrift(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{}))
	documents := []authz.Document{}
	for _, version := range []string{"1", "2"} {
		documents = append(documents, authz.Document{Resource: authz.Resource{Kind: "component", ID: "records", Version: version, Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}})
	}
	store, err := authz.NewStaticStore(documents)
	if err != nil {
		t.Fatal(err)
	}
	service.resourceAccess.Store = store

	original, err := service.exactDefinition(ctx, "records", 1)
	if err != nil {
		t.Fatal(err)
	}
	older := strings.Replace(original.dql, "t.name", "'old-version' AS name", 1)
	active := strings.Replace(original.dql, "t.name", "'active-version' AS name", 1)
	if _, err := service.studio.ExecContext(ctx, "UPDATE report_versions SET authored_dql=?,generated_dql=? WHERE report_id='records' AND version_no=1", older, older); err != nil {
		t.Fatal(err)
	}
	_, err = service.studio.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
   SELECT report_id,2,'published',authoring_mode,?, ?,component_spec_json,spec_format_version,'hash-v2',type_manifest_json,compile_status,datly_version,compiler_version,2,created_by,created_at FROM report_versions WHERE report_id='records' AND version_no=1`, active, active)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.studio.ExecContext(ctx, "UPDATE report_publications SET active_version_no=2 WHERE report_id='records'"); err != nil {
		t.Fatal(err)
	}
	if err = service.Reload(ctx, 0); err != nil {
		t.Fatal(err)
	}
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	pin, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewForgeComponentDispatcher(service, []ComponentDispatchMapping{
		{Service: "records-api", Method: "fetch", Component: ref},
		{Service: "records-api", Method: "fetch", Component: ComponentReference{Kind: "dynamic", ID: "records", Revision: "2", Method: http.MethodGet, Route: "/records"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !dispatcher.IsComponentProducer("records-api") || dispatcher.IsComponentProducer("other-api") || dispatcher.IsComponentProducer(" records-api") {
		t.Fatal("producer classification must match only canonical configured service names")
	}
	observed, err := dispatcher.ObserveComponent(ctx, "records-api", "fetch", pin)
	if err != nil || observed != pin {
		t.Fatalf("exact historical component observation=%+v err=%v", observed, err)
	}
	result, err := dispatcher.ExecuteComponent(ctx, "records-api", "fetch", pin, nil)
	if err != nil || !strings.Contains(string(result), "old-version") || strings.Contains(string(result), "active-version") {
		t.Fatalf("older component changed dispatch: %s %v", result, err)
	}
	for label, altered := range map[string]portable.ComponentBinding{
		"active revision":           {Kind: pin.Kind, ID: pin.ID, Revision: "active", ContentFingerprint: pin.ContentFingerprint, SchemaFingerprint: pin.SchemaFingerprint},
		"foreign identity":          {Kind: pin.Kind, ID: "other", Revision: pin.Revision, ContentFingerprint: pin.ContentFingerprint, SchemaFingerprint: pin.SchemaFingerprint},
		"unavailable revision":      {Kind: pin.Kind, ID: pin.ID, Revision: "3", ContentFingerprint: pin.ContentFingerprint, SchemaFingerprint: pin.SchemaFingerprint},
		"content fingerprint drift": {Kind: pin.Kind, ID: pin.ID, Revision: pin.Revision, ContentFingerprint: identity.ContentFingerprint([]byte("other-content")), SchemaFingerprint: pin.SchemaFingerprint},
		"schema fingerprint drift":  {Kind: pin.Kind, ID: pin.ID, Revision: pin.Revision, ContentFingerprint: pin.ContentFingerprint, SchemaFingerprint: identity.ContentFingerprint([]byte("other-schema"))},
	} {
		if output, err := dispatcher.ExecuteComponent(ctx, "records-api", "fetch", altered, nil); err == nil || len(output) != 0 {
			t.Fatalf("%s dispatch was accepted: %s %v", label, output, err)
		}
		if _, err := dispatcher.ObserveComponent(ctx, "records-api", "fetch", altered); err == nil {
			t.Fatalf("%s observation was accepted", label)
		}
	}
	ref2 := ref
	ref2.Revision = "2"
	pin2, err := service.ResolveComponentBinding(ctx, ref2)
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.ExecuteComponentJSON(ctx, ref2, pin2, nil)
	if err != nil || !strings.Contains(string(result), "active-version") {
		t.Fatalf("exact new version did not run: %s %v", result, err)
	}
	if _, err := service.ExecutePublishedJSON(ctx, http.MethodGet, "/records", nil); err == nil {
		t.Fatal("unbound active route remained available")
	}
	for _, revision := range []string{"", "working", "active", "latest", "0", "01", "-1", "999"} {
		invalid := ref
		invalid.Revision = revision
		if _, err := service.ResolveComponentBinding(ctx, invalid); err == nil {
			t.Fatalf("invalid component revision accepted: %q", revision)
		}
	}
	if _, err := service.studio.ExecContext(ctx, "UPDATE report_versions SET generated_dql=? WHERE report_id='records' AND version_no=1", active); err != nil {
		t.Fatal(err)
	}
	if result, err := dispatcher.ExecuteComponent(ctx, "records-api", "fetch", pin, nil); err == nil || len(result) != 0 {
		t.Fatalf("changed pinned component leaked output: %s %v", result, err)
	}
}

type linkedInput struct {
	Name string `parameter:"Name,kind=query,in=name,dataType=string,required=false"`
}
type linkedOutput struct {
	Message string `json:"message"`
}
type linkedHandler struct{}

func (linkedHandler) Exec(_ context.Context, _ xhandler.Session, input *linkedInput, output *linkedOutput) error {
	output.Message = "linked-v1:" + input.Name
	return nil
}
func linkedBuild(context.Context, *typecatalog.Catalog) (*application.Build, error) {
	component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/linked", Name: "Greeting"}, Routes: []*spec.Route{{Method: http.MethodGet, Path: "/linked"}}}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[linkedInput](), OutputType: reflect.TypeFor[linkedOutput]()})
	if err != nil {
		return nil, err
	}
	registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[linkedInput, linkedOutput](linkedHandler{})})
	if err != nil {
		return nil, err
	}
	return &application.Build{Components: []*registry.RegisteredComponent{registered}, Version: "binary-v1"}, nil
}

type linkedFacts struct {
	deadline time.Time
	roles    []string
}

func (p *linkedFacts) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "person", Issuer: "issuer", Tenant: "team", Roles: p.roles, ValidUntil: p.deadline}, nil
}
func TestComponentDispatchExecutesNativeLinkedArtifactAndRechecksIdentity(t *testing.T) {
	testDynamicHost(t, false, false, dynamicHostExtension{verify: func(service *Service, _ string) {
		previousConfig, previousAccess := service.config, service.resourceAccess
		defer func() { service.config, service.resourceAccess = previousConfig, previousAccess }()
		ctx := context.Background()
		facts := &linkedFacts{deadline: time.Now().Add(time.Minute), roles: []string{"reader"}}
		resource := authz.Resource{Kind: "component", ID: "greeting", Version: "build-v1", Tenant: "team"}
		store, err := authz.NewStaticStore([]authz.Document{{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
		if err != nil {
			t.Fatal(err)
		}
		service.config.Access = &ResourceAccessConfig{Tenant: "team", Provider: facts}
		service.resourceAccess = &authz.Service{Provider: facts, Store: store}
		service.config.LinkedComponents = []LinkedComponentSource{{ID: "greeting", Revision: "build-v1", ArtifactFingerprint: identity.ContentFingerprint([]byte("binary-v1")), Build: linkedBuild}}
		ref := ComponentReference{Kind: "linked", ID: "greeting", Revision: "build-v1", Method: http.MethodGet, Route: "/linked"}
		pin, err := service.ResolveComponentBinding(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		dispatcher, err := NewForgeComponentDispatcher(service, []ComponentDispatchMapping{{Service: "linked-api", Method: "greet", Component: ref}})
		if err != nil {
			t.Fatal(err)
		}
		observed, err := dispatcher.ObserveComponent(ctx, "linked-api", "greet", pin)
		if err != nil || observed != pin {
			t.Fatalf("linked observation=%+v err=%v", observed, err)
		}
		output, err := dispatcher.ExecuteComponent(ctx, "linked-api", "greet", pin, map[string]interface{}{"name": "Ada"})
		if err != nil || !json.Valid(output) || !strings.Contains(string(output), "linked-v1:Ada") {
			t.Fatalf("linked runtime output=%s err=%v", output, err)
		}
		service.config.LinkedComponents[0].ArtifactFingerprint = identity.ContentFingerprint([]byte("binary-v2"))
		if output, err := dispatcher.ExecuteComponent(ctx, "linked-api", "greet", pin, nil); err == nil || len(output) > 0 {
			t.Fatalf("linked artifact drift accepted: %s %v", output, err)
		}
		service.config.LinkedComponents[0].ArtifactFingerprint = identity.ContentFingerprint([]byte("binary-v1"))
		facts.roles = nil
		if output, err := service.ExecuteComponentJSON(ctx, ref, pin, nil); err == nil || len(output) > 0 {
			t.Fatalf("revoked linked authority leaked output: %s %v", output, err)
		}
	}})
}

func TestComponentDispatchRejectsChangedVersionedSQLResource(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{}))
	resource := authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}
	store, err := authz.NewStaticStore([]authz.Document{{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	service.resourceAccess.Store = store
	query := "SELECT t.id, t.name FROM tasks t ORDER BY t.id"
	source := strings.Replace(unscopedRecordsDQL, query, "${embed:sql/records.sql}", 1)
	if source == unscopedRecordsDQL {
		t.Fatal("SQL resource fixture did not replace the source")
	}
	if _, err := service.studio.ExecContext(ctx, "UPDATE report_versions SET authored_dql=?,generated_dql=? WHERE report_id='records' AND version_no=1", source, source); err != nil {
		t.Fatal(err)
	}
	if _, err := service.studio.ExecContext(ctx, `INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES('records',1,?,'records','sql/records.sql',?,?,?,FALSE,CURRENT_TIMESTAMP)`, identity.ContentFingerprint([]byte("records-resource")), []byte(query), len(query), identity.ContentFingerprint([]byte(query))); err != nil {
		t.Fatal(err)
	}
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	pin, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	output, err := service.ExecuteComponentJSON(ctx, ref, pin, nil)
	if err != nil || !strings.Contains(string(output), "alpha-101") {
		t.Fatalf("native SQL resource not executed: %s %v", output, err)
	}
	changed := "SELECT t.id, 'changed-resource' AS name FROM tasks t ORDER BY t.id"
	if _, err := service.studio.ExecContext(ctx, "UPDATE report_resource_files SET content=?,content_size=?,content_sha256=? WHERE report_id='records' AND version_no=1", []byte(changed), len(changed), identity.ContentFingerprint([]byte(changed))); err != nil {
		t.Fatal(err)
	}
	if output, err := service.ExecuteComponentJSON(ctx, ref, pin, nil); err == nil || len(output) != 0 {
		t.Fatalf("changed SQL resource released data: %s %v", output, err)
	}
}

func TestComponentDispatchDoesNotRequireDeploymentSource(t *testing.T) {
	testDynamicHost(t, false, false, dynamicHostExtension{
		configure: func(config *Config) { config.ModulePath = "example.com/runtime" },
		verify: func(service *Service, _ string) {
			if err := os.Remove(filepath.Join(service.config.RootDir, "go.mod")); err != nil {
				t.Fatal(err)
			}
			ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
			binding, err := service.ResolveComponentBinding(context.Background(), ref)
			if err != nil {
				t.Fatal(err)
			}
			output, err := service.ExecuteComponentJSON(context.Background(), ref, binding, nil)
			if err != nil || !strings.Contains(string(output), "ready") || !strings.Contains(string(output), "primary") {
				t.Fatalf("source-free dispatch failed: %s %v", output, err)
			}
		},
	})
}
