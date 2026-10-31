package leasepredicate

import (
	"context"
	"fmt"
	"github.com/viant/xdatly/predicate"
)

// Condition constrains native UPDATE execution, not just the preceding read.
type Condition struct {
	Input any `bind:"kind=input,required"`
}

func (p *Condition) Compute(ctx context.Context, _ any) (*predicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface {
		LeaseConditionScope() (string, string, int64)
	})
	if !ok {
		return nil, fmt.Errorf("refresh lease condition input unavailable")
	}
	operation, owner, now := input.LeaseConditionScope()
	if owner == "" || now <= 0 {
		return nil, fmt.Errorf("refresh lease condition identity required")
	}
	switch operation {
	case "acquire":
		return &predicate.Criteria{Expression: "bff_sessions.expires_at_unix > ? AND (bff_sessions.refresh_lease_owner = '' OR bff_sessions.refresh_lease_until_unix < ?)", Placeholders: []any{now, now}}, nil
	case "complete":
		return &predicate.Criteria{Expression: "bff_sessions.refresh_lease_owner = ? AND bff_sessions.refresh_lease_until_unix >= ? AND bff_sessions.expires_at_unix > ?", Placeholders: []any{owner, now, now}}, nil
	case "release":
		return &predicate.Criteria{Expression: "bff_sessions.refresh_lease_owner = ?", Placeholders: []any{owner}}, nil
	default:
		return nil, fmt.Errorf("unsupported refresh lease operation %q", operation)
	}
}
