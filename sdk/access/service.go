package access

import (
	"context"
	"errors"
	"time"
)

var ErrConflict = errors.New("policy revision conflict")

type Document struct {
	Resource Resource          `json:"resource"`
	Revision int64             `json:"revision"`
	Policies map[string]Policy `json:"policies"`
}

// Store persists immutable policy revisions. Replace must compare the current
// revision atomically, preventing authorization against a stale policy snapshot.
// Initial policy provisioning is a separate server-owned bootstrap operation.
type Store interface {
	Get(context.Context, Resource) (Document, error)
	Replace(context.Context, Document, int64, string) (Document, error)
}

// Service separates policy inspection from policy administration. The provider
// supplies verified identity/facts; neither operation accepts them from a client.
type Service struct {
	Store     Store
	Provider  Provider
	Directory Directory
	Decisions DecisionProvider
}

// DecisionProvider adds a trusted remote policy decision to local ACL rules.
// It can narrow or deny access; it cannot broaden a local decision.
type DecisionProvider interface {
	Evaluate(context.Context, Request, Document, Facts) (Decision, error)
}

func (s *Service) evaluate(ctx context.Context, request Request, doc Document, facts Facts) (Decision, error) {
	if ctx.Err() != nil {
		return Decision{}, ErrDenied
	}
	local, err := Evaluate(request, doc.Policies, facts, time.Now())
	if err != nil || s.Decisions == nil || doc.Policies[request.Action].Mode == "public" {
		return local, err
	}
	remote, err := s.Decisions.Evaluate(ctx, request, doc, facts)
	if err != nil || ctx.Err() != nil || !facts.ValidUntil.After(time.Now()) {
		return Decision{}, ErrDenied
	}
	return Intersect(local, remote)
}

func (s *Service) load(ctx context.Context, resource Resource, action string) (Document, Facts, error) {
	if s.Store == nil || s.Provider == nil {
		return Document{}, Facts{}, ErrDenied
	}
	facts, err := s.Provider.Resolve(ctx)
	if err != nil {
		return Document{}, Facts{}, ErrDenied
	}
	doc, err := s.Store.Get(ctx, resource)
	if err != nil {
		return Document{}, Facts{}, ErrDenied
	}
	if doc.Resource != resource || doc.Revision < 1 {
		return Document{}, Facts{}, ErrDenied
	}
	decision, err := s.evaluate(ctx, Request{Resource: resource, Action: action}, doc, facts)
	// Policy management is an operation on the whole resource. Entity-bounded
	// decisions cannot authorize a whole-document policy read or replacement.
	if err != nil || decision.Bounded {
		return Document{}, Facts{}, ErrDenied
	}
	return doc, facts, nil
}

// Authorize evaluates current policy for execution. Callers must apply any
// returned entity scope before releasing data.
func (s *Service) Authorize(ctx context.Context, request Request) (Decision, error) {
	decision, _, err := s.AuthorizeWithFacts(ctx, request)
	return decision, err
}

// AuthorizeWithFacts returns the verified facts used for this exact decision.
// A downstream component can bind the narrowed decision together with its
// principal roles/exposures without resolving a second, potentially different
// identity snapshot. Facts are never accepted from the request payload.
func (s *Service) AuthorizeWithFacts(ctx context.Context, request Request) (Decision, Facts, error) {
	if s.Store == nil {
		return Decision{}, Facts{}, ErrDenied
	}
	doc, err := s.Store.Get(ctx, request.Resource)
	if err != nil || doc.Resource != request.Resource || doc.Revision < 1 {
		return Decision{}, Facts{}, ErrDenied
	}
	p, ok := doc.Policies[request.Action]
	if !ok {
		return Decision{}, Facts{}, ErrDenied
	}
	var facts Facts
	if p.Mode != "public" || request.Resource.Tenant != "*" {
		if s.Provider == nil {
			return Decision{}, Facts{}, ErrDenied
		}
		facts, err = s.Provider.Resolve(ctx)
		if err != nil {
			return Decision{}, Facts{}, ErrDenied
		}
	}
	decision, err := s.evaluate(ctx, request, doc, facts)
	if err != nil {
		return Decision{}, Facts{}, err
	}
	return decision, facts, nil
}

func (s *Service) Get(ctx context.Context, resource Resource) (Document, error) {
	doc, _, err := s.load(ctx, resource, "viewAccess")
	return doc, err
}

func (s *Service) Replace(ctx context.Context, candidate Document) (Document, error) {
	current, facts, err := s.load(ctx, candidate.Resource, "manageAccess")
	if err != nil {
		return Document{}, err
	}
	if current.Revision != candidate.Revision {
		return Document{}, ErrConflict
	}
	if len(candidate.Policies) == 0 {
		return Document{}, ErrDenied
	}
	for action, policy := range candidate.Policies {
		if err = ValidatePolicy(action, policy); err != nil {
			return Document{}, err
		}
	}
	if err = ctx.Err(); err != nil {
		return Document{}, err
	}
	return s.Store.Replace(ctx, candidate, current.Revision, facts.Subject)
}

func ValidatePolicy(action string, p Policy) error {
	if action == "" {
		return ErrDenied
	}
	switch p.Mode {
	case "public":
		if p.Rule != nil || p.EntityType != "" {
			return ErrDenied
		}
		switch action {
		case "discover", "describe", "execute", "retrieve":
			return nil
		}
	case "protected":
		if p.Rule != nil && valid(*p.Rule, 0) {
			return nil
		}
	}
	return ErrDenied
}
