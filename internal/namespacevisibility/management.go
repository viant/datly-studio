package namespacevisibility

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespaceroles"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/sqlx/metadata/registry"
	"github.com/viant/xdatly/connector"
	xexec "github.com/viant/xdatly/exec"
	xpredicate "github.com/viant/xdatly/predicate"
)

type roleFacts struct {
	facts      authz.Facts
	credential [32]byte
}

const roleFactsKey = "datly-studio.namespace-role-facts"

func rememberRoles(ctx context.Context, facts authz.Facts) {
	if execution := xexec.GetContext(ctx); execution != nil && oauth.Bearer(ctx) != "" {
		facts.Roles = append([]string(nil), facts.Roles...)
		execution.SetValue(roleFactsKey, roleFacts{facts: facts, credential: sha256.Sum256([]byte(oauth.Bearer(ctx)))})
	}
}

// scopedRoleFacts is an invocation snapshot, never a cross-request user-info cache.
func scopedRoleFacts(ctx context.Context, subject string) (authz.Facts, bool) {
	if ctx == nil || ctx.Err() != nil || subject == "" {
		return authz.Facts{}, false
	}
	if execution := xexec.GetContext(ctx); execution != nil {
		if value, ok := execution.Value(roleFactsKey); ok {
			if saved, ok := value.(roleFacts); ok && saved.facts.Subject == subject && saved.facts.Issuer != "" && saved.facts.ValidUntil.After(time.Now()) && (oauth.Bearer(ctx) == "" || saved.credential == sha256.Sum256([]byte(oauth.Bearer(ctx)))) {
				return saved.facts, true
			}
		}
	}
	return authz.Facts{}, false
}

func verifiedRoles(ctx context.Context, subject string) ([]string, error) {
	if ctx == nil || ctx.Err() != nil || subject == "" {
		return nil, forbidden("namespace identity is required")
	}
	if saved, ok := scopedRoleFacts(ctx, subject); ok {
		return saved.Roles, nil
	}
	provider, err := accessprovider.FromEnvironment(ctx)
	if err != nil || provider == nil {
		return nil, forbidden("namespace role verifier is unavailable")
	}
	facts, err := provider.Resolve(ctx)
	if err != nil || facts.Subject != subject || facts.Issuer == "" || !facts.ValidUntil.After(time.Now()) {
		return nil, forbidden("namespace role identity is invalid")
	}
	rememberRoles(ctx, facts)
	return facts.Roles, nil
}

// CanManage evaluates the persisted namespace grant, never the proposed update.
func CanManage(ctx context.Context, subject, owner string, encoded *string) bool {
	if subject == "" || owner == "" || ctx.Err() != nil {
		return false
	}
	if subject == owner || IsAdministrator(ctx, subject) {
		return true
	}
	return IsManager(ctx, subject, encoded)
}

// IsManager applies only explicit grants in this namespace, excluding global
// administration and implicit ownership of unrelated components.
func IsManager(ctx context.Context, subject string, encoded *string) bool {
	grants, err := namespaceroles.Decode(encoded)
	if err != nil || len(grants.Manage) == 0 {
		return false
	}
	roles, err := verifiedRoles(ctx, subject)
	return err == nil && namespaceroles.Matches(grants.Manage, roles)
}

// AddComponentManagers extends authoring access for the component's actual
// namespace. Resource execution policies and the caller's selection still apply.
func AddComponentManagers(ctx context.Context, connectors connector.Provider, subject, idColumn string, criteria *xpredicate.Criteria) (*xpredicate.Criteria, error) {
	roles, err := verifiedRoles(ctx, subject)
	if err != nil || len(roles) == 0 {
		return criteria, nil
	}
	granted, err := roleCriteria(ctx, connectors, "n", roles, true)
	if err != nil {
		return nil, err
	}
	criteria.Expression = "(" + criteria.Expression + ` OR EXISTS (
 SELECT 1 FROM components managed_component
 JOIN namespaces n ON n.owner_id=managed_component.owner_id AND n.name=managed_component.namespace
 WHERE managed_component.id=` + idColumn + ` AND managed_component.deleted_at IS NULL
 AND n.deleted_at IS NULL AND n.status='active' AND ` + granted.Expression + "))"
	criteria.Placeholders = append(criteria.Placeholders, granted.Placeholders...)
	return criteria, nil
}

// ManagementCriteria is row-scoped; no manager role becomes a global admin.
func ManagementCriteria(ctx context.Context, connectors connector.Provider, subject, alias string) (*xpredicate.Criteria, error) {
	if subject == "" || (alias != "n" && alias != "namespaces") {
		return nil, forbidden("invalid namespace management scope")
	}
	if IsAdministrator(ctx, subject) {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	result := &xpredicate.Criteria{Expression: alias + ".owner_id = ?", Placeholders: []any{subject}}
	roles, err := verifiedRoles(ctx, subject)
	if err != nil {
		return result, nil
	} // owner access remains independent of role service availability
	roleCriteria, err := roleCriteria(ctx, connectors, alias, roles, true)
	if err != nil {
		return nil, err
	}
	result.Expression = "(" + result.Expression + " OR " + roleCriteria.Expression + ")"
	result.Placeholders = append(result.Placeholders, roleCriteria.Placeholders...)
	return result, nil
}

func roleCriteria(ctx context.Context, connectors connector.Provider, alias string, roles []string, managersOnly bool) (*xpredicate.Criteria, error) {
	result := &xpredicate.Criteria{Expression: "1=0"}
	if len(roles) == 0 {
		return result, nil
	}
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
	column := alias + ".allowed_roles_json"
	paths := []string{"manage"}
	if !managersOnly {
		paths = append(paths, "view")
	}
	seen := map[string]bool{}
	for _, role := range roles {
		if role == "" || strings.TrimSpace(role) != role || seen[role] {
			continue
		}
		seen[role] = true
		for _, path := range paths {
			value := "json_extract(" + column + ", '$." + path + "')"
			if path == "view" {
				value = "CASE WHEN json_type(" + column + ")='array' THEN " + column + " ELSE " + value + " END"
			}
			switch strings.ToLower(product.Name) {
			case "sqlite", "sqlite3":
				result.Expression += " OR EXISTS (SELECT 1 FROM json_each(" + value + ") namespace_role WHERE namespace_role.type='text' AND namespace_role.value=? COLLATE BINARY)"
				result.Placeholders = append(result.Placeholders, role)
			case "mysql":
				value = "JSON_EXTRACT(" + column + ", '$." + path + "')"
				if path == "view" {
					value = "CASE WHEN JSON_TYPE(" + column + ")='ARRAY' THEN " + column + " ELSE " + value + " END"
				}
				result.Expression += " OR JSON_CONTAINS(" + value + ", ?)"
				encoded, _ := json.Marshal(role)
				result.Placeholders = append(result.Placeholders, string(encoded))
			default:
				return nil, fmt.Errorf("namespace roles unsupported for %s", product.Name)
			}
		}
	}
	result.Expression = "(" + result.Expression + ")"
	return result, nil
}
