package warmup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/internal/datatest"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	xhandler "github.com/viant/xdatly/handler"
)

type privateRequestValue struct{}
type identityInput struct {
	JWT *jwt.Claims `parameter:"JWT,kind=header,in=Authorization,dataType=string,required,errorCode=401" codec:"JwtClaim"`
}
type identityHandler struct{}

func (identityHandler) Exec(_ context.Context, _ xhandler.Session, input *identityInput, output *jwt.Claims) error {
	if input.JWT == nil || input.JWT.Subject != "alice" {
		return fmt.Errorf("identity rejected")
	}
	*output = *input.JWT
	return nil
}

func TestMCPWarmupRetainsAuthorizedContextAfterClientDisconnect(t *testing.T) {
	ctx := t.Context()
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "studio.db")
	db := datatest.OpenSQLite(t, "warmup_mcp_worker", "studio")
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dbPath); err != nil {
		t.Fatal(err)
	}
	studioDSN := "file:" + dbPath + "?cache=shared"
	connections, err := connector.Open(ctx, []connector.Config{{Name: "studio", Driver: "sqlite", DSN: studioDSN}}, "studio")
	if err != nil {
		t.Fatal(err)
	}
	defer connections.Close()
	db = connections.SQL.DB
	sourcePath := filepath.Join(directory, "source.db")
	sourceDSN := "file:" + sourcePath + "?cache=shared"
	source, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err = source.ExecContext(ctx, "CREATE TABLE RECORDS(ID INTEGER PRIMARY KEY, OWNER INTEGER, NAME TEXT); INSERT INTO RECORDS VALUES(1,1,'One'),(2,2,'Two')"); err != nil {
		t.Fatal(err)
	}
	batchDSN := "file:" + filepath.Join(directory, "batch.db") + "?cache=shared"
	batch, err := sql.Open("sqlite", batchDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer batch.Close()
	if _, err = batch.ExecContext(ctx, "CREATE TABLE RECORDS(ID INTEGER PRIMARY KEY, OWNER INTEGER, NAME TEXT); INSERT INTO RECORDS VALUES(1,1,'BatchOne'),(2,1,'BatchTwo'),(3,2,'BatchThree')"); err != nil {
		t.Fatal(err)
	}
	namespace := namespaceaccess.ID("alice", "alpha")
	other := namespaceaccess.ID("alice", "beta")
	now := "2026-09-29 00:00:00"
	if err = datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "dsn_template": sourceDSN, "owner_id": "alice", "status": "active", "options_json": "{}", "created_at": now, "updated_at": now}, {"name": "batch", "driver": "sqlite", "dsn_template": batchDSN, "owner_id": "alice", "status": "active", "options_json": "{}", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"namespace_id": namespace, "owner_id": "alice", "name": "alpha", "title": "Alpha", "status": "active", "created_at": now, "updated_at": now}, {"namespace_id": other, "owner_id": "alice", "name": "beta", "title": "Beta", "status": "active", "created_at": now, "updated_at": now}}},
	); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"accepted", "rejected", "missing"} {
		scope := "example.com/fixture/" + id
		warmupConnector := "batch"
		if id == "missing" {
			warmupConnector = "unavailable"
		}
		dql := fmt.Sprintf(`#package('%s')
#import('jwt','github.com/viant/scy/auth/jwt')
#setting($_ = $connector('main'))
#setting($_ = $route('/records/%s','GET'))
#setting($_ = $cache('records','1m').WithLocation('%s'))
#setting($_ = $cache_warmup('ID','Connector=%s','Limit=0'))
#define($_ = $Caller<string,*jwt.Claims>(header/Authorization).WithCodec('JwtClaim').Required())
#define($_ = $Identity<*jwt.Claims>(component/GET:/identity))
#define($_ = $Tenant<int>(param/Identity.UserID).WithPredicate(0,'expr','OWNER = ?'))
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*, type(records,'Record') FROM (SELECT ID, NAME FROM RECORDS ${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}) records`, scope, id, filepath.Join(directory, id, "cache"), warmupConnector)
		if err = datatest.Hydrate(ctx, db,
			datatest.Table{Name: "components", Rows: []datatest.Row{{"id": id, "namespace_id": namespace, "namespace": "alpha", "slug": id, "title": id, "owner_id": "alice", "status": "draft", "default_connector_name": "main", "component_scope": scope, "component_name": "reader", "created_at": now, "updated_at": now}}},
			datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": id, "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": dql, "generated_dql": dql, "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": id, "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": now}}},
		); err != nil {
			t.Fatal(err)
		}
	}
	jwtFixture := datatest.NewJWTFixture(t)
	deniedFixture := datatest.NewJWTFixture(t)
	publicPath := filepath.Join(directory, "public.pem")
	if err = os.WriteFile(publicPath, jwtFixture.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "warmup-test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", publicPath)
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	reached, release := make(chan struct{}, 1), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	var cleaned atomic.Int32
	var previewing atomic.Bool
	authoring, err := host.NewAuthoring(host.AuthoringConfig{ModulePath: "example.com/fixture", Dependencies: func(ctx context.Context, component *spec.Component) (*preview.DependencyRuntime, error) {
		selected, ok := sdk.NamespaceSelectionFromContext(ctx)
		if !ok || selected != namespace {
			return nil, fmt.Errorf("authorized namespace lost")
		}
		principal, ok := sdk.PrincipalFromContext(ctx)
		if !ok || principal.Subject != "alice" {
			return nil, fmt.Errorf("verified principal lost")
		}
		credential, ok := sdk.VerifiedCredentialFromContext(ctx)
		if !ok || credential.Bearer == "" {
			return nil, fmt.Errorf("verified credential lost")
		}
		if ctx.Value(privateRequestValue{}) != nil && !previewing.Load() {
			return nil, fmt.Errorf("unrelated request state retained")
		}
		if _, ok := ctx.Deadline(); !ok {
			return nil, fmt.Errorf("server-owned timeout lost")
		}
		selectedFactory := jwtFixture.Factory
		if strings.HasSuffix(component.Key.Scope, "/accepted") {
			select {
			case reached <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		} else {
			selectedFactory = deniedFixture.Factory
		}
		artifact, e := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "fixture/identity", Name: "Identity"}, Routes: []*spec.Route{{Method: "GET", Path: "/identity", Internal: true}}}, InputType: reflect.TypeFor[identityInput](), OutputType: reflect.TypeFor[jwt.Claims](), CodecFactory: selectedFactory})
		if e != nil {
			return nil, e
		}
		entry, e := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[identityInput, jwt.Claims](identityHandler{})})
		if e != nil {
			return nil, e
		}
		return &preview.DependencyRuntime{Components: []*registry.RegisteredComponent{entry}, Shutdown: func(context.Context) error { cleaned.Add(1); return nil }}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := (config.Loader{}).Load(ctx, filepath.Join(root, "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.BaseDir = root
	cfg.Endpoint.Address = "127.0.0.1:0"
	cfg.MCP.Address = "127.0.0.1:0"
	cfg.Connectors = []connector.Config{{Name: "studio", Driver: "sqlite", DSN: studioDSN}, {Name: "authz", AliasOf: "studio"}}
	cfg.JWTValidator = &verifier.Config{RSA: []*scy.Resource{{URL: "fixture-public-key", Data: jwtFixture.PublicKeyPEM(t)}}}
	cfg.JWTClaims.Issuer = "warmup-test"
	cfg.JWTClaims.Audience = "studio"
	server, err := standalone.New(ctx, standalone.Options{Config: cfg, RequireLinked: true, HTTPMiddleware: func(next http.Handler) http.Handler {
		next = authoring.Middleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := context.WithValue(r.Context(), privateRequestValue{}, true)
			c = studiocodecs.WithFactory(c, studiocodecs.New(jwtFixture.Factory))
			next.ServeHTTP(w, r.WithContext(c))
		})
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown(context.Background())
	serveCtx, stop := context.WithCancel(ctx)
	defer stop()
	served := make(chan error, 1)
	go func() { served <- server.Serve(serveCtx, io.Discard) }()
	readyCtx, cancelReady := context.WithTimeout(ctx, 10*time.Second)
	defer cancelReady()
	addresses, err := server.WaitReady(readyCtx)
	if err != nil {
		t.Fatal(err)
	}
	bearer := jwtFixture.BearerWithClaims(t, jwtv5.MapClaims{"sub": "alice", "user_id": 1, "iss": "warmup-test", "aud": "studio", "tenant": "tenant", "exp": time.Now().Add(time.Hour).Unix()})
	callRaw := func(callCtx context.Context, tool, token string, arguments map[string]any) (json.RawMessage, bool) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": arguments}})
		request, e := http.NewRequestWithContext(callCtx, http.MethodPost, "http://"+addresses[1]+"/mcp", bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("Authorization", token)
		response, e := http.DefaultClient.Do(request)
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(response.Body)
		response.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		var wire struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError bool            `json:"isError"`
				Data    json.RawMessage `json:"structuredContent"`
			} `json:"result"`
		}
		if e = json.Unmarshal(data, &wire); e != nil {
			t.Fatalf("MCP response %d: %s", response.StatusCode, data)
		}
		return wire.Result.Data, response.StatusCode == 200 && len(wire.Error) == 0 && !wire.Result.IsError
	}
	call := func(callCtx context.Context, tool, token string, arguments map[string]any) (sdk.WarmupRun, bool) {
		raw, ok := callRaw(callCtx, tool, token, arguments)
		var run sdk.WarmupRun
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &run); err != nil {
				t.Fatal(err)
			}
		}
		return run, ok
	}
	args := func(id, selected string) map[string]any {
		return map[string]any{"namespaceId": selected, "reportId": id, "versionNo": 1}
	}
	for _, tc := range []struct{ name, token, selection string }{{"invalid JWT", "Bearer invalid.token.signature", namespace}, {"wrong namespace", bearer, other}} {
		t.Run(tc.name, func(t *testing.T) {
			if run, ok := call(ctx, "studio.sdk.versions.warmup", tc.token, args("accepted", tc.selection)); ok || run.RunID != "" {
				t.Fatalf("unauthorized warmup accepted: %+v", run)
			}
		})
	}
	var jobs int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_warmup_runs").Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if jobs != 0 {
		t.Fatal("rejection created a job")
	}
	for _, id := range []string{"accepted", "rejected", "missing"} {
		t.Run(id, func(t *testing.T) {
			clientCtx, disconnect := context.WithCancel(ctx)
			accepted, ok := call(clientCtx, "studio.sdk.versions.warmup", bearer, args(id, namespace))
			if !ok || accepted.Status != "accepted" {
				disconnect()
				t.Fatalf("warmup not accepted: %+v", accepted)
			}
			if id == "accepted" {
				select {
				case <-reached:
				case <-time.After(10 * time.Second):
					t.Fatal("worker did not reach dependency resolver")
				}
			}
			disconnect()
			http.DefaultClient.CloseIdleConnections()
			if id == "accepted" {
				close(release)
			}
			var run sdk.WarmupRun
			deadline := time.Now().Add(15 * time.Second)
			for time.Now().Before(deadline) {
				var found bool
				run, found = call(ctx, "studio.sdk.versions.warmup_get", bearer, map[string]any{"namespaceId": namespace, "reportId": id, "runId": accepted.RunID})
				if !found {
					t.Fatal("durable result unavailable")
				}
				if run.Status != "accepted" && run.Status != "running" {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			expected := "completed"
			if id != "accepted" {
				expected = "failed"
			}
			if run.Status != expected {
				t.Fatalf("terminal run=%+v", run)
			}
			if id == "rejected" && (len(run.Diagnostics) != 1 || !strings.Contains(run.Diagnostics[0].Message, "prepare dynamic cache warmup")) {
				t.Fatalf("rejection did not come from native JWT preparation: %+v", run.Diagnostics)
			}
			if id == "missing" && (len(run.Diagnostics) != 1 || !strings.Contains(run.Diagnostics[0].Message, `requires active Studio connector "unavailable"`)) {
				t.Fatalf("missing warmup connector did not fail explicitly: %+v", run.Diagnostics)
			}
			if id == "accepted" && (run.Entries != 2 || run.CompletedCases != 1) {
				t.Fatalf("successful run=%+v", run)
			}
			files := 0
			e := filepath.WalkDir(filepath.Join(directory, id, "cache"), func(_ string, entry fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if !entry.IsDir() {
					files++
				}
				return nil
			})
			if e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
			if id == "accepted" && files == 0 {
				t.Fatal("cache preparation produced no files")
			}
			if id != "accepted" && (files != 0 || run.Entries != 0) {
				t.Fatalf("JWT rejection wrote cache: files=%d run=%+v", files, run)
			}
		})
	}
	// Two indexed entries prove warmup used batch: main has only one authorized
	// row. Ordinary preview must still use main without requiring batch.
	if _, err = db.ExecContext(ctx, "UPDATE connectors SET status='disabled' WHERE name='batch'"); err != nil {
		t.Fatal(err)
	}
	previewing.Store(true)
	raw, ok := callRaw(ctx, "studio.sdk.preview.execute", bearer, map[string]any{"namespaceId": namespace, "reportId": "accepted", "versionNo": 1, "input": map[string]any{"limit": 10}})
	if !ok || !bytes.Contains(raw, []byte(`"Name":"One"`)) || bytes.Contains(raw, []byte("Batch")) {
		t.Fatalf("ordinary preview required a warmup-only source or changed its native identity scope: %s", raw)
	}

	if cleaned.Load() != 4 {
		t.Fatalf("dependency cleanup count=%d", cleaned.Load())
	}
}
