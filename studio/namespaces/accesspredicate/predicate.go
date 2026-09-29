// Package accesspredicate owns the server-only namespace authorization scope.
package accesspredicate

import (
	"context"
	"errors"
	"strings"

	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/xdatly/connector"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type NamespaceAccess struct {
	Input      any                `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
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
	criteria, err := namespacevisibility.Criteria(ctx, p.Connectors, subject, "n")
	if err != nil {
		return nil, err
	}
	if permission == "edit" || permission == "publish" {
		return &xpredicate.Criteria{Expression: "n.name = ? AND n.owner_id = ?", Placeholders: []any{name, subject}}, nil
	}
	switch permission {
	case "view", "run", "dql":
	default:
		return nil, forbidden("namespace permission is invalid")
	}
	criteria.Expression = "n.name = ? AND " + criteria.Expression
	criteria.Placeholders = append([]any{name}, criteria.Placeholders...)
	return criteria, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }

// NamespaceDirectory consumes only the private store reader's trusted scope.
type NamespaceDirectory struct {
	Input      any                `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

func (p *NamespaceDirectory) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ NamespaceDirectoryScope() (string, bool) })
	if !ok {
		return nil, forbidden("namespace directory scope is unavailable")
	}
	subject, scoped := input.NamespaceDirectoryScope()
	if !scoped {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	return namespacevisibility.Criteria(ctx, p.Connectors, subject, "n")
}
