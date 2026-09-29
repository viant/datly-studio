package namespacevisibility

import (
	"context"
	"encoding/hex"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	"strings"
)

// ComponentCriteria adds the selected workspace boundary to a component ID.
// ID-column expressions are supplied by linked Go predicates, never callers.
func ComponentCriteria(ctx context.Context, connectors connector.Provider, subject string, namespaceID *string, idColumn string) (*xpredicate.Criteria, error) {
	if namespaceID == nil {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	decoded, err := hex.DecodeString(*namespaceID)
	if err != nil || len(decoded) != 32 || *namespaceID != strings.ToLower(*namespaceID) {
		return nil, forbidden("valid namespace selection is required")
	}
	visibility, err := Criteria(ctx, connectors, subject, "n")
	if err != nil {
		return nil, err
	}
	expression := `EXISTS (SELECT 1 FROM components namespace_component
 JOIN namespaces n ON n.owner_id=namespace_component.owner_id AND n.name=namespace_component.namespace
 WHERE namespace_component.id=` + idColumn + `
 AND namespace_component.deleted_at IS NULL AND n.deleted_at IS NULL
 AND n.status='active' AND n.namespace_id=? AND ` + visibility.Expression + `)`
	return &xpredicate.Criteria{Expression: expression, Placeholders: append([]any{*namespaceID}, visibility.Placeholders...)}, nil
}
