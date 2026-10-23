package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/forge/backend/artifactcohort"
	forgeservice "github.com/viant/forge/backend/provider"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type cohortNotificationRecorder struct {
	mu     sync.Mutex
	events []jsonrpc.Notification
}

type cohortTestSessionKey struct{}
type countedCohortToolClient struct {
	next       mcpclient.Interface
	mu         sync.Mutex
	calls      map[string]int
	fetchByURI map[string]int
}

func (c *countedCohortToolClient) CallTool(ctx context.Context, params *schema.CallToolRequestParams, options ...mcpclient.RequestOption) (*schema.CallToolResult, error) {
	c.mu.Lock()
	c.calls[params.Name]++
	if params.Name == artifactcohort.MethodFetch {
		uri := ""
		switch root := params.Arguments["root"].(type) {
		case artifactcohort.RootRef:
			uri = root.URI
		case map[string]any:
			uri, _ = root["uri"].(string)
		}
		if uri != "" {
			c.fetchByURI[uri]++
		}
	}
	c.mu.Unlock()
	return c.next.CallTool(ctx, params, options...)
}
func (c *countedCohortToolClient) count(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[name]
}
func (c *countedCohortToolClient) fetchCount(uri string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fetchByURI[uri]
}

func (r *cohortNotificationRecorder) Notify(_ context.Context, event *jsonrpc.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *event
	copy.Params = append(json.RawMessage(nil), event.Params...)
	r.events = append(r.events, copy)
	return nil
}
func (r *cohortNotificationRecorder) snapshot() []jsonrpc.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]jsonrpc.Notification(nil), r.events...)
}

func closedWindowForCohort(t *testing.T, id string) json.RawMessage {
	t.Helper()
	window := &types.Window{View: types.View{Content: &types.Container{ID: id}}}
	variant := types.WindowResourceVariant{Window: window, DataSources: map[string]json.RawMessage{}}
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

func TestPublishedArtifactCohortToolsOnAuthenticatedForgeMCP(t *testing.T) {
	identity := newStockOAuthFixture(t)
	publisher := artifactcohort.NewPublisher()
	catalog := NewPublishedArtifactCatalog(publisher)
	ctx := context.Background()
	root := artifactcohort.RootRef{URI: "window://team/published", Kind: "window", Version: 1}
	validate := func(_ context.Context, raw json.RawMessage) error {
		var window types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &window); err != nil {
			return err
		}
		return window.Validate()
	}
	first, gen, err := catalog.Publish(ctx, root, closedWindowForCohort(t, "first"), validate)
	if err != nil || gen != 1 {
		t.Fatalf("publish first: %v generation=%d", err, gen)
	}
	intentRoot := artifactcohort.RootRef{URI: "intent://team/help", Kind: "intent", Version: 1}
	intent, gen, err := catalog.Publish(ctx, intentRoot, json.RawMessage(`{"intent":"help"}`), func(_ context.Context, raw json.RawMessage) error {
		if !json.Valid(raw) {
			return artifactcohort.ErrInvalid
		}
		return nil
	})
	if err != nil || gen != 2 {
		t.Fatalf("publish intent: %v generation=%d", err, gen)
	}
	policyResource := authz.Resource{Tenant: "team", Kind: "cohort", ID: "team/operators", Version: "logical"}
	pinnedPolicy := authz.Resource{Tenant: "team", Kind: "cohort", ID: "team/pinned", Version: "logical"}
	policies := []authz.Document{
		{Resource: policyResource, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}}}},
		{Resource: pinnedPolicy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "pinner"}}}},
	}
	manifest, err := catalog.ReplaceCatalog(ctx, policies, []artifactcohort.Cohort{
		{Tenant: "team", Name: "operators", PolicyResource: policyResource, Roots: []artifactcohort.RootRef{{URI: root.URI, Kind: root.Kind, Version: 0}, {URI: intent.Root.URI, Kind: intent.Root.Kind, Version: 0}}},
		{Tenant: "team", Name: "pinned", PolicyResource: pinnedPolicy, Roots: []artifactcohort.RootRef{{URI: root.URI, Kind: root.Kind, Version: 1}}},
	}, gen)
	if err != nil {
		t.Fatalf("catalog publish: %v", err)
	}
	windowInManifest, windowFound := artifactcohort.FindRoot(manifest.Cohorts[0].Roots, first.Root.URI)
	intentInManifest, intentFound := artifactcohort.FindRoot(manifest.Cohorts[0].Roots, intent.Root.URI)
	if !windowFound || !intentFound || windowInManifest != first.Root || intentInManifest != intent.Root {
		t.Fatalf("catalog publish: %v", err)
	}
	if len(manifest.Cohorts[0].FollowPublished) != 2 || len(manifest.Cohorts[1].FollowPublished) != 0 {
		t.Fatal("follow-published selection was not preserved")
	}
	serviceToken := identity.token(t, identity.private, "agently-service", []string{"operator"}, time.Now().Add(time.Minute))
	userToken := identity.token(t, identity.private, "browser-user", []string{"operator"}, time.Now().Add(time.Minute))
	provider := &forgeservice.PrimitiveProvider{Host: forgeservice.PrimitiveHostFuncs{CatalogFunc: func(context.Context, *windowprotocol.CatalogInput) (*windowprotocol.Catalog, error) {
		return &windowprotocol.Catalog{ContractVersion: windowprotocol.Version, CatalogRevision: "fixture", Windows: []windowprotocol.WindowSummary{}}, nil
	}}, Authority: forgeservice.PrimitiveAuthorityFuncs{AuthenticateFunc: func(context.Context) (string, error) { return "fixture", nil }, AuthorizeFunc: func(context.Context, string, string, string, string) error { return nil }}}
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: identity.provider}
		config.ForgeProvider = provider
		config.ArtifactCohorts = catalog
		config.AuthorizeArtifactHost = func(ctx context.Context) error {
			principal, err := identity.provider.ResolvePrincipal(ctx)
			if err != nil || principal.Facts.Subject != "agently-service" {
				return errors.New("configured MCP service identity required")
			}
			return nil
		}
	}, verify: func(_ *Service, address string) {
		var token atomic.Value
		token.Store(serviceToken)
		notificationHandler := artifactcohort.NewNotificationHandler(ctx, nil)
		defer notificationHandler.Close()
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHandler(mcpclient.NewHandler(notificationHandler)), streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var m struct {
				Method string
				Params struct{ Name string }
			}
			if err := json.Unmarshal(body, &m); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, m.Method)
			if m.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, m.Params.Name)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		client := mcpclient.New("artifact-host", "1", transport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer client.Close()
		initialized, err := client.Initialize(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if initialized.Capabilities.Extensions[artifactcohort.Capability] == nil {
			t.Fatal("cohort capability not negotiated")
		}
		subscriberCount := func() int { catalog.mu.Lock(); defer catalog.mu.Unlock(); return len(catalog.sinks) }
		if got := subscriberCount(); got != 0 {
			t.Fatalf("MCP initialization subscribed before host admission: %d", got)
		}
		tools, err := client.ListTools(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		names := []string{}
		for _, tool := range tools.Tools {
			names = append(names, tool.Name)
		}
		sort.Strings(names)
		for _, name := range []string{artifactcohort.MethodDiscover, artifactcohort.MethodFetch} {
			if !sort.StringsAreSorted(names) || sort.SearchStrings(names, name) == len(names) || names[sort.SearchStrings(names, name)] != name {
				t.Fatalf("missing tool %s", name)
			}
		}
		call := func(name string, args map[string]any) (*schema.CallToolResult, error) {
			return client.CallTool(ctx, &schema.CallToolRequestParams{Name: name, Arguments: args}, mcpclient.WithNoRetry())
		}
		seen, err := call(artifactcohort.MethodDiscover, map[string]any{})
		if err != nil || seen.IsError != nil && *seen.IsError {
			t.Fatalf("discover: %v %+v", err, seen)
		}
		if got := subscriberCount(); got != 1 {
			t.Fatalf("successful host admission did not subscribe exactly once: %d", got)
		}
		var got artifactcohort.Manifest
		raw, _ := json.Marshal(seen.StructuredContent)
		if json.Unmarshal(raw, &got) != nil || artifactcohort.ValidateManifest(got) != nil || got.Hash != manifest.Hash {
			t.Fatal("invalid manifest")
		}
		mcpText := func(result *schema.CallToolResult) string {
			if result == nil || len(result.Content) != 1 {
				return ""
			}
			switch content := result.Content[0].(type) {
			case schema.TextContent:
				return content.Text
			case map[string]interface{}:
				text, _ := content["text"].(string)
				return text
			}
			return ""
		}
		unchanged, err := call(artifactcohort.MethodDiscover, map[string]any{"knownHash": got.Hash})
		if err != nil || unchanged.IsError != nil && *unchanged.IsError {
			t.Fatalf("conditional discover: %v", err)
		}
		var compact artifactcohort.Manifest
		if json.Unmarshal([]byte(mcpText(unchanged)), &compact) != nil || !compact.Unchanged || compact.Hash != got.Hash || compact.Generation != 0 || len(compact.Policies) != 0 || len(compact.Cohorts) != 0 || len(compact.PartitionHashes) != 0 || len(mcpText(unchanged)) >= len(mcpText(seen)) {
			t.Fatal("matching hash did not return compact unchanged manifest")
		}
		t.Logf("artifact discover full=%d compact-unchanged=%d bytes", len(mcpText(seen)), len(mcpText(unchanged)))
		caller := artifactcohort.MCPCaller{Client: client}
		callerFull, err := caller.Discover(ctx)
		if err != nil || callerFull.Hash != got.Hash {
			t.Fatalf("shared Forge caller full discover: %v", err)
		}
		callerSame, err := caller.DiscoverKnown(ctx, callerFull.Hash)
		if err != nil || !callerSame.Unchanged || callerSame.Hash != got.Hash {
			t.Fatalf("shared Forge caller conditional discover: %v", err)
		}
		stale, err := call(artifactcohort.MethodDiscover, map[string]any{"knownHash": artifactcohort.Hash([]byte("stale"))})
		if err != nil || stale.IsError != nil && *stale.IsError {
			t.Fatalf("stale-hash discover: %v", err)
		}
		var full artifactcohort.Manifest
		if json.Unmarshal([]byte(mcpText(stale)), &full) != nil || full.Unchanged || artifactcohort.ValidateManifest(full) != nil {
			t.Fatal("stale hash did not return full manifest")
		}
		for _, invalid := range []map[string]any{{"unknown": "value"}, {"knownHash": "invalid"}, {"knownHash": 7}} {
			denial, err := call(artifactcohort.MethodDiscover, invalid)
			if err != nil || denial.IsError == nil || !*denial.IsError {
				t.Fatalf("invalid conditional argument accepted: %v", invalid)
			}
		}
		encodedRoot, _ := json.Marshal(first.Root)
		var rootArgs map[string]any
		_ = json.Unmarshal(encodedRoot, &rootArgs)
		fetched, err := call(artifactcohort.MethodFetch, map[string]any{"tenant": "team", "root": rootArgs})
		if err != nil || fetched.IsError != nil && *fetched.IsError {
			t.Fatalf("fetch: %v %+v", err, fetched)
		}
		var artifact artifactcohort.Artifact
		var textBody string
		switch content := fetched.Content[0].(type) {
		case schema.TextContent:
			textBody = content.Text
		case map[string]interface{}:
			textBody, _ = content["text"].(string)
		}
		decodeErr := json.Unmarshal([]byte(textBody), &artifact)
		validationErr := artifactcohort.ValidateArtifact(&artifact)
		if textBody == "" || decodeErr != nil || validationErr != nil || artifact.Root != first.Root || string(artifact.Payload) != string(first.Payload) {
			t.Fatalf("invalid fetched artifact: contentType=%T decode=%v validate=%v sameRoot=%v samePayload=%v originalHash=%s receivedHash=%s", fetched.Content[0], decodeErr, validationErr, artifact.Root == first.Root, string(artifact.Payload) == string(first.Payload), artifactcohort.Hash(first.Payload), artifactcohort.Hash(artifact.Payload))
		}
		callerArtifact, err := caller.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: first.Root})
		if err != nil || callerArtifact.Root != first.Root || string(callerArtifact.Payload) != string(first.Payload) {
			t.Fatalf("shared Forge caller exact fetch: %v", err)
		}
		counted := &countedCohortToolClient{next: client, calls: map[string]int{}, fetchByURI: map[string]int{}}
		replica, err := artifactcohort.NewReplica(artifactcohort.Config{Enabled: true, Source: "datly-mcp", Tenant: "team", Account: "21", Caller: artifactcohort.MCPCaller{Client: counted}, Facts: identity.provider, SessionKey: func(ctx context.Context) (string, error) {
			session, _ := ctx.Value(cohortTestSessionKey{}).(string)
			if session == "" {
				return "", artifactcohort.ErrDenied
			}
			return session, nil
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err := notificationHandler.AttachReplica(replica); err != nil {
			t.Fatal(err)
		}
		if err := replica.Sync(ctx); err != nil {
			t.Fatalf("real MCP replica startup sync: %v", err)
		}
		if counted.count(artifactcohort.MethodDiscover) != 1 || counted.count(artifactcohort.MethodFetch) != 2 || counted.fetchCount(first.Root.URI) != 1 || counted.fetchCount(intent.Root.URI) != 1 {
			t.Fatalf("unexpected startup calls discover=%d fetch=%d", counted.count(artifactcohort.MethodDiscover), counted.count(artifactcohort.MethodFetch))
		}
		users := make([]context.Context, 100)
		openTimes := make([]time.Duration, 100)
		for i := range users {
			session := fmt.Sprintf("user-%03d", i)
			verified := oauth.WithBearer(ctx, identity.token(t, identity.private, session, []string{"operator"}, time.Now().Add(time.Minute)))
			users[i] = context.WithValue(verified, cohortTestSessionKey{}, session)
			if err := replica.Admit(users[i]); err != nil {
				t.Fatalf("replica admit user %d: %v", i, err)
			}
			started := time.Now()
			view, err := replica.Open(users[i], first.Root.URI)
			openTimes[i] = time.Since(started)
			if err != nil || view.Ref() != first.Root || string(view.Payload()) != string(first.Payload) {
				t.Fatalf("replica local open user %d: %v", i, err)
			}
		}
		pinnedUser := context.WithValue(oauth.WithBearer(ctx, identity.token(t, identity.private, "pinned-user", []string{"pinner"}, time.Now().Add(time.Minute))), cohortTestSessionKey{}, "pinned-user")
		if err := replica.Admit(pinnedUser); err != nil {
			t.Fatalf("pinning cohort admission: %v", err)
		}
		if view, err := replica.Open(pinnedUser, first.Root.URI); err != nil || view.Ref() != first.Root {
			t.Fatalf("pinned cohort did not open version 1: %v", err)
		}
		sort.Slice(openTimes, func(i, j int) bool { return openTimes[i] < openTimes[j] })
		if counted.count(artifactcohort.MethodDiscover) != 1 || counted.count(artifactcohort.MethodFetch) != 2 {
			t.Fatal("100 local admitted Opens contacted MCP")
		}
		t.Logf("real-MCP replica 100 verified-user local Open p50=%s max=%s with zero provider calls", openTimes[50], openTimes[99])
		if err := replica.Sync(ctx); err != nil {
			t.Fatalf("same-hash replica sync: %v", err)
		}
		if counted.count(artifactcohort.MethodDiscover) != 2 || counted.count(artifactcohort.MethodFetch) != 2 {
			t.Fatal("same-hash replica sync fetched artifact bytes")
		}
		var browserToken atomic.Value
		browserToken.Store(userToken)
		browserTransport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &browserToken}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var message struct {
				Method string
				Params struct{ Name string }
			}
			if err := json.Unmarshal(body, &message); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, message.Method)
			if message.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, message.Params.Name)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		browser := mcpclient.New("ordinary-browser", "1", browserTransport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer browser.Close()
		if _, err := browser.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		ordinary, err := browser.CallTool(ctx, &schema.CallToolRequestParams{Name: artifactcohort.MethodDiscover, Arguments: map[string]any{}}, mcpclient.WithNoRetry())
		if err != nil || ordinary.IsError == nil || !*ordinary.IsError || subscriberCount() != 1 {
			t.Fatalf("ordinary browser subscribed or discovered: error=%v subscribers=%d", err, subscriberCount())
		}
		token.Store(userToken)
		denied, err := call(artifactcohort.MethodDiscover, map[string]any{})
		if err != nil || denied.IsError == nil || !*denied.IsError {
			t.Fatal("ordinary browser principal received cohort catalog")
		}
		token.Store(serviceToken)
		recorder := &cohortNotificationRecorder{}
		stop := catalog.SubscribeResources(recorder)
		defer stop()
		second, nextGen, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: root.URI, Kind: root.Kind, Version: 2}, closedWindowForCohort(t, "second"), validate)
		if err != nil || nextGen <= manifest.Generation {
			t.Fatalf("publish second: %v", err)
		}
		historicalFetch, err := call(artifactcohort.MethodFetch, map[string]any{"tenant": "team", "root": rootArgs})
		if err != nil || historicalFetch.IsError != nil && *historicalFetch.IsError {
			t.Fatalf("pinned historical root could not be fetched: %v", err)
		}
		var historical artifactcohort.Artifact
		if json.Unmarshal([]byte(mcpText(historicalFetch)), &historical) != nil || artifactcohort.ValidateArtifact(&historical) != nil || historical.Root != first.Root || string(historical.Payload) != string(first.Payload) {
			t.Fatal("historical exact-byte MCP fetch changed after newer publication")
		}
		if _, _, err = catalog.Publish(ctx, artifactcohort.RootRef{URI: root.URI, Kind: root.Kind, Version: 2}, closedWindowForCohort(t, "changed-same-version"), validate); !errors.Is(err, artifactcohort.ErrConflict) {
			t.Fatalf("same-version changed bytes accepted: %v", err)
		}
		if current, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: second.Root}); err != nil || current.Root != second.Root {
			t.Fatalf("failed candidate displaced prior version: %v", err)
		}
		events := recorder.snapshot()
		var update struct {
			URI string `json:"uri"`
		}
		if len(events) != 1 || events[0].Method != artifactcohort.NotificationUpdated || json.Unmarshal(events[0].Params, &update) != nil || update.URI != second.Root.URI {
			t.Fatalf("missing standard change notifications: %v", events)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			updated, openErr := replica.Open(users[0], first.Root.URI)
			if openErr == nil && updated.Ref() == second.Root {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("stateful MCP notification did not activate v2: %v", openErr)
			}
			time.Sleep(5 * time.Millisecond)
		}
		if counted.fetchCount(first.Root.URI) != 2 || counted.fetchCount(intent.Root.URI) != 1 {
			t.Fatalf("unrelated intent was refetched: window=%d intent=%d", counted.fetchCount(first.Root.URI), counted.fetchCount(intent.Root.URI))
		}
		updatedView, err := replica.Open(users[0], first.Root.URI)
		if err != nil || updatedView.Ref() != second.Root || string(updatedView.Payload()) != string(second.Payload) {
			t.Fatalf("updated local Open did not switch to v2: %v", err)
		}
		pinnedView, err := replica.Open(pinnedUser, first.Root.URI)
		if err != nil || pinnedView.Ref() != first.Root || string(pinnedView.Payload()) != string(first.Payload) {
			t.Fatalf("pinned cohort did not retain v1 when follow cohort advanced: %v", err)
		}
		if _, err := catalog.ReplaceCatalog(ctx, policies, manifest.Cohorts, catalog.Publisher.Generation()); err != nil {
			t.Fatalf("catalog replacement: %v", err)
		}
		events = recorder.snapshot()
		if len(events) != 2 || events[1].Method != artifactcohort.NotificationListChanged {
			t.Fatalf("catalog replacement did not emit list_changed: %v", events)
		}
		deadline = time.Now().Add(2 * time.Second)
		for counted.count(artifactcohort.MethodDiscover) < 4 {
			if time.Now().After(deadline) {
				t.Fatal("stateful MCP list_changed did not trigger conditional discover")
			}
			time.Sleep(5 * time.Millisecond)
		}
		if counted.fetchCount(first.Root.URI) != 2 || counted.fetchCount(intent.Root.URI) != 1 {
			t.Fatal("unchanged list sync refetched root bytes")
		}
		if err := catalog.RemoveRoot(ctx, "window://team/unknown"); err != nil || len(recorder.snapshot()) != 2 {
			t.Fatalf("unknown root removal changed catalog: %v", err)
		}
		beforeRemovalDiscover := counted.count(artifactcohort.MethodDiscover)
		if err := catalog.RemoveRoot(ctx, second.Root.URI); err != nil {
			t.Fatalf("remove active window: %v", err)
		}
		events = recorder.snapshot()
		if len(events) != 3 || events[2].Method != artifactcohort.NotificationUpdated || json.Unmarshal(events[2].Params, &update) != nil || update.URI != second.Root.URI {
			t.Fatalf("root removal did not emit exact updated URI: %v", events)
		}
		deadline = time.Now().Add(2 * time.Second)
		for {
			_, openErr := replica.Open(users[0], second.Root.URI)
			if errors.Is(openErr, artifactcohort.ErrDenied) && counted.count(artifactcohort.MethodDiscover) > beforeRemovalDiscover {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("removed window remained open after notification: %v", openErr)
			}
			time.Sleep(5 * time.Millisecond)
		}
		if _, err := replica.Open(pinnedUser, first.Root.URI); !errors.Is(err, artifactcohort.ErrDenied) {
			t.Fatalf("removed URI remained open through pinned cohort: %v", err)
		}
		if _, err := replica.Open(users[0], intent.Root.URI); err != nil {
			t.Fatalf("unrelated intent blocked by window removal: %v", err)
		}
		if counted.fetchCount(first.Root.URI) != 2 || counted.fetchCount(intent.Root.URI) != 1 {
			t.Fatal("root removal fetched unchanged artifact bytes")
		}
		if _, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: second.Root}); !errors.Is(err, artifactcohort.ErrDenied) {
			t.Fatalf("removed bytes remained available through producer fetch: %v", err)
		}
		if _, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: first.Root}); !errors.Is(err, artifactcohort.ErrDenied) {
			t.Fatalf("removed pinned revision remained available through producer fetch: %v", err)
		}
		if err := catalog.RemoveRoot(ctx, intent.Root.URI); err != nil {
			t.Fatalf("remove final cohort root: %v", err)
		}
		emptyRoots, err := catalog.Discover(ctx)
		if err != nil || artifactcohort.ValidateManifest(emptyRoots) != nil || len(emptyRoots.Cohorts) != 2 || len(emptyRoots.Cohorts[0].Roots) != 0 || len(emptyRoots.Cohorts[0].FollowPublished) != 0 {
			t.Fatalf("empty cohort root set invalid: %v", err)
		}
		if _, err := catalog.ReplaceCatalog(ctx, nil, nil, catalog.Publisher.Generation()); err != nil {
			t.Fatalf("empty catalog replacement: %v", err)
		}
		emptyCatalog, err := catalog.Discover(ctx)
		if err != nil || artifactcohort.ValidateManifest(emptyCatalog) != nil || len(emptyCatalog.Cohorts) != 0 {
			t.Fatalf("empty catalog invalid: %v", err)
		}
	}})
}

func TestPublishedArtifactCatalogNotifiesOnlyAdvertisedRootChanges(t *testing.T) {
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	root := artifactcohort.RootRef{URI: "window://team/pinned-only", Kind: "window", Version: 1}
	validate := func(_ context.Context, raw json.RawMessage) error {
		var window types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &window); err != nil {
			return err
		}
		return window.Validate()
	}
	first, generation, err := catalog.Publish(ctx, root, closedWindowForCohort(t, "first"), validate)
	if err != nil {
		t.Fatal(err)
	}
	policyRef := authz.Resource{Tenant: "team", Kind: "cohort", ID: "team/pinned", Version: "logical"}
	policies := []authz.Document{{Resource: policyRef, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "pinner"}}}}}
	manifest, err := catalog.ReplaceCatalog(ctx, policies, []artifactcohort.Cohort{{Tenant: "team", Name: "pinned", PolicyResource: policyRef, Roots: []artifactcohort.RootRef{first.Root}}}, generation)
	if err != nil {
		t.Fatal(err)
	}
	recorder := &cohortNotificationRecorder{}
	stop := catalog.SubscribeResources(recorder)
	defer stop()
	second, nextGeneration, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: root.URI, Kind: root.Kind, Version: 2}, closedWindowForCohort(t, "second"), validate)
	if err != nil || nextGeneration <= manifest.Generation {
		t.Fatalf("retain unselected revision: %v", err)
	}
	unchanged, err := catalog.Discover(ctx)
	if err != nil || unchanged.Hash != manifest.Hash || len(recorder.snapshot()) != 0 {
		t.Fatalf("unselected revision changed advertised catalog or emitted notification: %v", err)
	}
	if _, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: second.Root}); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("unselected revision became fetchable: %v", err)
	}
	if _, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: first.Root}); err != nil {
		t.Fatalf("pinned historical revision was lost: %v", err)
	}
	follow, err := catalog.ReplaceCatalog(ctx, policies, []artifactcohort.Cohort{{Tenant: "team", Name: "pinned", PolicyResource: policyRef, Roots: []artifactcohort.RootRef{{URI: root.URI, Kind: root.Kind}}}}, nextGeneration)
	if err != nil || len(follow.Cohorts[0].FollowPublished) != 1 {
		t.Fatalf("select follow-published root: %v", err)
	}
	_, _, err = catalog.Publish(ctx, artifactcohort.RootRef{URI: root.URI, Kind: root.Kind, Version: 3}, closedWindowForCohort(t, "third"), validate)
	if err != nil {
		t.Fatal(err)
	}
	events := recorder.snapshot()
	if len(events) != 2 || events[0].Method != artifactcohort.NotificationListChanged || events[1].Method != artifactcohort.NotificationUpdated {
		t.Fatalf("follow advance did not emit one exact-root notification: %v", events)
	}
}

func TestArtifactManifestScopeRejectsOtherNamespaceAndUnrelatedPolicy(t *testing.T) {
	policyRef := authz.Resource{Tenant: "team", Kind: "cohort", ID: "team/operators", Version: "logical"}
	policy := authz.Document{Resource: policyRef, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}}}}
	publisher := artifactcohort.NewPublisher()
	root := artifactcohort.RootRef{URI: "window://team/one", Kind: "window", Version: 1}
	artifact, generation, err := publisher.Publish(context.Background(), root, closedWindowForCohort(t, "scope"), func(_ context.Context, raw json.RawMessage) error {
		var value types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		return value.Validate()
	})
	if err != nil {
		t.Fatal(err)
	}
	root = artifact.Root
	cohort := artifactcohort.Cohort{Tenant: "team", Name: "operators", PolicyResource: policyRef, Roots: []artifactcohort.RootRef{{URI: root.URI, Kind: root.Kind}}}
	manifest, err := publisher.ReplaceCatalog(context.Background(), []authz.Document{policy}, []artifactcohort.Cohort{cohort}, generation)
	if err != nil {
		t.Fatal(err)
	}
	if err := artifactcohort.ValidateManifest(manifest); err != nil {
		t.Fatal(err)
	}
	if err := validateArtifactManifestScope(context.Background(), manifest, "team"); err != nil {
		t.Fatalf("owned namespace denied: %v", err)
	}
	foreign := manifest
	foreign.Cohorts = append([]artifactcohort.Cohort(nil), manifest.Cohorts...)
	foreign.Cohorts[0].Roots = []artifactcohort.RootRef{{URI: "window://other/one", Kind: "window", Version: 1, Hash: root.Hash}}
	if err := validateArtifactManifestScope(context.Background(), foreign, "team"); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("foreign namespace leaked: %v", err)
	}
	foreignOverride := manifest
	foreignOverride.Cohorts = append([]artifactcohort.Cohort(nil), manifest.Cohorts...)
	foreignOverride.Cohorts[0].RevisionOverrides = []artifactcohort.RootRef{{URI: "window://other/one", Kind: "window", Version: 1, Hash: root.Hash}}
	if err := validateArtifactManifestScope(context.Background(), foreignOverride, "team"); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("foreign override namespace leaked: %v", err)
	}
	unrelated := manifest
	unrelated.Policies = append(append([]authz.Document(nil), manifest.Policies...), authz.Document{Resource: authz.Resource{Tenant: "team", Kind: "cohort", ID: "other/operators", Version: "logical"}, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}}}})
	if err := validateArtifactManifestScope(context.Background(), unrelated, "team"); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("unrelated policy leaked: %v", err)
	}
}

func TestAuthenticatedMCPFetchOverrideOnlyRetainedRevision(t *testing.T) {
	identity := newStockOAuthFixture(t)
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	uri := "window://team/preview-only"
	validate := func(_ context.Context, raw json.RawMessage) error {
		var value types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		return value.Validate()
	}
	first, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 1}, closedWindowForCohort(t, "first"), validate)
	if err != nil {
		t.Fatal(err)
	}
	second, generation, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 2}, closedWindowForCohort(t, "second"), validate)
	if err != nil {
		t.Fatal(err)
	}
	policyRef := authz.Resource{Tenant: "team", Kind: "cohort", ID: "team/preview", Version: "logical"}
	policy := authz.Document{Resource: policyRef, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "agently-service"}}}}
	if _, err := catalog.ReplaceCatalog(ctx, []authz.Document{policy}, []artifactcohort.Cohort{{Tenant: "team", Name: "preview", PolicyResource: policyRef, RevisionOverrides: []artifactcohort.RootRef{{URI: uri, Kind: "window", Version: 1}}}}, generation); err != nil {
		t.Fatal(err)
	}
	serviceToken := identity.token(t, identity.private, "agently-service", nil, time.Now().Add(time.Minute))
	provider := &forgeservice.PrimitiveProvider{Host: forgeservice.PrimitiveHostFuncs{CatalogFunc: func(context.Context, *windowprotocol.CatalogInput) (*windowprotocol.Catalog, error) {
		return &windowprotocol.Catalog{ContractVersion: windowprotocol.Version, CatalogRevision: "fixture", Windows: []windowprotocol.WindowSummary{}}, nil
	}}, Authority: forgeservice.PrimitiveAuthorityFuncs{AuthenticateFunc: func(context.Context) (string, error) { return "fixture", nil }, AuthorizeFunc: func(context.Context, string, string, string, string) error { return nil }}}
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: identity.provider}
		config.ForgeProvider = provider
		config.ArtifactCohorts = catalog
		config.AuthorizeArtifactHost = func(ctx context.Context) error {
			principal, err := identity.provider.ResolvePrincipal(ctx)
			if err != nil || principal.Facts.Subject != "agently-service" {
				return artifactcohort.ErrDenied
			}
			return nil
		}
	}, verify: func(_ *Service, address string) {
		var token atomic.Value
		token.Store(serviceToken)
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var message struct {
				Method string
				Params struct{ Name string }
			}
			if err := json.Unmarshal(body, &message); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, message.Method)
			if message.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, message.Params.Name)
			}
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		client := mcpclient.New("override-host", "1", transport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
		defer client.Close()
		if _, err := client.Initialize(ctx); err != nil {
			t.Fatal(err)
		}
		caller := artifactcohort.MCPCaller{Client: client}
		if _, err := caller.Discover(ctx); err != nil {
			t.Fatalf("override-only manifest was not discoverable: %v", err)
		}
		fetched, err := caller.Fetch(ctx, artifactcohort.FetchInput{Tenant: "team", Root: first.Root})
		if err != nil || fetched.Root != first.Root || string(fetched.Payload) != string(first.Payload) {
			t.Fatalf("MCP host could not stage exact override-only bytes: %+v %v", fetched, err)
		}
		badHash := first.Root
		badHash.Hash = second.Root.Hash
		for _, input := range []artifactcohort.FetchInput{{Tenant: "team", Root: second.Root}, {Tenant: "team", Root: badHash}, {Tenant: "other", Root: first.Root}} {
			if _, err := caller.Fetch(ctx, input); err == nil {
				t.Fatalf("undeclared MCP root became fetchable: %+v", input)
			}
		}
	}})
}
