package settingstore_test

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/settingstore"
	"testing"
)

func TestNativeSettingsWritesPreserveOmittedKeysAndRejectStaleOrInvalidUpdates(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "settings_native", "studio")
	require.NoError(t, datatest.Hydrate(ctx, db, datatest.Table{Name: "namespaces", Rows: []datatest.Row{
		{"owner_id": "owner", "name": "alpha", "title": "Alpha", "status": "active", "etag": 1, "created_at": "2026-10-28 00:00:00", "updated_at": "2026-10-28 00:00:00"},
		{"owner_id": "other", "name": "alpha", "title": "Other", "status": "active", "etag": 1, "created_at": "2026-10-28 00:00:00", "updated_at": "2026-10-28 00:00:00"},
	}}))
	require.NoError(t, settingstore.Write(ctx, db, "owner", "alpha", 1, "owner", map[string]json.RawMessage{"resourceLimits": json.RawMessage(`{"maxBytesProcessed":100}`)}))
	got, err := settingstore.Read(ctx, db, "owner", "alpha")
	require.NoError(t, err)
	require.EqualValues(t, 100, *got.ResourceLimits.MaxBytesProcessed)
	require.NoError(t, settingstore.Write(ctx, db, "owner", "alpha", 2, "owner", map[string]json.RawMessage{"security": json.RawMessage(`{"predicatePackages":[],"requireSecurityPredicate":false}`)}))
	got, err = settingstore.Read(ctx, db, "owner", "alpha")
	require.NoError(t, err)
	require.NotNil(t, got.Security)
	require.EqualValues(t, 100, *got.ResourceLimits.MaxBytesProcessed)
	require.Error(t, settingstore.Write(ctx, db, "owner", "alpha", 1, "owner", map[string]json.RawMessage{"resourceLimits": json.RawMessage(`{"maxBytesProcessed":200}`)}))
	require.Error(t, settingstore.Write(ctx, db, "owner", "alpha", 3, "owner", map[string]json.RawMessage{"resourceLimits": json.RawMessage(`{"maxBytesProcessed":300}`), "security": json.RawMessage(`{"unrecognized":true}`)}))
	got, err = settingstore.Read(ctx, db, "owner", "alpha")
	require.NoError(t, err)
	require.EqualValues(t, 100, *got.ResourceLimits.MaxBytesProcessed)
	var token int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT etag FROM namespaces WHERE owner_id='owner' AND name='alpha'").Scan(&token))
	require.EqualValues(t, 3, token)
	other, err := settingstore.Read(ctx, db, "other", "alpha")
	require.NoError(t, err)
	require.Nil(t, other.ResourceLimits)
	var title string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT title FROM namespaces WHERE owner_id='owner' AND name='alpha'").Scan(&title))
	require.Equal(t, "Alpha", title)
}
