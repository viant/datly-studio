package host

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/schema"
)

func TestDefinitionGenerationAdmissionAllowsEmptyAndRejectsUnavailable(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, row := range []struct {
		generation        int
		namespace, status string
	}{{1, a, "building"}, {2, a, "active"}, {3, a, "failed"}, {4, a, "retired"}, {5, b, "building"}} {
		_, err = db.ExecContext(ctx, `INSERT INTO runtime_generations(namespace_id,generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(?,?,?,?,0,'{}','owner',CURRENT_TIMESTAMP)`, row.namespace, row.generation, row.status+row.namespace, row.status)
		require.NoError(t, err)
	}
	store, err := newPublishedDefinitionStore(db)
	require.NoError(t, err)
	defer store.Close(ctx)
	for _, generation := range []int64{1, 2} {
		definitions, err := store.Definitions(ctx, &generation, a)
		require.NoError(t, err)
		require.Empty(t, definitions, "an authorized unpublish/empty namespace generation is valid")
	}
	for _, generation := range []int64{-1, 0, 3, 4, 5, 99} {
		_, err := store.Definitions(ctx, &generation, a)
		require.Error(t, err, "generation %d must not become an active fallback", generation)
	}
	_, err = store.Definitions(ctx, nil, a)
	require.NoError(t, err, "ordinary generation-zero startup stays independent of staged state")
}
