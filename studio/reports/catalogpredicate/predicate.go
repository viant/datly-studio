// Package catalogpredicate owns the server-only report catalog ACL predicate.
// It does not import the SDK transport or the public Studio authorization host.
package catalogpredicate

import (
	"context"
	"errors"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/xdatly/connector"
	"reflect"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type ReportCatalogRead struct {
	Input         any                `bind:"kind=input,required"`
	NamespaceID   *string            `bind:"kind=header,in=X-Studio-Namespace"`
	Authorization string             `bind:"kind=header,in=Authorization"`
	Connectors    connector.Provider `bind:"kind=connector"`
}

// LinkedTypes keeps the predicate visible to Datly's selected-package
// runtime type scan without a registry or prefix-based discovery.
type LinkedTypes struct {
	ReportCatalogRead ReportCatalogRead
	RunNamespaceRead  RunNamespaceRead
}

func CatalogPredicateDatlyType() reflect.Type { return reflect.TypeFor[LinkedTypes]() }

var CatalogPredicateDatlyLinkedType = CatalogPredicateDatlyType()

// Compute scopes the server-owned SDK reader to reports the verified
// principal owns or may view. Only the SDK wrapper sets the bound scope;
// unscoped in-process access is reserved for trusted internal callers.
func (p *ReportCatalogRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ ReportCatalogScope() (string, bool) })
	if !ok {
		return nil, forbidden("report catalog scope is unavailable")
	}
	principal, scoped := input.ReportCatalogScope()
	if !scoped {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	principal = strings.TrimSpace(principal)
	if principal == "" {
		return nil, forbidden("report catalog subject is required")
	}
	criteria := &xpredicate.Criteria{Expression: `(r.owner_id = ? OR EXISTS (
SELECT 1 FROM report_acl studio_sdk_acl
WHERE studio_sdk_acl.report_id = r.id
  AND studio_sdk_acl.subject_type = 'user'
  AND studio_sdk_acl.subject_id = ?
  AND studio_sdk_acl.can_view = TRUE))`, Placeholders: []any{principal, principal}}
	if p.NamespaceID != nil {
		scope, err := namespacevisibility.ComponentCriteria(oauth.WithBearer(ctx, strings.TrimPrefix(p.Authorization, "Bearer ")), p.Connectors, principal, p.NamespaceID, "r.id")
		if err != nil {
			return nil, err
		}
		criteria.Expression = "(" + criteria.Expression + ") AND (" + scope.Expression + ")"
		criteria.Placeholders = append(criteria.Placeholders, scope.Placeholders...)
	}
	return criteria, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
