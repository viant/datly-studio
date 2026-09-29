package sqltransport

import (
	"context"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk"
	acl "github.com/viant/datly-studio/sdk/access"
	store "github.com/viant/datly-studio/store/sql/accesscatalog"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"time"
)

// The authoring catalog uses the same Studio component visibility as the
// component catalog. It does not supply business roles, exposures or entities.
func (t *Transport) listAccessResources(ctx context.Context, input, output any) error {
	var in acl.CatalogInput
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	policyStore := catalogOnlyStore{Store: &store.Store{DB: t.DB}}
	visible := map[string]bool{}
	service := &acl.Catalog{Service: &authz.Service{Store: policyStore, Provider: catalogSDKPrincipal{}}, AuthoringAccess: func(ctx context.Context, id string) bool {
		if id == "" {
			return false
		}
		if allowed, found := visible[id]; found {
			return allowed
		}
		if _, scoped := selectedNamespace(ctx); scoped {
			rows, err := t.readReportCatalog(ctx, reportCatalogRequest{ID: id, Limit: 2})
			if err != nil || len(rows) != 1 {
				visible[id] = false
				return false
			}
		}
		caps, err := t.reportCapabilities(ctx, id)
		allowed := err == nil && caps.CanView
		visible[id] = allowed
		return allowed
	}}
	if selected, scoped := selectedNamespace(ctx); scoped {
		service.ResourceScope = func(_ context.Context, row *catalog.Entry) bool { return row.NamespaceID == selected.NamespaceID }
	}
	page, err := service.List(ctx, in)
	if err != nil {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource catalog is not permitted", Cause: err}
	}
	return assign(output, &page)
}

type catalogSDKPrincipal struct{}

func (catalogSDKPrincipal) Resolve(ctx context.Context) (authz.Facts, error) {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return authz.Facts{}, authz.ErrDenied
	}
	credential, ok := sdk.VerifiedCredentialFromContext(ctx)
	if !ok {
		return authz.Facts{}, authz.ErrDenied
	}
	claims, ok := credential.Claims.(jwtlib.Claims)
	if !ok {
		return authz.Facts{}, authz.ErrDenied
	}
	subject, err := claims.GetSubject()
	if err != nil || subject != principal.Subject {
		return authz.Facts{}, authz.ErrDenied
	}
	issuer, err := claims.GetIssuer()
	if err != nil || issuer == "" {
		return authz.Facts{}, authz.ErrDenied
	}
	expiry, err := claims.GetExpirationTime()
	if err != nil || expiry == nil || !expiry.After(time.Now()) {
		return authz.Facts{}, authz.ErrDenied
	}
	return authz.Facts{Subject: subject, Issuer: issuer, ValidUntil: expiry.Time}, nil
}

// The authoring catalog must not acquire policy inspection/write capabilities.
// Its visibility callback is the existing Studio component authorization.
type catalogOnlyStore struct{ *store.Store }

func (catalogOnlyStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	return authz.Document{}, authz.ErrDenied
}
func (catalogOnlyStore) Replace(context.Context, authz.Document, int64, string) (authz.Document, error) {
	return authz.Document{}, authz.ErrDenied
}
