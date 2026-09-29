package access

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/authz"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CatalogInput struct {
	Kind   string `json:"kind,omitempty"`
	Access string `json:"access,omitempty"`
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
type CatalogVersion struct {
	Published   bool           `json:"published"`
	Resource    authz.Resource `json:"resource"`
	Public      bool           `json:"public"`
	ExplicitACL bool           `json:"explicitAcl"`
	Inherited   bool           `json:"inherited"`
}
type CatalogEntry struct {
	Published   bool             `json:"published"`
	Versions    []CatalogVersion `json:"versions,omitempty"`
	Public      bool             `json:"public"`
	ExplicitACL bool             `json:"explicitAcl"`
	Kind        string           `json:"kind"`
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	OwnerName   string           `json:"ownerName,omitempty"`
	Resource    authz.Resource   `json:"resource"`
	Inherited   bool             `json:"inherited"`
}
type CatalogPage struct {
	Items   []CatalogEntry `json:"items"`
	HasMore bool           `json:"hasMore"`
}
type catalogStore interface {
	Catalog(context.Context, int, int) ([]*catalog.Entry, error)
}

type Catalog struct {
	Source          catalogStore
	Service         *authz.Service
	AuthoringAccess func(context.Context, string) bool
}

func (s *Catalog) List(ctx context.Context, in CatalogInput) (CatalogPage, error) {
	page := CatalogPage{Items: []CatalogEntry{}}
	if in.Limit == 0 {
		in.Limit = 10
	}
	if in.Access != "" && in.Access != "public" && in.Access != "protected" {
		return page, fmt.Errorf("invalid access filter")
	}
	if in.Limit < 1 || in.Limit > 100 || in.Offset < 0 {
		return page, fmt.Errorf("invalid catalog pagination")
	}
	store := s.Source
	if store == nil {
		store, _ = s.Service.Store.(catalogStore)
	}
	if store == nil {
		return page, fmt.Errorf("resource catalog is unavailable")
	}
	// Resolve identity before loading candidates; an invalid credential must not
	// obtain even an empty catalog. Studio authoring visibility or exact unbounded
	// policy inspection is required before any resource metadata is returned.
	if s.Service.Provider == nil {
		return page, authz.ErrDenied
	}
	facts, err := s.Service.Provider.Resolve(ctx)
	if err != nil || facts.Subject == "" || facts.Issuer == "" || !facts.ValidUntil.After(time.Now()) {
		return page, authz.ErrDenied
	}

	query := strings.ToLower(strings.TrimSpace(in.Query))
	groups := map[string]*CatalogEntry{}
	order := []string{}
	for offset := 0; ; offset += 100 {
		rows, err := store.Catalog(ctx, 100, offset)
		if err != nil {
			return page, err
		}
		for _, row := range rows {
			if !facts.ValidUntil.After(time.Now()) {
				return page, authz.ErrDenied
			}
			if row == nil {
				return page, fmt.Errorf("nil catalog row")
			}
			if in.Kind != "" && row.Kind != in.Kind {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(row.Name+" "+row.ID+" "+row.OwnerName+" "+row.Tenant), query) {
				continue
			}
			resource := authz.Resource{Kind: row.PolicyKind, ID: row.PolicyID, Tenant: row.Tenant, Version: row.Version}
			allowed := row.OwnerID != "" && row.OwnerID == facts.Subject
			if s.AuthoringAccess != nil {
				allowed = s.AuthoringAccess(ctx, row.ComponentID)
			}
			if !allowed {
				if _, err = s.Service.Get(ctx, resource); err != nil {
					if ctx.Err() != nil {
						return page, ctx.Err()
					}
					continue
				}
			}
			key := row.Kind + "\x00" + row.ID
			entry := groups[key]
			if entry == nil {
				entry = &CatalogEntry{Kind: row.Kind, ID: row.ID, Name: row.Name, OwnerName: row.OwnerName}
				groups[key] = entry
				order = append(order, key)
			}
			entry.Versions = append(entry.Versions, CatalogVersion{Published: row.Published, Resource: resource, Public: CatalogPublic(row), ExplicitACL: row.HasPolicy, Inherited: row.Kind != row.PolicyKind || row.ID != row.PolicyID})
		}
		if len(rows) < 100 {
			break
		}
	}
	visible := 0
	for _, key := range order {
		entry := groups[key]
		sort.SliceStable(entry.Versions, func(i, j int) bool {
			a, e1 := strconv.Atoi(entry.Versions[i].Resource.Version)
			b, e2 := strconv.Atoi(entry.Versions[j].Resource.Version)
			if e1 == nil && e2 == nil && a != b {
				return a > b
			}
			return entry.Versions[i].Resource.Version > entry.Versions[j].Resource.Version
		})
		current := entry.Versions[0]
		entry.Published = current.Published
		entry.Resource = current.Resource
		entry.Public = current.Public
		entry.ExplicitACL = current.ExplicitACL
		entry.Inherited = current.Inherited
		if in.Access == "public" && !entry.Public || in.Access == "protected" && entry.Public {
			continue
		}
		if visible < in.Offset {
			visible++
			continue
		}
		if len(page.Items) == in.Limit {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, *entry)
	}
	if !facts.ValidUntil.After(time.Now()) {
		return CatalogPage{}, authz.ErrDenied
	}
	return page, nil

}

// CatalogPublic describes consumption rules, not permission-administration
// grants. An inherited gate or an explicit scoped DQL dependency stays protected.
func CatalogPublic(row *catalog.Entry) bool {
	if strings.Contains(row.SourceDQL, "/_studio/access/context/") {
		return false
	}
	if !row.HasPolicy {
		return true
	}
	var policies map[string]authz.Policy
	if json.Unmarshal([]byte(row.PolicyJSON), &policies) != nil {
		return false
	}
	hasConsumption := false
	for _, action := range []string{"discover", "describe", "execute", "retrieve"} {
		if policy, ok := policies[action]; ok {
			hasConsumption = true
			if policy.Mode != "public" || policy.Rule != nil || policy.EntityType != "" {
				return false
			}
		}
	}
	return hasConsumption
}
