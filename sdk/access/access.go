// Package access supplies resource-neutral ACL contracts and evaluation for
// Studio and embedding applications. Authentication providers own fact issuance.
package access

import (
	"context"
	"errors"
	"time"
)

var ErrDenied = errors.New("access denied")

type Resource struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version"`
	Tenant  string `json:"tenant"`
}

type Entity struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Facts must originate from a verified OAuth/OIDC identity and its configured
// authorization provider. Exposures are feature entitlements, not publications.
type Facts struct {
	Subject   string   `json:"subject"`
	Tenant    string   `json:"tenant"`
	Issuer    string   `json:"issuer"`
	Roles     []string `json:"roles"`
	Exposures []string `json:"exposures"`
	// EntityGroups is the canonical typed authorization fact.
	EntityGroups EntityGroups `json:"allowedEntities,omitempty"`
	// Entities is a compatibility view for existing flat fact providers.
	Entities   []Entity  `json:"-"`
	ValidUntil time.Time `json:"validUntil"`
}

// Provider resolves facts using server-verified credentials in ctx. It must not
// trust a subject, role list or entity list supplied in an operation's payload.
type Provider interface {
	Resolve(context.Context) (Facts, error)
}

type Rule struct {
	Kind   string  `json:"kind"` // subject, role, exposure, entity, all, any
	Value  string  `json:"value,omitempty"`
	Entity *Entity `json:"entity,omitempty"`
	Rules  []Rule  `json:"rules,omitempty"`
}

type Policy struct {
	Mode string `json:"mode"` // public or protected
	Rule *Rule  `json:"rule,omitempty"`
	// EntityType requires scoped authorization outside the rule's OR branches.
	EntityType string `json:"entityType,omitempty"`
}

type Request struct {
	Resource Resource
	Action   string
	// Nil selection expands to all allowed entities; empty explicitly denies.
	Selection *[]Entity
}

type Decision struct {
	Entities []Entity
	Bounded  bool
}

// Evaluate checks one explicitly configured action. Callers resolve policies
// from trusted storage and compose dependency decisions separately.
func Evaluate(req Request, policies map[string]Policy, facts Facts, now time.Time) (Decision, error) {
	deny := func() (Decision, error) { return Decision{}, ErrDenied }
	if req.Resource.Kind == "" || req.Resource.ID == "" || req.Resource.Tenant == "" || req.Action == "" {
		return deny()
	}
	p, ok := policies[req.Action]
	if !ok {
		return deny()
	}
	if req.Resource.Tenant != "*" && req.Resource.Tenant != facts.Tenant {
		return deny()
	}
	switch p.Mode {
	case "public":
		// Public management is forbidden, including unrecognized custom actions.
		switch req.Action {
		case "discover", "describe", "execute", "retrieve":
		default:
			return deny()
		}
		if p.Rule != nil || p.EntityType != "" {
			return deny()
		}
	case "protected":
		flat, err := facts.FlatEntities()
		if err != nil {
			return deny()
		}
		facts.Entities = flat
		if facts.Subject == "" || facts.Tenant == "" || facts.Issuer == "" || !facts.ValidUntil.After(now) || p.Rule == nil || !valid(*p.Rule, 0) || !matches(*p.Rule, facts) {
			return deny()
		}
	default:
		return deny()
	}
	if p.EntityType == "" {
		if req.Selection != nil {
			return deny()
		}
		return Decision{}, nil
	}
	allowed := map[Entity]bool{}
	d := Decision{Bounded: true}
	for _, e := range facts.Entities {
		if e.Type == p.EntityType && e.ID != "" && !allowed[e] {
			allowed[e] = true
			d.Entities = append(d.Entities, e)
		}
	}
	if req.Selection != nil {
		d.Entities = nil
		seen := map[Entity]bool{}
		for _, e := range *req.Selection {
			if !allowed[e] || seen[e] {
				return deny()
			}
			seen[e] = true
			d.Entities = append(d.Entities, e)
		}
	}
	if len(d.Entities) == 0 {
		return deny()
	}
	return d, nil
}

func valid(r Rule, depth int) bool {
	if depth > 32 {
		return false
	}
	switch r.Kind {
	case "all", "any":
		if len(r.Rules) == 0 || r.Value != "" || r.Entity != nil {
			return false
		}
		for _, c := range r.Rules {
			if !valid(c, depth+1) {
				return false
			}
		}
		return true
	case "entity":
		return r.Value == "" && len(r.Rules) == 0 && r.Entity != nil && r.Entity.Type != "" && r.Entity.ID != ""
	case "subject", "role", "exposure":
		return r.Value != "" && r.Entity == nil && len(r.Rules) == 0
	}
	return false
}

func matches(r Rule, f Facts) bool {
	contains := func(values []string) bool {
		for _, v := range values {
			if v == r.Value {
				return true
			}
		}
		return false
	}
	switch r.Kind {
	case "subject":
		return f.Subject == r.Value
	case "role":
		return contains(f.Roles)
	case "exposure":
		return contains(f.Exposures)
	case "entity":
		for _, e := range f.Entities {
			if e == *r.Entity {
				return true
			}
		}
	case "all":
		for _, c := range r.Rules {
			if !matches(c, f) {
				return false
			}
		}
		return true
	case "any":
		for _, c := range r.Rules {
			if matches(c, f) {
				return true
			}
		}
	}
	return false
}
