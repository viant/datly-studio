package sdk

import "context"

// OperationGuard performs generic resource authorization before SDK invocation.
// Implementations resolve resource/policy records on the server and enforce any
// mandatory scope. Returning nil must never mean scope was silently discarded.
type OperationGuard interface {
	CheckOperation(context.Context, string, any) error
}

// GuardedTransport composes an external ACL with existing transport checks.
// Keep the underlying transport private to the server assembly.
type GuardedTransport struct {
	Next  Transport
	Guard OperationGuard
}

func (t *GuardedTransport) Invoke(ctx context.Context, operation string, input, output any) error {
	if t.Next == nil || t.Guard == nil {
		return &Error{Code: ErrorForbidden, Message: "authorization is required"}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.Guard.CheckOperation(ctx, operation, input); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return t.Next.Invoke(ctx, operation, input, output)
}
