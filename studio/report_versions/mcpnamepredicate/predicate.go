// Package mcpnamepredicate selects other reports' latest or active revisions
// for server-side MCP tool-name collision validation.
package mcpnamepredicate

import (
	"context"
	"errors"
	"strings"

	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

type OtherReportCandidate struct {
	Input any `bind:"kind=input,required"`
}

func (p *OtherReportCandidate) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, ok := p.Input.(interface{ MCPNameReportID() string })
	if !ok {
		return nil, forbidden("MCP name report identity is unavailable")
	}
	reportID := strings.TrimSpace(input.MCPNameReportID())
	if reportID == "" {
		return nil, forbidden("MCP name report identity is required")
	}
	return &xpredicate.Criteria{Expression: `r.id <> ? AND r.deleted_at IS NULL
AND (v.version_no = (SELECT MAX(v2.version_no) FROM report_versions v2 WHERE v2.report_id = r.id)
  OR v.version_no = p.active_version_no)`, Placeholders: []any{reportID}}, nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
