// Package catalogpredicate scopes the server-only active runtime reader
// catalog to one generation and an optional verified principal.
package catalogpredicate

import (
	"context"
	"errors"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type ReaderScope struct {
	Input any `bind:"kind=input,required"`
}

func (p *ReaderScope) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ RuntimeReaderScope() (int64, string, bool) })
	if !ok {
		return nil, forbidden("runtime reader scope is unavailable")
	}
	generation, subject, scoped := input.RuntimeReaderScope()
	if generation <= 0 {
		return nil, forbidden("active generation is required")
	}
	criteria := &xpredicate.Criteria{Expression: "p.active_generation = ? AND p.publication_status = 'active' AND r.deleted_at IS NULL",
		Placeholders: []any{generation}}
	if !scoped {
		return criteria, nil
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, forbidden("runtime reader subject is required")
	}
	criteria.Expression += ` AND (r.owner_id = ? OR EXISTS (
SELECT 1 FROM report_acl acl WHERE acl.report_id = r.id
  AND acl.subject_type = 'user' AND acl.subject_id = ? AND acl.can_publish = TRUE))`
	criteria.Placeholders = append(criteria.Placeholders, subject, subject)
	return criteria, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
