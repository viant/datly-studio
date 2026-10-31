package datatest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/viant/datly-studio/sdk/artifactrelease/nativeruntime"
	"github.com/viant/datly-studio/sdk/artifactrelease/nativestore"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
)

// NativeReleaseStore uses the actual generated release components against the
// fixture-owned connector. It is test setup, not a compatibility store.
func NativeReleaseStore(t testing.TB, db *sql.DB, scope releasevalue.Scope, actor string) (*nativestore.Store, error) {
	return NativeReleaseStoreWithActor(t, db, scope, func(context.Context) (string, error) { return actor, nil })
}
func NativeReleaseStoreWithActor(t testing.TB, db *sql.DB, scope releasevalue.Scope, actor func(context.Context) (string, error)) (*nativestore.Store, error) {
	t.Helper()
	runtime, err := nativeruntime.New(t.Context(), db)
	if err != nil {
		return nil, err
	}
	return nativestore.New(runtime, scope, actor)
}
