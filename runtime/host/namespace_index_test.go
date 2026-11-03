package host

import (
	"context"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/namespacemcp"
)

func TestPreparedNamespaceAuthorizationNeedsNoDatabaseAndSeesUpdates(t *testing.T) {
	index := &namespacemcp.DefinitionIndex{}
	row := namespacemcp.Definition{NamespaceID: "namespace", OwnerID: "owner", Active: true, Visibility: "private", AllowedRoles: []string{"viewer"}}
	index.Replace([]namespacemcp.Definition{row})
	service := &Service{config: Config{NamespaceID: row.NamespaceID, namespaceDefinitions: index}, resourceAccess: &authz.Service{Provider: namespaceFacts{authz.Facts{Subject: "reader", Issuer: "fixture", ValidUntil: time.Now().Add(time.Minute), Roles: []string{"viewer"}}}}}
	ctx := context.Background()
	for n := 0; n < 100; n++ {
		if err := service.authorizeNamespace(ctx); err != nil {
			t.Fatal(err)
		}
	}
	row.AllowedRoles = nil
	index.Replace([]namespacemcp.Definition{row})
	if err := service.authorizeNamespace(ctx); err == nil {
		t.Fatal("revoked viewer remained authorized")
	}
	row.AllowedRoles = []string{"viewer"}
	index.Replace([]namespacemcp.Definition{row})
	if err := service.authorizeNamespace(ctx); err != nil {
		t.Fatal(err)
	}
	index.Replace(nil)
	if err := service.authorizeNamespace(ctx); err == nil {
		t.Fatal("withdrawn namespace remained authorized")
	}
}
