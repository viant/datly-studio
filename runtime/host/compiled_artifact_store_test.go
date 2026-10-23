package host

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/resourceartifact"
	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestCompiledArtifactUsesGenericResourceStoreAndRepairsCorruptBytes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	now := time.Now().UTC()
	namespaceID := namespaceaccess.ID("owner", "general")
	_, err = db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES('main','sqlite','owner','active','{}',1,?,?)`, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO components(id,namespace_id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('report',?,'general','report','Report','owner','active','main','example.com/report','reader',1,?,?)`, namespaceID, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO component_versions(namespace_id,report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,'report',1,'published','dql','{}','studio.v1','hash','{}','valid','v1','studio',1,'owner',?)`, namespaceID, now)
	require.NoError(t, err)
	store, err := newCompiledArtifactStore(db)
	require.NoError(t, err)
	artifact, err := resourceartifact.New("component://"+namespaceID+"/report@1", strings.Repeat("a", 64))
	require.NoError(t, err)
	payload := []byte(`{"version":"datly.runtime-contract/v2"}`)
	_, err = store.Get(ctx, "report", 1, artifact)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, store.Put(ctx, namespaceID, "report", 1, artifact, payload))
	read, err := store.Get(ctx, "report", 1, artifact)
	require.NoError(t, err)
	require.Equal(t, payload, read)
	require.NoError(t, store.Put(ctx, namespaceID, "report", 1, artifact, payload), "same bytes should be idempotent")
	require.ErrorIs(t, store.Put(ctx, namespaceID, "report", 1, artifact, []byte(`{"different":true}`)), errCompiledArtifactConflict)
	_, err = db.Exec(`UPDATE component_resource_files SET content=? WHERE report_id='report' AND version_no=1 AND resource_id=?`, []byte("damaged"), artifact.Key)
	require.NoError(t, err)
	corruptBytes, err := store.Get(ctx, "report", 1, artifact)
	require.ErrorIs(t, err, errCompiledArtifactCorrupt)
	corruptDigest := sha256.Sum256(corruptBytes)
	require.NoError(t, store.Repair(ctx, namespaceID, "report", 1, artifact, payload, hex.EncodeToString(corruptDigest[:])))
	read, err = store.Get(ctx, "report", 1, artifact)
	require.NoError(t, err)
	require.Equal(t, payload, read)
	malformed := []byte(`{"version":`)
	malformedDigest := sha256.Sum256(malformed)
	_, err = db.Exec(`UPDATE component_resource_files SET content=?,content_size=?,content_sha256=? WHERE report_id='report' AND version_no=1 AND resource_id=?`, malformed, len(malformed), hex.EncodeToString(malformedDigest[:]), artifact.Key)
	require.NoError(t, err)
	prior, err := store.Get(ctx, "report", 1, artifact)
	require.NoError(t, err, "checksum-valid but malformed JSON is detected by compiler hydration")
	require.Equal(t, malformed, prior)
	require.ErrorIs(t, store.Put(ctx, namespaceID, "report", 1, artifact, payload), errCompiledArtifactConflict)
	require.NoError(t, store.Repair(ctx, namespaceID, "report", 1, artifact, payload, hex.EncodeToString(malformedDigest[:])))
	read, err = store.Get(ctx, "report", 1, artifact)
	require.NoError(t, err)
	require.Equal(t, payload, read)
	require.NoError(t, store.Close(ctx))
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id='report'`).Scan(&count), "closing the artifact runtime cannot close the borrowed Studio DB")
	require.Equal(t, 1, count)
}
