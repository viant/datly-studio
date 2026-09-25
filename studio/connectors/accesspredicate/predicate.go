// Package accesspredicate owns the server-only connector authorization scope.
package accesspredicate

import (
	"context"
	"errors"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type ConnectorAccess struct {
	Input any `bind:"kind=input,required"`
}

func (p *ConnectorAccess) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface {
		ConnectorAccessScope() (string, string, string)
	})
	if !ok {
		return nil, forbidden("connector access scope is unavailable")
	}
	subject, name, permission := input.ConnectorAccessScope()
	subject, name = strings.TrimSpace(subject), strings.TrimSpace(name)
	if subject == "" || name == "" {
		return nil, forbidden("connector access identity is required")
	}
	column := "can_view"
	switch strings.TrimSpace(permission) {
	case "run":
		column = "can_run"
	case "edit":
		column = "can_edit"
	case "publish":
		column = "can_publish"
	case "dql":
		column = "can_use_dql"
	}
	return &xpredicate.Criteria{Expression: `c.name = ? AND (c.owner_id = ? OR EXISTS (
SELECT 1 FROM reports r JOIN report_acl acl ON acl.report_id = r.id
WHERE r.default_connector_name = c.name AND r.deleted_at IS NULL
  AND acl.subject_type = 'user' AND acl.subject_id = ? AND acl.` + column + ` = TRUE))`,
		Placeholders: []any{name, subject, subject}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
