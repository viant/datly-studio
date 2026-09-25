package access

import (
	"context"
)

type Choice struct {
	ID     string  `json:"id,omitempty"`
	Label  string  `json:"label"`
	Entity *Entity `json:"entity,omitempty"`
}

type Choices struct {
	Subjects    []Choice `json:"subject"`
	Roles       []Choice `json:"role"`
	Exposures   []Choice `json:"exposure"`
	Entities    []Choice `json:"entity"`
	EntityTypes []string `json:"entityTypes"`
}

// Directory optionally supplies a tenant-scoped administrative catalog from the
// trusted identity provider. Implementations must authorize their own reads.
type Directory interface {
	Choices(context.Context, Resource, Facts) (Choices, error)
}

type EditorContext struct {
	Choices   Choices `json:"choices"`
	CanManage bool    `json:"canManage"`
	Source    string  `json:"source"`
}

func (s *Service) EditorContext(ctx context.Context, r Resource) (EditorContext, error) {
	doc, facts, err := s.load(ctx, r, "viewAccess")
	if err != nil {
		return EditorContext{}, err
	}
	d, manageErr := s.evaluate(ctx, Request{Resource: r, Action: "manageAccess"}, doc, facts)
	result := EditorContext{CanManage: manageErr == nil && !d.Bounded, Source: "verified-principal"}
	if s.Directory != nil {
		result.Choices, err = s.Directory.Choices(ctx, r, facts)
		if err != nil {
			return EditorContext{}, ErrDenied
		}
		result.Source = "provider-directory"
		return result, nil
	}
	result.Choices.Subjects = []Choice{{ID: facts.Subject, Label: facts.Subject}}
	for _, role := range facts.Roles {
		result.Choices.Roles = append(result.Choices.Roles, Choice{ID: role, Label: role})
	}
	for _, exposure := range facts.Exposures {
		result.Choices.Exposures = append(result.Choices.Exposures, Choice{ID: exposure, Label: exposure})
	}
	types := map[string]bool{}
	flat, err := facts.FlatEntities()
	if err != nil {
		return EditorContext{}, ErrDenied
	}
	for _, entity := range flat {
		e := entity
		result.Choices.Entities = append(result.Choices.Entities, Choice{Entity: &e, Label: e.Type + ": " + e.ID})
		if !types[e.Type] {
			types[e.Type] = true
			result.Choices.EntityTypes = append(result.Choices.EntityTypes, e.Type)
		}
	}
	return result, nil
}
