package sqltransport

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/sdk/executionguard"
)

// Specialized authoring adapters remain the owner of their existing transaction.
// An embedding-server guard is an optional extra completion fence, not a new
// authorization system, transaction engine or source of caller claims.
func commitWithServerGuard(ctx context.Context, tx *sql.Tx) error {
	if err := executionguard.Check(ctx); err != nil {
		return err
	}
	return tx.Commit()
}
