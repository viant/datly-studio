package namespacevisibility

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/accessconfig"
	"github.com/viant/datly-studio/runtime/accessprovider"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

// Roles are resolved from a server-configured verifier, never request selectors.
func Criteria(ctx context.Context, connectors connector.Provider, subject, alias string) (*xpredicate.Criteria, error) {
	if subject == "" {
		return nil, forbidden("namespace subject is required")
	}
	if alias != "n" && alias != "namespaces" {
		return nil, fmt.Errorf("invalid namespace SQL alias")
	}
	if IsAdministrator(ctx, subject) {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	expression := "(namespaces.owner_id = ? OR namespaces.visibility = 'public'"
	values := []any{subject}
	provider, err := accessprovider.FromEnvironment(ctx)
	if err != nil {
		return nil, forbidden("namespace role verifier is unavailable")
	}
	if provider != nil {
		roles, err := verifiedRoles(ctx, subject)
		if err != nil {
			// A user-info outage must not hide a namespace from its verified
			// owner or from authenticated viewers of a public namespace. It
			// does fail closed for role-based visibility.
			identity, identityErr := accessconfig.IdentityFromEnvironment()
			if identityErr != nil || identity == nil {
				return nil, forbidden("namespace role identity is invalid")
			}
			verified, verifyErr := identity.Resolve(ctx)
			if verifyErr != nil || verified.Subject != subject || verified.Issuer == "" || !verified.ValidUntil.After(time.Now()) {
				return nil, forbidden("namespace role identity is invalid")
			}
			return &xpredicate.Criteria{Expression: strings.ReplaceAll(expression+")", "namespaces.", alias+"."), Placeholders: values}, nil
		}
		roleScope, err := roleCriteria(connectors, ctx, "namespaces", roles, false)
		if err != nil {
			return nil, err
		}
		expression += " OR " + roleScope.Expression
		values = append(values, roleScope.Placeholders...)
	}
	return &xpredicate.Criteria{Expression: strings.ReplaceAll(expression+")", "namespaces.", alias+"."), Placeholders: values}, nil
}

func forbidden(message string) error {
	return &xresponse.Error{Code: 403, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
