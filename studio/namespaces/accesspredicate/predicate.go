// Package accesspredicate owns the server-only namespace authorization scope.
package accesspredicate

import (
	"context"
	"errors"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type NamespaceAccess struct {
	Input any `bind:"kind=input,required"`
}

func (p *NamespaceAccess) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface {
		NamespaceAccessScope() (string, string, string)
	})
	if !ok {
		return nil, forbidden("namespace access scope is unavailable")
	}
	subject, name, permission := input.NamespaceAccessScope()
	subject, name = strings.TrimSpace(subject), strings.TrimSpace(name)
	if subject == "" || name == "" {
		return nil, forbidden("namespace access identity is required")
	}
	permission = strings.TrimSpace(permission)
	if permission == "edit" || permission == "publish" {
		return &xpredicate.Criteria{Expression: "n.name = ? AND n.owner_id = ?", Placeholders: []any{name, subject}}, nil
	}
	column := "can_view"
	switch permission {
	case "run":
		column = "can_run"
	case "dql":
		column = "can_use_dql"
	}
	return &xpredicate.Criteria{Expression: `n.name = ? AND (n.owner_id = ? OR EXISTS (
SELECT 1 FROM reports r JOIN report_acl acl ON acl.report_id = r.id
WHERE r.owner_id = n.owner_id AND r.namespace = n.name AND r.deleted_at IS NULL
  AND acl.subject_type = 'user' AND acl.subject_id = ? AND acl.` + column + ` = TRUE))`,
		Placeholders: []any{name, subject, subject}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
