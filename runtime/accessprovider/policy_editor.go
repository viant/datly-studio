package accessprovider

import (
	"context"

	"github.com/viant/authz"
)

type policyEditorKey struct{}

// PolicyEditorCheck is a trusted host authorization binding. The policy service
// verifies identity, tenant and lease before invoking it. It is never decoded
// from MCP arguments, headers, or a policy document.
type PolicyEditorCheck func(context.Context, authz.Facts, authz.Resource) (bool, error)

func WithPolicyEditor(ctx context.Context, check PolicyEditorCheck) context.Context {
	return context.WithValue(ctx, policyEditorKey{}, check)
}

func PolicyEditor(ctx context.Context) PolicyEditorCheck {
	check, _ := ctx.Value(policyEditorKey{}).(PolicyEditorCheck)
	return check
}
