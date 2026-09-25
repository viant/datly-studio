package reader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

func TestPublicationEventReaderFollowsCurrentReportOwner(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "publication_event_owner", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}, {"owner_id": "carol", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_publication_events", Rows: []datatest.Row{{"event_id": "e1", "report_id": "r1", "owner_id": "alice", "operation": "publish", "status": "succeeded", "requested_by": "alice", "occurred_at": "2026-09-17 10:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[PublicationEventComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("publication event component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(PublicationEventDatlyResourceNamespace, PublicationEventDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[Input](), OutputType: reflect.TypeFor[Output](),
		Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err = connector.RegisterConnector("studio", db); err != nil {
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
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	read := func(subject string) []*PublicationEvent {
		request := httptest.NewRequest(http.MethodGet, "/v1/studio/publication-events?reportId=r1", nil)
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, readErr := runtime.ExecuteRoute(ctx, http.MethodGet, "/v1/studio/publication-events", scope)
		if readErr != nil {
			t.Fatal(readErr)
		}
		return value.(*Output).Events
	}
	if actual := read("alice"); len(actual) != 1 || actual[0].EventID == nil || *actual[0].EventID != "e1" {
		t.Fatalf("initial owner events=%+v", actual)
	}
	if actual := read("bob"); len(actual) != 0 {
		t.Fatalf("unrelated subject events=%+v", actual)
	}
	if _, err = db.ExecContext(ctx, `UPDATE reports SET owner_id='carol' WHERE id='r1'`); err != nil {
		t.Fatal(err)
	}
	if actual := read("alice"); len(actual) != 0 {
		t.Fatalf("former owner events=%+v", actual)
	}
	if actual := read("carol"); len(actual) != 1 || actual[0].EventID == nil || *actual[0].EventID != "e1" {
		t.Fatalf("current owner events=%+v", actual)
	}
}
