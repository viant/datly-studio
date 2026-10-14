package authorization_test

import (
	"context"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	"testing"
)

func TestRuntimeStatusForVisibleNamespaceWithoutComponents(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "empty_namespace_status", "studio")
	ns := namespaceaccess.ID("owner", "skills")
	if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','skills','Skills','active','private',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, ns); err != nil {
		t.Fatal(err)
	}
	transport := &sqltransport.Transport{DB: db, Authorizer: &authorization.SDKAuthorizer{DB: db}, RuntimeProbe: sqltransport.RuntimeHostProbeFunc(func(ctx context.Context) (*sdk.RuntimeHost, error) {
		id, _ := sdk.NamespaceSelectionFromContext(ctx)
		return &sdk.RuntimeHost{NamespaceID: id, MCPURL: "http://127.0.0.1:12345/mcp", Status: "ready"}, nil
	})}
	defer transport.Close(ctx)
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	status, err := client.Runtime().Status(sdk.WithNamespaceSelection(owner, ns))
	if err != nil || status == nil || status.Host == nil || status.Host.Status != "ready" || status.ReportCount != 0 {
		t.Fatalf("empty namespace status: %+v %v", status, err)
	}
	outsider := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "outsider"})
	if _, err = client.Runtime().Status(sdk.WithNamespaceSelection(outsider, ns)); err == nil {
		t.Fatal("private namespace leaked to outsider")
	}
	if _, err = client.Runtime().Status(owner); err == nil {
		t.Fatal("unscoped runtime status bypassed global authorization")
	}
}
