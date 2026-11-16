package store_write

import (
	"context"
	"github.com/viant/datly-studio/sdk/executionguard"
	xhandler "github.com/viant/xdatly/handler"
)

// BeforeCommit retains embedding-server guards at native transaction
// completion for both direct writes and writes joining an existing owner.
func (o *Output) BeforeCommit(ctx context.Context, _ xhandler.ComponentInvoker) error {
	return executionguard.Check(ctx)
}
