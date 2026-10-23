package reader

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/resourceartifact"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

func TestPublicFileReaderExcludesHostOwnedCompiledRows(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "public_file_reader_compiled_exclusion", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "component_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}}},
		datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": "SELECT 1", "generated_dql": "SELECT 1", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-1", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct{ namespace, path, content string }{
		{"assets", "assets/public.txt", "public authored text"},
		{resourceartifact.Namespace, "compiled/private.json", "private compiled text"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO component_resource_files
			(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
			VALUES('r1',1,?, ?, ?, ?, ?, ?, FALSE, CURRENT_TIMESTAMP)`,
			file.path, file.namespace, file.path, []byte(file.content), len(file.content), file.path); err != nil {
			t.Fatal(err)
		}
	}
	resources := resource.New()
	if err := resources.Register(FileDatlyResourceNamespace, FileDatlyResources); err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	field, ok := reflect.TypeFor[FileComponent]().FieldByName("Contract")
	if !ok {
		t.Fatal("file reader contract missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("file reader metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: "FileComponent", FieldName: "Contract", PackageName: "reader", PackagePath: reflect.TypeFor[FileComponent]().PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[Input](), OutputType: reflect.TypeFor[Output](),
		Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{datatest.AuthRegistration(t, db, jwt.Factory, resources), entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	request := httptest.NewRequest(http.MethodGet, "/v1/studio/reports/r1/versions/1/resources/files", nil)
	request.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
	response := httptest.NewRecorder()
	gateway.NewHandler(runtime, nil, "test").ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("public authored text")) || bytes.Contains(response.Body.Bytes(), []byte("private compiled text")) {
		t.Fatalf("public reader status=%d body=%s", response.Code, response.Body.String())
	}
}
