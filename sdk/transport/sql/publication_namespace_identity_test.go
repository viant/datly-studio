package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type cubeScopeIdentity struct{}

func (cubeScopeIdentity) Resolve(ctx context.Context) (authz.Facts, error) {
	if oauth.Bearer(ctx) != "verified-credential" {
		return authz.Facts{}, errors.New("verified namespace bearer is required")
	}
	return authz.Facts{Subject: "owner", Issuer: "test", Tenant: "studio", Roles: []string{"READER"}, ValidUntil: time.Now().Add(time.Hour)}, nil
}
func TestPublicationNamespaceScopeWithVerifiedIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 64)
	if _, err = db.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,visibility,allowed_roles_json,mcp_enabled,mcp_port,etag,created_at,updated_at) VALUES(?,'test','Test','owner','active','private','[]',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id); err != nil {
		t.Fatal(err)
	}
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	ctx = oauth.WithBearer(ctx, "verified-credential")
	ctx = accessprovider.WithProvider(ctx, cubeScopeIdentity{})
	ctx = sdk.WithNamespaceSelection(ctx, id)
	transport := &Transport{DB: db}
	if _, err = transport.resolveNamespace(ctx); err != nil {
		if e, ok := err.(*sdk.Error); ok {
			t.Fatalf("namespace cause: %T %v", e.Cause, e.Cause)
		}
		t.Fatal(err)
	}
}
