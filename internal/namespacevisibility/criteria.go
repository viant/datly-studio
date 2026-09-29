package namespacevisibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/accessconfig"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/sqlx/metadata/registry"
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
	expression := "(namespaces.owner_id = ? OR namespaces.visibility = 'public'"
	values := []any{subject}
	provider, err := accessconfig.FromEnvironment()
	if err != nil {
		return nil, forbidden("namespace role verifier is unavailable")
	}
	if provider != nil {
		facts, err := provider.Resolve(ctx)
		if err != nil || facts.Subject != subject || facts.Issuer == "" || !facts.ValidUntil.After(time.Now()) {
			return nil, forbidden("namespace role identity is invalid")
		}
		if len(facts.Roles) > 0 {
			if connectors == nil {
				return nil, forbidden("namespace connector is unavailable")
			}
			db, err := connectors.Connector(ctx, "studio")
			if err != nil || db == nil {
				return nil, forbidden("namespace connector is unavailable")
			}
			product := registry.MatchProduct(db)
			if product == nil {
				return nil, fmt.Errorf("namespace role SQL dialect is unavailable")
			}
			seen := map[string]bool{}
			for _, role := range facts.Roles {
				if role == "" || strings.TrimSpace(role) != role || seen[role] {
					continue
				}
				seen[role] = true
				switch strings.ToLower(product.Name) {
				case "sqlite", "sqlite3":
					expression += " OR (namespaces.visibility = 'private' AND EXISTS (SELECT 1 FROM json_each(namespaces.allowed_roles_json) role WHERE role.type = 'text' AND role.value = ? COLLATE BINARY))"
					values = append(values, role)
				case "mysql":
					expression += " OR (namespaces.visibility = 'private' AND JSON_CONTAINS(namespaces.allowed_roles_json, ?))"
					encoded, err := json.Marshal(role)
					if err != nil {
						return nil, err
					}
					values = append(values, string(encoded))
				default:
					return nil, fmt.Errorf("namespace roles unsupported for %s", product.Name)
				}
			}
		}
	}
	return &xpredicate.Criteria{Expression: strings.ReplaceAll(expression+")", "namespaces.", alias+"."), Placeholders: values}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
