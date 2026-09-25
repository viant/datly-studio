// Package globalpredicate owns the server-only global publish authorization
// predicate. A publish grant must belong to a live report.
package globalpredicate

import (
	"context"
	"errors"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type GlobalPublish struct {
	Input any `bind:"kind=input,required"`
}

func (p *GlobalPublish) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ GlobalPublishSubject() string })
	if !ok {
		return nil, forbidden("global publish scope is unavailable")
	}
	subject := strings.TrimSpace(input.GlobalPublishSubject())
	if subject == "" {
		return nil, forbidden("global publish subject is required")
	}
	return &xpredicate.Criteria{Expression: `(r.owner_id = ? OR EXISTS (
SELECT 1 FROM report_acl acl
WHERE acl.report_id = r.id AND acl.subject_type = 'user'
  AND acl.subject_id = ? AND acl.can_publish = TRUE))`,
		Placeholders: []any{subject, subject}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
