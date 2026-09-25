// Package otheractivepredicate selects other active publication rows for
// generation repointing inside the caller's activation transaction.
package otheractivepredicate

import (
	"context"
	"errors"
	"strings"

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
	input, ok := p.Input.(interface{ ExcludedPublicationReport() string })
	if !ok {
		return nil, forbidden("excluded publication identity is unavailable")
	}
	reportID := strings.TrimSpace(input.ExcludedPublicationReport())
	if reportID == "" {
		return nil, forbidden("excluded publication identity is required")
	}
	return &xpredicate.Criteria{Expression: `p.report_id <> ? AND p.publication_status = 'active'`,
		Placeholders: []any{reportID}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
