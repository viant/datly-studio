package host

import (
	"context"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	"net/http"
	"strings"
	"testing"
)

func TestNamespaceDefaultPolicyHTTPAndMCP(t *testing.T) {
	host, err := newScopedHost(t, map[string]string{"tasks": scopedTasksDQL, "records": unscopedRecordsDQL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err = host.service.studio.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES('ns','owner','test','Test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = host.service.studio.Exec(`UPDATE components SET namespace_id='ns'`); err != nil {
		t.Fatal(err)
	}
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	document, err := host.store.Provision(ctx, authz.Document{Resource: namespacepolicy.Resource("ns", scopeTestTenant), Policies: map[string]authz.Policy{"discover": policy, "describe": policy, "execute": policy}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	token := host.token(t, "alice", []authz.Entity{{Type: "project", ID: "101"}})
	status, body := host.get(t, "/tasks", token, nil)
	if status != http.StatusOK {
		t.Fatalf("inherited HTTP: %d %s", status, body)
	}
	assertRows(t, body, []string{"alpha-101"}, []string{"beta-102", "gamma-103", "delta-104"})
	status, body = host.callTool(t, token, "tasks.list", nil)
	if status != http.StatusOK || strings.Contains(body, `"isError":true`) {
		t.Fatalf("inherited MCP: %d %s", status, body)
	}
	assertRows(t, body, []string{"alpha-101"}, []string{"beta-102", "gamma-103", "delta-104"})
	status, _ = host.get(t, "/records", token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("inherited bounded policy executed without scope binding: %d", status)
	}
	// Removing entity bounds updates every sibling without rebuilding the host.
	document.Policies["execute"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	document, err = host.store.Replace(ctx, document, document.Revision, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	status, body = host.get(t, "/records", token, nil)
	if status != 200 {
		t.Fatalf("sibling did not pick up default: %d %s", status, body)
	}
	_, err = host.store.Provision(ctx, authz.Document{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Policies: map[string]authz.Policy{"execute": {Mode: "deny"}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	status, _ = host.get(t, "/records", token, nil)
	if status != 403 {
		t.Fatalf("namespace bypassed tool deny: %d", status)
	}
	document.Policies["execute"] = authz.Policy{Mode: "deny"}
	_, err = host.store.Replace(ctx, document, document.Revision, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	status, _ = host.get(t, "/tasks", token, nil)
	if status != 403 {
		t.Fatalf("namespace revocation ignored: %d", status)
	}
}
