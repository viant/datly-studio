//go:build artifactcohortintegration

package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	coreui "github.com/viant/agently-core/adapter/http/ui"
	dsproto "github.com/viant/agently-core/protocol/datasource"
	coresdk "github.com/viant/agently-core/sdk"
	coreapi "github.com/viant/agently-core/sdk/api"
	coreds "github.com/viant/agently-core/service/datasource"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/forge/backend/artifactcohort"
	forgeservice "github.com/viant/forge/backend/provider"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type notifiedBusinessExecutor struct {
	client  mcpclient.Interface
	calls   atomic.Int64
	entered chan struct{}
	release chan struct{}
}

func (e *notifiedBusinessExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if name != "studio:records.read" {
		return "", fmt.Errorf("unexpected business method")
	}
	e.calls.Add(1)
	if args["block"] == true && e.entered != nil {
		select {
		case e.entered <- struct{}{}:
		default:
		}
		select {
		case <-e.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	result, err := e.client.CallTool(ctx, &schema.CallToolRequestParams{Name: "records.read", Arguments: args}, mcpclient.WithNoRetry())
	if err != nil || result == nil || result.IsError != nil && *result.IsError {
		return "", fmt.Errorf("business MCP call failed")
	}
	raw, err := json.Marshal(result.StructuredContent)
	return string(raw), err
}

func notifiedWindowBody(t *testing.T, id string) json.RawMessage {
	return notifiedWindowBodyWithSource(t, id, "records")
}
func notifiedWindowBodyWithSource(t *testing.T, id, sourceID string) json.RawMessage {
	t.Helper()
	descriptor, err := types.CanonicalWindowDescriptor(json.RawMessage(`{"id":"` + sourceID + `","selectors":{"data":"data"},"backend":{"kind":"mcp_tool","service":"studio","method":"records.read"}}`))
	if err != nil {
		t.Fatal(err)
	}
	descriptorHash, err := types.WindowDescriptorFingerprint(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	variant := types.WindowResourceVariant{Window: &types.Window{WindowKey: "published", View: types.View{Content: &types.Container{ID: id}}, DataSource: map[string]types.DataSource{sourceID: {}}, ResourceDependencies: map[string]string{sourceID: descriptorHash}}, DataSources: map[string]json.RawMessage{sourceID: descriptor}}
	fingerprint, err := types.WindowVariantFingerprint(variant)
	if err != nil {
		t.Fatal(err)
	}
	envelope := types.WindowResourceEnvelope{SchemaVersion: 2, Format: types.WindowBundleFormat, Targets: []types.WindowTargetBinding{{Variant: fingerprint}}, Variants: map[string]types.WindowResourceVariant{fingerprint: variant}}
	if err := envelope.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// This fixture simulates a publication change on an authenticated stateful
// Forge MCP server. It is not evidence that the native stateless :8096 server
// delivers push notifications; production there uses conditional polling.
func TestPublishedNotificationRebindsCoreHTTPWindowAndDatasource(t *testing.T) {
	identity := newStockOAuthFixture(t)
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	uri := "window://team/published"
	validate := func(_ context.Context, raw json.RawMessage) error {
		var e types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		return e.Validate()
	}
	v1, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 1}, notifiedWindowBody(t, "version-one"), validate)
	if err != nil {
		t.Fatal(err)
	}
	otherURI := "window://team/other"
	other, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: otherURI, Kind: "window", Version: 1}, notifiedWindowBodyWithSource(t, "other-window", "other"), validate)
	if err != nil {
		t.Fatal(err)
	}
	policy := authz.Resource{Tenant: "team", Kind: "cohort", ID: "operators", Version: "logical"}
	docs := []authz.Document{{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}}}}}
	if _, err = catalog.ReplaceCatalog(ctx, docs, []artifactcohort.Cohort{{Tenant: "team", Name: "operators", PolicyResource: policy, Roots: []artifactcohort.RootRef{{URI: otherURI, Kind: "window", Version: 0}, {URI: uri, Kind: "window", Version: 0}}}}, catalog.Publisher.Generation()); err != nil {
		t.Fatal(err)
	}
	serviceToken := identity.token(t, identity.private, "agently-service", []string{"operator"}, time.Now().Add(time.Minute))
	provider := &forgeservice.PrimitiveProvider{Host: forgeservice.PrimitiveHostFuncs{CatalogFunc: func(context.Context, *windowprotocol.CatalogInput) (*windowprotocol.Catalog, error) {
		return &windowprotocol.Catalog{ContractVersion: windowprotocol.Version, CatalogRevision: "fixture", Windows: []windowprotocol.WindowSummary{}}, nil
	}}, Authority: forgeservice.PrimitiveAuthorityFuncs{AuthenticateFunc: func(context.Context) (string, error) { return "fixture", nil }, AuthorizeFunc: func(context.Context, string, string, string, string) error { return nil }}}
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: identity.provider}
		config.ForgeProvider = provider
		config.ArtifactCohorts = catalog
		config.AuthorizeArtifactHost = func(c context.Context) error {
			p, e := identity.provider.ResolvePrincipal(c)
			if e != nil || p.Facts.Subject != "agently-service" {
				return errors.New("configured host required")
			}
			return nil
		}
	}, verify: func(_ *Service, address string) {
		var token atomic.Value
		token.Store(serviceToken)
		notifications := artifactcohort.NewNotificationHandler(ctx, nil)
		defer notifications.Close()
		newTransport := func(path string, bearer *atomic.Value, handler *artifactcohort.NotificationHandler) (*streamable.Client, error) {
			return streamable.New(ctx, "http://"+address+path, streamable.WithHandler(mcpclient.NewHandler(handler)), streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: bearer}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
				var request struct {
					Method string
					Params struct{ Name string }
				}
				if err := json.Unmarshal(body, &request); err != nil {
					return err
				}
				headers.Set(schema.HeaderMethod, request.Method)
				if request.Method == schema.MethodToolsCall {
					headers.Set(schema.HeaderName, request.Params.Name)
				}
				return nil
			}))
		}
		transport, err := newTransport("/forge/mcp", &token, notifications)
		if err != nil {
			t.Fatal(err)
		}
		mcp := mcpclient.New("core-notification", "1", transport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer mcp.Close()
		if _, err := mcp.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		counted := &countedCohortToolClient{next: mcp, calls: map[string]int{}, fetchByURI: map[string]int{}}
		replica, err := artifactcohort.NewReplica(artifactcohort.Config{Enabled: true, Source: "datly-mcp", Tenant: "team", Account: "21", Caller: artifactcohort.MCPCaller{Client: counted}, Facts: identity.provider, SessionKey: func(c context.Context) (string, error) {
			s, _ := c.Value(cohortTestSessionKey{}).(string)
			if s == "" {
				return "", artifactcohort.ErrDenied
			}
			return s, nil
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err := notifications.AttachReplica(replica); err != nil {
			t.Fatal(err)
		}
		if err := replica.Sync(ctx); err != nil {
			t.Fatal(err)
		}
		userToken := identity.token(t, identity.private, "owner", []string{"operator"}, time.Now().Add(time.Minute))
		user := context.WithValue(oauth.WithBearer(ctx, userToken), cohortTestSessionKey{}, "owner")
		if err := replica.Admit(user); err != nil {
			t.Fatal(err)
		}
		var businessToken atomic.Value
		businessToken.Store(userToken)
		businessNotifications := artifactcohort.NewNotificationHandler(ctx, nil)
		defer businessNotifications.Close()
		businessTransport, err := newTransport("/mcp", &businessToken, businessNotifications)
		if err != nil {
			t.Fatal(err)
		}
		businessMCP := mcpclient.New("core-business", "1", businessTransport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer businessMCP.Close()
		if _, err := businessMCP.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		business := &notifiedBusinessExecutor{client: businessMCP}
		datasources := coreds.New(coreds.Options{Executor: business, ReadContext: func(context.Context) (context.Context, error) { t.Fatal("legacy metadata read"); return nil, nil }})
		execute := func(c context.Context, d *dsproto.DataSource, inputs map[string]interface{}, parameters map[string]any) (*coreapi.FetchDatasourceOutput, error) {
			out, e := datasources.FetchPublished(c, d, inputs, parameters)
			if e != nil {
				return nil, e
			}
			return &coreapi.FetchDatasourceOutput{Rows: out.Rows, DataInfo: out.DataInfo, Metrics: out.Metrics}, nil
		}
		open := func(c context.Context, provider, root string, target *types.WindowTarget) (*types.WindowResourceVariant, string, error) {
			if provider != "datly-mcp" {
				return nil, "", artifactcohort.ErrDenied
			}
			return replica.OpenWindowVariant(c, root, target)
		}
		owner := func(c context.Context) (string, error) {
			s, _ := c.Value(cohortTestSessionKey{}).(string)
			if s == "" {
				return "", artifactcohort.ErrDenied
			}
			return s, nil
		}
		bindings := coresdk.NewPublishedWindowBindings(open, execute, owner)
		ui := coreui.NewEmbeddedHandlerWithAuthorization(t.TempDir(), nil, nil, nil, coreui.WithArtifactCohortWindowSource("datly-mcp", replica, bindings.Bind))
		sdk := coresdk.NewHandler(nil, coresdk.WithPublishedWindowBindings(bindings))
		mux := http.NewServeMux()
		trusted := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r.WithContext(user)) })
		}
		mux.Handle("/window/", trusted(ui))
		mux.Handle("/v1/api/", trusted(sdk))
		server := httptest.NewServer(mux)
		defer server.Close()
		query := url.Values{"providerIdentity": {"datly-mcp"}, "namespace": {"team"}, "resourceRef": {`{"uri":"window://team/published"}`}, "clientId": {"client"}, "windowId": {"instance"}}
		get := func() (int, string, string) {
			request, _ := http.NewRequest(http.MethodGet, server.URL+"/window/published?"+query.Encode(), nil)
			request.Header.Set("Mcp-Session-Id", "ui-session")
			response, e := server.Client().Do(request)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := io.ReadAll(response.Body)
			response.Body.Close()
			var body struct {
				Data types.Window `json:"data"`
			}
			_ = json.Unmarshal(raw, &body)
			if body.Data.View.Content == nil {
				return response.StatusCode, "", string(raw)
			}
			return response.StatusCode, body.Data.View.Content.ID, string(raw)
		}
		postAt := func(c context.Context, sourceID, windowID, body string) (int, string, error) {
			request, _ := http.NewRequestWithContext(c, http.MethodPost, server.URL+"/v1/api/datasources/"+sourceID+"/fetch?cohort=1&windowId="+windowID, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Mcp-Session-Id", "ui-session")
			response, e := server.Client().Do(request)
			if e != nil {
				return 0, "", e
			}
			out, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return response.StatusCode, string(out), nil
		}
		post := func() (int, string) {
			status, body, e := postAt(ctx, "records", "instance", `{"inputs":{}}`)
			if e != nil {
				t.Fatal(e)
			}
			return status, body
		}
		if status, id, raw := get(); status != 200 || id != "version-one" || !strings.Contains(raw, "/v1/api/datasources/records/fetch?cohort=1") {
			t.Fatalf("v1 Core GET %d %s %s", status, id, raw)
		}
		if status, body := post(); status != 200 || business.calls.Load() != 1 {
			t.Fatalf("v1 business POST %d calls=%d body=%s", status, business.calls.Load(), body)
		}
		unchangedFetch := counted.count(artifactcohort.MethodFetch)
		if err := replica.Sync(ctx); err != nil || counted.count(artifactcohort.MethodFetch) != unchangedFetch {
			t.Fatalf("unchanged fetch count=%d err=%v", counted.count(artifactcohort.MethodFetch), err)
		}
		v2, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 2}, notifiedWindowBody(t, "version-two"), validate)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(3 * time.Second)
		for {
			view, e := replica.Open(user, uri)
			if e == nil && view.Ref() == v2.Root {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("notification did not activate v2: %v", e)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if counted.count(artifactcohort.MethodFetch) != unchangedFetch+1 {
			t.Fatalf("changed root fetches=%d", counted.count(artifactcohort.MethodFetch)-unchangedFetch)
		}
		if status, _ := post(); status != 403 || business.calls.Load() != 1 {
			t.Fatalf("old binding survived v2: status=%d calls=%d", status, business.calls.Load())
		}
		if status, id, raw := get(); status != 200 || id != "version-two" {
			t.Fatalf("v2 Core GET %d %s %s", status, id, raw)
		}
		if status, body := post(); status != 200 || business.calls.Load() != 2 {
			t.Fatalf("v2 business POST %d calls=%d body=%s", status, business.calls.Load(), body)
		}
		if v1.Root.Hash == v2.Root.Hash {
			t.Fatal("changed closed bytes kept old hash")
		}
		otherQuery := url.Values{"providerIdentity": {"datly-mcp"}, "namespace": {"team"}, "resourceRef": {`{"uri":"window://team/other"}`}, "clientId": {"client"}, "windowId": {"other-instance"}}
		getOther := func() (int, string) {
			req, _ := http.NewRequest(http.MethodGet, server.URL+"/window/other?"+otherQuery.Encode(), nil)
			req.Header.Set("Mcp-Session-Id", "ui-session")
			resp, e := server.Client().Do(req)
			if e != nil {
				t.Fatal(e)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resp.StatusCode, string(body)
		}
		if status, body := getOther(); status != 200 || !strings.Contains(body, "other-window") || !strings.Contains(body, "/v1/api/datasources/other/fetch?cohort=1") {
			t.Fatalf("independent window GET %d %s", status, body)
		}
		if status, body, e := postAt(ctx, "other", "other-instance", `{"inputs":{}}`); e != nil || status != 200 {
			t.Fatalf("independent datasource POST %d %v %s", status, e, body)
		}
		business.entered = make(chan struct{}, 2)
		business.release = make(chan struct{})
		defer close(business.release)
		type callResult struct {
			status int
			err    error
		}
		slowDone := make(chan callResult, 1)
		go func() {
			status, _, e := postAt(ctx, "records", "instance", `{"inputs":{"block":true}}`)
			slowDone <- callResult{status, e}
		}()
		select {
		case <-business.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("slow business call did not enter")
		}
		v3, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 3}, notifiedWindowBody(t, "version-three"), validate)
		if err != nil {
			t.Fatal(err)
		}
		deadline = time.Now().Add(2 * time.Second)
		for {
			view, e := replica.Open(user, uri)
			if e == nil && view.Ref() == v3.Root {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("notification stalled behind active tool: %v", e)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if counted.fetchCount(other.Root.URI) != 1 {
			t.Fatalf("unchanged other window refetched %d", counted.fetchCount(other.Root.URI))
		}
		fastStart := time.Now()
		if status, body := getOther(); status != 200 || !strings.Contains(body, "other-window") {
			t.Fatalf("other window blocked %d %s", status, body)
		}
		if status, body, e := postAt(ctx, "other", "other-instance", `{"inputs":{}}`); e != nil || status != 200 {
			t.Fatalf("other datasource blocked %d %v %s", status, e, body)
		}
		if elapsed := time.Since(fastStart); elapsed > 600*time.Millisecond {
			t.Fatalf("unrelated GET+POST blocked %s", elapsed)
		}
		business.release <- struct{}{}
		select {
		case result := <-slowDone:
			if result.err != nil || result.status != 403 {
				t.Fatalf("stale in-flight result escaped: %+v", result)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("slow call did not release")
		}
		if status, id, _ := get(); status != 200 || id != "version-three" {
			t.Fatalf("v3 GET %d %s", status, id)
		}
		cancelCtx, cancelSlow := context.WithCancel(ctx)
		cancelDone := make(chan callResult, 1)
		go func() {
			status, _, e := postAt(cancelCtx, "records", "instance", `{"inputs":{"block":true}}`)
			cancelDone <- callResult{status, e}
		}()
		select {
		case <-business.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("cancelled call did not enter")
		}
		cancelSlow()
		select {
		case <-cancelDone:
		case <-time.After(time.Second):
			t.Fatal("cancelled call held execution")
		}
		if status, _, e := postAt(ctx, "other", "other-instance", `{"inputs":{}}`); e != nil || status != 200 {
			t.Fatalf("cancel held unrelated datasource: %d %v", status, e)
		}
	}})
}
