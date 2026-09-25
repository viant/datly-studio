// Package catalogpredicate owns the server-only report catalog ACL predicate.
// It does not import the SDK transport or the public Studio authorization host.
package catalogpredicate

import (
	"context"
	"errors"
	"reflect"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type ReportCatalogRead struct {
	Input any `bind:"kind=input,required"`
}

// LinkedTypes keeps the predicate visible to Datly's selected-package
// runtime type scan without a registry or prefix-based discovery.
type LinkedTypes struct{ ReportCatalogRead ReportCatalogRead }

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
	return &xpredicate.Criteria{Expression: `(r.owner_id = ? OR EXISTS (
SELECT 1 FROM report_acl studio_sdk_acl
WHERE studio_sdk_acl.report_id = r.id
  AND studio_sdk_acl.subject_type = 'user'
  AND studio_sdk_acl.subject_id = ?
  AND studio_sdk_acl.can_view = TRUE))`, Placeholders: []any{principal, principal}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
