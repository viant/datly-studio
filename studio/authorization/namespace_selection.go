package authorization

import (
	"context"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	"strings"
)

// NamespaceSelection binds selection independently of the resource permission.
// These headers never supply roles or override verified JWT identity.
type NamespaceSelection struct {
	NamespaceID   *string            `bind:"kind=header,in=X-Studio-Namespace"`
	Authorization string             `bind:"kind=header,in=Authorization"`
	Connectors    connector.Provider `bind:"kind=connector"`
}

func (selection *NamespaceSelection) report(ctx context.Context, input any, idColumn, permission string) (*xpredicate.Criteria, error) {
	criteria, err := reportCriteria(ctx, input, idColumn, permission)
	if err != nil {
		return nil, err
	}
	return selection.constrain(ctx, input, criteria, idColumn)
}

func (selection *NamespaceSelection) constrain(ctx context.Context, input any, criteria *xpredicate.Criteria, idColumn string) (*xpredicate.Criteria, error) {
	if selection.NamespaceID == nil {
		return criteria, nil
	}
	principal, err := subject(ctx, input)
	if err != nil {
		return nil, err
	}
	scope, err := namespacevisibility.ComponentCriteria(oauth.WithBearer(ctx, strings.TrimPrefix(selection.Authorization, "Bearer ")), selection.Connectors, principal, selection.NamespaceID, idColumn)
	if err != nil {
		return nil, err
	}
	criteria.Expression = "(" + criteria.Expression + ") AND (" + scope.Expression + ")"
	criteria.Placeholders = append(criteria.Placeholders, scope.Placeholders...)
	return criteria, nil
}
