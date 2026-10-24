package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/sdk"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/datly-studio/sdk/artifactrelease/usegrant"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauthorization "github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type releaseE2ESessionKey struct{}

func TestNamedReleaseMCPWindowFreshRowsAndRollback(t *testing.T) {
	f := newStockOAuthFixture(t)
	ownerToken := f.token(t, f.private, "owner", []string{"publisher"}, time.Now().Add(time.Minute))
	recipientToken := f.token(t, f.private, "recipient", []string{"release_reader"}, time.Now().Add(time.Minute))
	visitorToken := f.token(t, f.private, "visitor", []string{"visitor"}, time.Now().Add(time.Minute))
	var catalog *PublishedArtifactCatalog
	var controller *sdkrelease.Controller
	var controllerConfig sdkrelease.ControllerConfig
	var rootAvailable func(context.Context, artifactcohort.RootRef) error
	var first, second *artifactcohort.Artifact
	var pin windowprotocol.ComponentBinding
	var notifications atomic.Int32
	component := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	policyResource := authz.Resource{Tenant: "team", Kind: "policy", ID: "released-readers", Version: "logical"}
	policyRef := authz.NamedReference{Resource: policyResource, Revision: 1}
	policyDoc := authz.Document{Resource: policyResource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Released readers", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "release_reader"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "release_reader"}},
	}}}}}
	var currentPolicyRevision atomic.Int64
	currentPolicyRevision.Store(1)
	testDynamicHost(t, false, false, dynamicHostExtension{skipDefaultChecks: true, configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "team", Provider: f.provider}
		config.Forge = &ForgeConfig{ProviderIdentity: "release-fixture", Windows: []ForgeWindow{{URI: "window://demo/home", DefinitionPath: "release-window.yaml", Components: map[string]ComponentReference{"records": component}}}, Policies: []authz.Document{{Resource: authz.Resource{Tenant: "team", Kind: "window", ID: "demo/home", Version: identity.WorkingCandidate}, Revision: 1, Policies: map[string]authz.Policy{"discover": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "publisher"}}, "describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "publisher"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "publisher"}}}}}, Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: "window", ID: "demo/home", Tenant: "team"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		if err := os.WriteFile(filepath.Join(config.RootDir, "release-window.yaml"), []byte("window:\n  view:\n    content:\n      id: released-records\n      title: Records\ndataSources:\n  records:\n    id: records\n    selectors:\n      data: Records\n    cardinality: many\n    parameters: []\n"), 0600); err != nil {
			t.Fatal(err)
		}
		catalog = NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
		if _, err := catalog.Publisher.ReplaceCatalog(context.Background(), nil, nil, 0); err != nil {
			t.Fatal(err)
		}
		config.ArtifactCohorts = catalog
		config.AuthorizeArtifactHost = func(ctx context.Context) error {
			p, err := f.provider.ResolvePrincipal(ctx)
			if err != nil || p.Facts.Subject != "owner" {
				return authz.ErrDenied
			}
			return nil
		}
	}, prepare: func(host *Service) {
		ctx := context.Background()
		manager, observed, err := host.componentRuntimeWithAuthority(ctx, component, false)
		if err != nil {
			t.Fatal(err)
		}
		_ = manager.Shutdown(ctx)
		pin = observed
		first = releaseWindowFixture(t, pin)
		var envelope types.WindowResourceEnvelope
		if err = json.Unmarshal(first.Payload, &envelope); err != nil {
			t.Fatal(err)
		}
		envelope.ViewMetadata = json.RawMessage(`{"release":"v2"}`)
		payload, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		second = &artifactcohort.Artifact{Root: artifactcohort.RootRef{URI: first.Root.URI, Kind: "window", Version: 2, Hash: artifactcohort.Hash(payload)}, Payload: payload}
		ownerOnly := authz.Document{Resource: authz.Resource{Tenant: "team", Kind: "component", ID: "records", Version: "1"}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}}}}
		store, err := authz.NewStaticStore([]authz.Document{ownerOnly})
		if err != nil {
			t.Fatal(err)
		}
		host.resourceAccess.Store = store
		scope := sqlstore.Scope{Tenant: "team", Account: "21", Workspace: "workspace", Namespace: "demo"}
		sqlStore, err := sqlstore.New(host.studio, scope, "verified-publisher")
		if err != nil {
			t.Fatal(err)
		}
		artifacts := map[string]*artifactcohort.Artifact{artifactcohort.RootKey(first.Root): first, artifactcohort.RootKey(second.Root): second}
		if _, err = host.studio.ExecContext(ctx, `CREATE TABLE release_source_status(uri TEXT PRIMARY KEY,active INTEGER NOT NULL); INSERT INTO release_source_status(uri,active) VALUES('window://demo/home',1)`); err != nil {
			t.Fatal(err)
		}
		rootAvailable = func(ctx context.Context, ref artifactcohort.RootRef) error {
			var active int
			err := host.studio.QueryRowContext(ctx, `SELECT active FROM release_source_status WHERE uri=?`, ref.URI).Scan(&active)
			if err != nil {
				return err
			}
			if active != 1 {
				return artifactcohort.ErrDenied
			}
			return nil
		}
		controllerConfig = sdkrelease.ControllerConfig{
			Scope: scope, Store: sqlStore, Publisher: catalog.Publisher,
			VerifyScope: func(ctx context.Context) (sqlstore.Scope, error) {
				p, err := f.provider.ResolvePrincipal(ctx)
				if err != nil {
					return sqlstore.Scope{}, err
				}
				if p.AccountID != "21" || p.Facts.Tenant != "team" {
					return sqlstore.Scope{}, authz.ErrDenied
				}
				return scope, nil
			},
			Authorize: func(ctx context.Context, action string) error {
				p, err := f.provider.ResolvePrincipal(ctx)
				if err != nil {
					return err
				}
				if action == "use" {
					return nil
				}
				if p.Facts.Subject != "owner" {
					return authz.ErrDenied
				}
				return nil
			},
			Candidates: func(context.Context) ([]contract.Candidate, error) {
				return []contract.Candidate{{Root: first.Root, Title: "Records"}}, nil
			},
			Prepare: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				result := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, ref := range refs {
					artifact := artifacts[artifactcohort.RootKey(ref)]
					if artifact == nil {
						return nil, artifactcohort.ErrDenied
					}
					result = append(result, artifact)
				}
				return result, nil
			},
			RestoreArtifacts: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				result := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, ref := range refs {
					artifact := artifacts[artifactcohort.RootKey(ref)]
					if artifact == nil {
						return nil, artifactcohort.ErrDenied
					}
					result = append(result, artifact)
				}
				return result, nil
			},
			AvailableRoot: rootAvailable,
			ValidateArtifact: func(ctx context.Context, artifact *artifactcohort.Artifact) error {
				_, err := CompileReleaseWindowBindings(ctx, artifact, map[string]ComponentReference{"records": component}, func(ctx context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
					m, observed, err := host.componentRuntimeWithAuthority(ctx, ref, false)
					if m != nil {
						_ = m.Shutdown(ctx)
					}
					return observed, err
				})
				return err
			},
			CompileBindings: func(ctx context.Context, _ contract.Bundle, values []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
				var result []sqlstore.ComponentBinding
				for _, artifact := range values {
					bindings, err := CompileReleaseWindowBindings(ctx, artifact, map[string]ComponentReference{"records": component}, func(ctx context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
						m, observed, err := host.componentRuntimeWithAuthority(ctx, ref, false)
						if m != nil {
							_ = m.Shutdown(ctx)
						}
						return observed, err
					})
					if err != nil {
						return nil, err
					}
					for _, b := range bindings {
						result = append(result, sqlstore.ComponentBinding{Root: b.Root, DataSourceID: b.DataSourceID, ComponentKind: b.Binding.Kind, ComponentID: b.Binding.ID, ComponentRevision: b.Binding.Revision, Method: b.Reference.Method, Route: b.Reference.Route, SchemaFingerprint: b.Binding.SchemaFingerprint, ContentFingerprint: b.Binding.ContentFingerprint})
					}
				}
				return result, nil
			},
			NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
				return []authz.Document{policyDoc}, nil
			},
			Fence: func(_ context.Context, deps []authz.NamedDependency) error {
				if len(deps) != 1 || deps[0].Resource != policyResource || deps[0].Revision != currentPolicyRevision.Load() {
					return contract.ErrConflict
				}
				return nil
			},
			AdmitAudience: sdkrelease.AudienceEvaluator(catalog.Publisher, f.provider, nil),
			Notify:        func(ctx context.Context) { notifications.Add(1); catalog.NotifyListChanged(ctx) },
		}
		controller, err = sdkrelease.NewController(controllerConfig)
		if err != nil {
			t.Fatal(err)
		}
		host.config.ReleaseManagement = controller
	}, verify: func(host *Service, address string) {
		ctx := context.Background()
		var token atomic.Value
		token.Store(ownerToken)
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name, URI string }
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, request.Method)
			if request.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, request.Params.Name)
			}
			if request.Method == schema.MethodResourcesRead {
				headers.Set(schema.HeaderName, request.Params.URI)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		client := mcpclient.New("release-e2e", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		if _, err = client.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		listener := artifactcohort.NewNotificationHandler(ctx, nil)
		defer listener.Close()
		stateful, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHandler(mcpclient.NewHandler(listener)), streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name, URI string }
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, request.Method)
			if request.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, request.Params.Name)
			}
			if request.Method == schema.MethodResourcesRead {
				headers.Set(schema.HeaderName, request.Params.URI)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		subscriber := mcpclient.New("release-subscriber", "1", stateful, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer subscriber.Close()
		if _, err = subscriber.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		replica, err := artifactcohort.NewReplica(artifactcohort.Config{Enabled: true, Source: "release-fixture", Tenant: "team", Account: "21", Caller: artifactcohort.MCPCaller{Client: subscriber}, Facts: f.provider, SessionKey: func(ctx context.Context) (string, error) {
			key, _ := ctx.Value(releaseE2ESessionKey{}).(string)
			if key == "" {
				return "", artifactcohort.ErrDenied
			}
			return key, nil
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err = listener.AttachReplica(replica); err != nil {
			t.Fatal(err)
		}
		if err = replica.Sync(ctx); err != nil {
			t.Fatal(err)
		}
		publish := func(artifact *artifactcohort.Artifact, expected uint64) contract.Result {
			t.Helper()
			in := contract.PublishInput{Namespace: "demo", Name: "main", Policies: []authz.NamedReference{policyRef}, Roots: []artifactcohort.RootRef{artifact.Root}, ExpectedRevision: expected}
			raw, _ := json.Marshal(in)
			var args map[string]any
			if err := json.Unmarshal(raw, &args); err != nil {
				t.Fatal(err)
			}
			result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: contract.MethodPublish, Arguments: args}, mcpclient.WithNoRetry())
			if err != nil || result == nil || result.IsError != nil && *result.IsError {
				t.Fatalf("publish MCP: %+v %v", result, err)
			}
			var out contract.Result
			wire, _ := json.Marshal(result.StructuredContent)
			if err = json.Unmarshal(wire, &out); err != nil {
				t.Fatal(err)
			}
			return out
		}
		recipientCtx := oauth.WithBearer(ctx, recipientToken)
		if _, err = host.ExecuteComponentJSON(recipientCtx, component, pin, nil); err == nil {
			t.Fatal("recipient executed before release publication")
		}
		v1 := publish(first, 0)
		if !v1.Activated || v1.Channel.Revision != 1 {
			t.Fatalf("release not active: %+v", v1)
		}
		recipientSession := context.WithValue(recipientCtx, releaseE2ESessionKey{}, "recipient-session")
		awaitReplica := func(want artifactcohort.RootRef) {
			t.Helper()
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if err := replica.Admit(recipientSession); err == nil {
					if view, openErr := replica.Open(recipientSession, want.URI); openErr == nil && view.Ref() == want {
						return
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Fatalf("standard MCP notification did not activate local root version %d", want.Version)
		}
		awaitReplica(first.Root)
		if _, admissionErr := host.admitReleasedComponent(recipientCtx, component, pin); admissionErr != nil {
			t.Fatalf("release component admission: %v", admissionErr)
		}
		rows, err := host.ExecuteComponentJSON(recipientCtx, component, pin, nil)
		if err != nil || !strings.Contains(string(rows), "ready") || !strings.Contains(string(rows), "primary") {
			t.Fatalf("released recipient fresh SQL: %s %v", rows, err)
		}
		visitorCtx := oauth.WithBearer(ctx, visitorToken)
		if _, err = host.ExecuteComponentJSON(visitorCtx, component, pin, nil); err == nil {
			t.Fatal("unassigned visitor executed released reader")
		}
		apiStore, err := sqlstore.New(host.studio, sqlstore.Scope{Tenant: "team", Account: "21", Workspace: "workspace", Namespace: "demo"}, "api-reader")
		if err != nil {
			t.Fatal(err)
		}
		grant := &usegrant.Reader{Store: apiStore, Scope: sqlstore.Scope{Tenant: "team", Account: "21", Workspace: "workspace", Namespace: "demo"}, VerifyScope: func(ctx context.Context) (sqlstore.Scope, error) {
			p, err := f.provider.ResolvePrincipal(ctx)
			if err != nil {
				return sqlstore.Scope{}, err
			}
			if p.AccountID != "21" {
				return sqlstore.Scope{}, authz.ErrDenied
			}
			return sqlstore.Scope{Tenant: "team", Account: "21", Workspace: "workspace", Namespace: "demo"}, nil
		}, Provider: f.provider, NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
			return []authz.Document{policyDoc}, nil
		}, FenceDependencies: func(context.Context, []authz.NamedDependency) error { return nil }, RootAvailable: rootAvailable, NativeConstraints: func(ctx context.Context, b sqlstore.ComponentBinding, _ string) error {
			return host.defaultCreatorUsePolicy(ctx, authz.Resource{Tenant: "team", Kind: "component", ID: b.ComponentID, Version: b.ComponentRevision})
		}}
		authorizer, err := studioauthorization.NewSDKAuthorizer(host.studio)
		if err != nil {
			t.Fatal(err)
		}
		defer authorizer.Close(ctx)
		authorizer.ReleaseUse = func(ctx context.Context, request sqltransport.AuthorizationRequest) (bool, error) {
			return grant.Allowed(ctx, request.ReportID, request.VersionNo, request.Permission)
		}
		sdkRecipient := sdk.WithPrincipal(recipientCtx, sdk.Principal{Subject: "recipient"})
		if err = authorizer.Authorize(sdkRecipient, sqltransport.AuthorizationRequest{ReportID: "records", VersionNo: "1", Permission: "run"}); err != nil {
			t.Fatalf("SDK released run denied: %v", err)
		}
		if err = authorizer.Authorize(sdkRecipient, sqltransport.AuthorizationRequest{ReportID: "records", Permission: "edit"}); err == nil {
			t.Fatal("released recipient gained edit")
		}
		v2 := publish(second, 1)
		if !v2.Activated || v2.Channel.Revision != 2 {
			t.Fatalf("v2 not active: %+v", v2)
		}
		awaitReplica(second.Root)
		if _, err = controller.AdmitRoot(recipientCtx, first.Root); err == nil {
			t.Fatal("v1 remained selected after v2 promotion")
		}
		promote := contract.PromoteInput{Namespace: "demo", Name: "main", BundleHash: v1.Bundle.Hash, ExpectedRevision: 2}
		raw, _ := json.Marshal(promote)
		var args map[string]any
		_ = json.Unmarshal(raw, &args)
		result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: contract.MethodPromote, Arguments: args}, mcpclient.WithNoRetry())
		if err != nil || result == nil || result.IsError != nil && *result.IsError {
			t.Fatalf("rollback MCP: %+v %v", result, err)
		}
		awaitReplica(first.Root)
		if _, err = controller.AdmitRoot(recipientCtx, second.Root); err == nil {
			t.Fatal("v2 remained selected after rollback")
		}
		if notifications.Load() < 3 {
			t.Fatalf("release changes omitted notifications: %d", notifications.Load())
		}
		var dsn string
		if err = host.studio.QueryRowContext(ctx, `SELECT dsn_template FROM connectors WHERE name='source'`).Scan(&dsn); err != nil {
			t.Fatal(err)
		}
		dataDB, err := sql.Open("sqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer dataDB.Close()
		if _, err = dataDB.ExecContext(ctx, `UPDATE records SET name='fresh' WHERE id=1`); err != nil {
			t.Fatal(err)
		}
		rows, err = host.ExecuteComponentJSON(recipientCtx, component, pin, nil)
		if err != nil || !strings.Contains(string(rows), "fresh") {
			t.Fatalf("rollback cached business rows: %s %v", rows, err)
		}
		// The named definition changed through its owning policy store. The
		// post-commit hook refreshes the SAME bundle with no manual Promote.
		definition := policyDoc.Policies[authz.NamedDefinitionAction].Definition
		definition.Policies["describe"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "visitor"}}
		definition.Policies["execute"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "visitor"}}
		policyDoc.Revision = 2
		currentPolicyRevision.Store(2)
		if err = controller.RefreshPolicies(ctx, policyResource); err != nil {
			t.Fatalf("automatic named-policy projection: %v", err)
		}
		if notifications.Load() < 4 {
			t.Fatal("named-policy change emitted no standard notification")
		}
		if _, err = host.ExecuteComponentJSON(recipientCtx, component, pin, nil); err == nil {
			t.Fatal("former named recipient survived update")
		}
		rows, err = host.ExecuteComponentJSON(visitorCtx, component, pin, nil)
		if err != nil || !strings.Contains(string(rows), "fresh") {
			t.Fatalf("new named recipient cannot run fresh SQL: %s %v", rows, err)
		}
		if err = authorizer.Authorize(sdkRecipient, sqltransport.AuthorizationRequest{ReportID: "records", VersionNo: "1", Permission: "run"}); err == nil {
			t.Fatal("separate SDK kept old policy grant")
		}
		visitorSDK := sdk.WithPrincipal(visitorCtx, sdk.Principal{Subject: "visitor"})
		if err = authorizer.Authorize(visitorSDK, sqltransport.AuthorizationRequest{ReportID: "records", VersionNo: "1", Permission: "run"}); err != nil {
			t.Fatalf("separate SDK missed new policy grant: %v", err)
		}
		visitorSession := context.WithValue(visitorCtx, releaseE2ESessionKey{}, "visitor-session")
		policyDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(policyDeadline) {
			if admitErr := replica.Admit(visitorSession); admitErr == nil {
				if view, openErr := replica.Open(visitorSession, first.Root.URI); openErr == nil && view.Ref() == first.Root {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		if view, openErr := replica.Open(visitorSession, first.Root.URI); openErr != nil || view.Ref() != first.Root {
			t.Fatalf("notification replica did not adopt policy update: %v", openErr)
		}
		if _, err = host.studio.ExecContext(ctx, `UPDATE release_source_status SET active=0 WHERE uri=?`, first.Root.URI); err != nil {
			t.Fatal(err)
		}
		if err = catalog.RemoveRoot(ctx, first.Root.URI); err != nil {
			t.Fatal(err)
		}
		if _, err = host.ExecuteComponentJSON(visitorCtx, component, pin, nil); err == nil {
			t.Fatal("withdrawn root still executed")
		}
		if err = authorizer.Authorize(visitorSDK, sqltransport.AuthorizationRequest{ReportID: "records", VersionNo: "1", Permission: "run"}); err == nil {
			t.Fatal("separate SDK process revived withdrawn root")
		}
		withdrawDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(withdrawDeadline) {
			if _, openErr := replica.Open(visitorSession, first.Root.URI); openErr != nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if _, openErr := replica.Open(visitorSession, first.Root.URI); openErr == nil {
			t.Fatal("notification subscriber kept withdrawn release open")
		}
		freshPublisher := artifactcohort.NewPublisher()
		if _, err = freshPublisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
			t.Fatal(err)
		}
		restartedConfig := controllerConfig
		restartedConfig.Publisher = freshPublisher
		restartedConfig.AdmitAudience = sdkrelease.AudienceEvaluator(freshPublisher, f.provider, nil)
		restarted, err := sdkrelease.NewController(restartedConfig)
		if err != nil {
			t.Fatal(err)
		}
		if err = restarted.Recover(ctx); err != nil {
			t.Fatalf("withdrawn root restart: %v", err)
		}
		if snapshot, readErr := restarted.ActiveSnapshot(recipientCtx); readErr != nil || len(snapshot.Selections) != 0 {
			t.Fatalf("fresh Publisher resurrected withdrawn root: %+v %v", snapshot, readErr)
		}
	}})
}
