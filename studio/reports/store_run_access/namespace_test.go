package store_run_access_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz/oauth"
	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/accessconfig"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/internal/versionrun"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	. "github.com/viant/datly-studio/studio/reports/store_run_access"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
)

func TestNativeRunAccessRetainsOwnerWithinSelectedNamespace(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "run_namespace", "studio")
	jwt := datatest.NewJWTFixture(t)
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, jwt.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "run-test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", keyPath)
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	now := "2026-09-29 00:00:00"
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"namespace_id": namespaceaccess.ID("alice", "alpha"), "owner_id": "alice", "name": "alpha", "title": "Alpha", "status": "active", "created_at": now, "updated_at": now}, {"namespace_id": namespaceaccess.ID("alice", "beta"), "owner_id": "alice", "name": "beta", "title": "Beta", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "report", "namespace_id": namespaceaccess.ID("alice", "alpha"), "namespace": "alpha", "slug": "report", "title": "Report", "owner_id": "alice", "status": "active", "default_connector_name": "main", "component_scope": "reader/report", "component_name": "report", "created_at": now, "updated_at": now}}},
	); err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err := resources.Register(AccessDatlyResourceNamespace, AccessDatlyResources); err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	entry, target, err := readercomponent.Compile(reflect.TypeFor[AccessComponent](), "store_run_access", reflect.TypeFor[Input](), reflect.TypeFor[Output](), resources, connector, datatest.StudioAuthorizationTypes(t))
	if err != nil {
		t.Fatal(err)
	}
	entry.Capabilities.Connector = connector
	parentComponent := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "test.run", Name: "check"}, Routes: []*spec.Route{{Method: "POST", Path: "/test/run", Handler: "NewRun"}}}
	parentHandler, err := custom.Factory(func() xhandler.Contract[runInput, runOutput] { return &runHandler{} })()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: parentComponent, InputType: reflect.TypeFor[runInput](), OutputType: reflect.TypeFor[runOutput](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: parentHandler, HandlerOwnedOutput: true})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := artifact.Registration(registry.RegisteredComponent{})
	if err != nil {
		t.Fatal(err)
	}
	parent.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry, parent, datatest.AuthRegistration(t, db, jwt.Factory, resources)}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	bearer := jwt.BearerWithClaims(t, jwtv5.MapClaims{"sub": "alice", "iss": "run-test", "aud": "studio", "tenant": "test", "exp": time.Now().Add(time.Hour).Unix()})
	provider, err := accessconfig.FromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := provider.Resolve(oauth.WithBearer(ctx, strings.TrimPrefix(bearer, "Bearer ")))
	if err != nil || facts.Subject != "alice" {
		t.Fatalf("fixture facts invalid: %v", err)
	}
	for _, check := range []struct {
		namespace string
		want      int
	}{{"", 1}, {namespaceaccess.ID("alice", "alpha"), 1}, {namespaceaccess.ID("alice", "beta"), 0}} {
		req := httptest.NewRequest(http.MethodGet, "/_studio/report-run-access?reportId=report&subject=alice", nil)
		req.Header.Set("Authorization", bearer)
		if check.namespace != "" {
			req.Header.Set("X-Studio-Namespace", check.namespace)
		}
		scope, err := requestprovider.New(req)
		if err != nil {
			t.Fatal(err)
		}
		input := &Input{}
		input.SetReportId("report")
		input.SetSubject("alice")
		value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input, Providers: scope.Providers()})
		scope.Close()
		if err != nil {
			t.Fatal(err)
		}
		output, ok := value.(*Output)
		if !ok || len(output.Allowed) != check.want {
			t.Fatalf("namespace=%q got=%+v want rows=%d", check.namespace, value, check.want)
		}
		parentRequest := httptest.NewRequest(http.MethodPost, "/test/run", bytes.NewBufferString(`{"reportId":"report","versionNo":12,"input":{"cube":true}}`))
		parentRequest.Header.Set("Content-Type", "application/json")
		parentRequest.Header.Set("Authorization", bearer)
		if check.namespace != "" {
			parentRequest.Header.Set("X-Studio-Namespace", check.namespace)
		}
		parentScope, err := requestprovider.New(parentRequest)
		if err != nil {
			t.Fatal(err)
		}
		parentValue, parentErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/test/run", parentScope)
		parentScope.Close()
		if check.want == 1 && parentErr != nil {
			t.Fatalf("nested owner access failed: %v", parentErr)
		}
		if check.want == 0 && parentErr == nil {
			t.Fatalf("nested foreign namespace allowed: %+v", parentValue)
		}
	}
}

type runInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true"`
	VersionNo   int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true"`
	Input       sdk.PreviewInput   `parameter:"Input,kind=body,in=input,dataType=sdk.PreviewInput"`
}
type runOutput struct {
	Allowed bool `parameter:"Allowed,kind=output,in=body,dataType=bool"`
}
type runHandler struct{}

func (*runHandler) Exec(ctx context.Context, session xhandler.Session, input *runInput, output *runOutput) error {
	_, _, cancel, err := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportID, input.VersionNo, time.Second)
	if err != nil {
		return err
	}
	defer cancel()
	output.Allowed = true
	return nil
}
