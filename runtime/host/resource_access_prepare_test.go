package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	"github.com/viant/mcp-protocol/schema"
)

type preparationIdentity struct{ calls atomic.Int64 }

func (p *preparationIdentity) Resolve(context.Context) (authz.Facts, error) {
	p.calls.Add(1)
	return authz.Facts{Subject: "fixture-reader", Issuer: "fixture", Tenant: scopeTestTenant, Roles: []string{"reader"}, ValidUntil: time.Now().Add(time.Minute)}, nil
}

func TestPublishedPolicyRuntimePreparedWithoutIdentityOrRetainedPolicies(t *testing.T) {
	provider := &preparationIdentity{}
	dql := strings.Replace(unscopedRecordsDQL, "#define($_ = $Records", "#setting($_ = $mcp('records.list','List records'))\n#define($_ = $Records", 1)
	fixture, err := newScopedHostConfigured(t, map[string]string{"records": dql}, nil, func(c *ResourceAccessConfig) { c.Provider = provider })
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls.Load() != 0 {
		t.Fatal("publication resolved request identity")
	}
	// The exact pinned Authz Store lazily owns its runtime. It must now already
	// exist before any client request, even when the published policy is absent.
	if reflect.ValueOf(fixture.store).Elem().FieldByName("runtime").IsNil() {
		t.Fatal("native policy reader still compiles on the first request")
	}
	ctx := t.Context()
	token := fixture.token(t, "fixture-reader", nil)
	nativeRuntime := reflect.ValueOf(fixture.store).Elem().FieldByName("runtime").Pointer()
	list := func() string {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": schema.MethodToolsList, "params": map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": schema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+fixture.mcpAddr+"/mcp", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set(schema.HeaderProtocolVersion, schema.LatestProtocolVersion)
		request.Header.Set(schema.HeaderMethod, schema.MethodToolsList)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("native list: %d %s", response.StatusCode, body)
		}
		if reflect.ValueOf(fixture.store).Elem().FieldByName("runtime").Pointer() != nativeRuntime {
			t.Fatal("listing replaced prepared native policy runtime")
		}
		return string(body)
	}
	if body := list(); !strings.Contains(body, `"name":"records.list"`) {
		t.Fatalf("first native list missing published reader: %s", body)
	}
	resource := authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}
	doc, err := fixture.store.Provision(ctx, authz.Document{Resource: resource, Policies: map[string]authz.Policy{"execute": {Mode: "deny"}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if body := list(); strings.Contains(body, `"name":"records.list"`) {
		t.Fatalf("new policy omitted discovery but leaked metadata: %s", body)
	}
	status, _ := fixture.get(t, "/records", token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("policy created after bootstrap was ignored: %d", status)
	}
	doc.Policies["execute"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	doc, err = fixture.store.Replace(ctx, doc, doc.Revision, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	status, body := fixture.get(t, "/records", token, nil)
	if status != http.StatusOK {
		t.Fatalf("fresh protected policy rejected: %d %s", status, body)
	}
	doc.Policies["execute"] = authz.Policy{Mode: "deny"}
	if _, err = fixture.store.Replace(ctx, doc, doc.Revision, "fixture"); err != nil {
		t.Fatal(err)
	}
	status, _ = fixture.get(t, "/records", token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("prepared runtime retained revoked policy: %d", status)
	}
}

func TestPolicyPreparationDoesNotMakeOptionalPolicyReadFailureFatal(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The owned fixture's missing policy table reproduces an optional authority
	// failure. Publication must retain its prior behavior; requests still fail
	// closed through the ordinary policy authorizer.
	if _, err = fixture.service.studio.ExecContext(t.Context(), "DROP TABLE resource_policies"); err != nil {
		t.Fatal(err)
	}
	if err = fixture.service.Reload(t.Context(), 0); err != nil {
		t.Fatalf("policy preparation added a startup gate: %v", err)
	}
	token := fixture.token(t, "fixture-reader", nil)
	if status, _ := fixture.get(t, "/records", token, nil); status < 400 {
		t.Fatalf("unavailable policy store permitted execution: %d", status)
	}
}

type preparationStore struct {
	authz.Store
	calls int
}

func (s *preparationStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	s.calls++
	return authz.Document{}, authz.ErrDenied
}

func TestPolicyPreparationDoesNotInvokeCustomPolicyStores(t *testing.T) {
	store := &preparationStore{}
	for _, configured := range []authz.Store{store, &namespacepolicy.Store{Store: store}} {
		service := &Service{resourceAccess: &authz.Service{Store: configured}}
		service.prepareResourcePolicyReader(t.Context(), authz.Resource{Kind: "component", ID: "fixture", Version: "1", Tenant: "fixture"})
	}
	if store.calls != 0 {
		t.Fatal("bootstrap invoked a custom policy authority")
	}
}
