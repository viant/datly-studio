// Package otheractivepredicate selects previously active runtime generations
// for retirement inside the caller-owned activation transaction.
package otheractivepredicate

import (
	"context"
	"errors"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type OtherActive struct {
	Input any `bind:"kind=input,required"`
}

func (p *OtherActive) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ ExcludedRuntimeGeneration() int64 })
	if !ok {
		return nil, forbidden("excluded runtime generation is unavailable")
	}
	generation := input.ExcludedRuntimeGeneration()
	if generation <= 0 {
		return nil, forbidden("excluded runtime generation is required")
	}
	return &xpredicate.Criteria{Expression: `g.generation_no <> ? AND g.status = 'active'`,
		Placeholders: []any{generation}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
