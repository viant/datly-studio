package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceRecoveryExcludesConflictingOwnersWithoutDroppingIndependentVersion(t *testing.T) {
	f := newSnapshotFixture(t)
	versions := []Version{{ReportID: "first", VersionNo: 1}, {ReportID: "second", VersionNo: 1}, {ReportID: "healthy", VersionNo: 1}}
	for _, v := range versions {
		f.version(v.ReportID, v.VersionNo)
	}
	f.file("first", 1, "one", "shared", "same.sql", "SELECT 1")
	f.file("second", 1, "two", "shared", "same.sql", "SELECT 2")
	f.file("healthy", 1, "three", "safe", "own.sql", "SELECT 3")
	_, err := Load(t.Context(), f.db, versions)
	require.Error(t, err, "strict authoring must reject the conflict")
	failed := map[Version]error{}
	loaded, err := LoadRecovering(t.Context(), f.db, versions, func(v Version, err error) bool { failed[v] = err; return true })
	require.NoError(t, err)
	require.Len(t, failed, 2)
	require.Len(t, loaded.ByVersion, 1)
	content, err := loaded.ByVersion[versions[2]].ReadFile("own.sql")
	require.NoError(t, err)
	require.Equal(t, "SELECT 3", string(content))
	_, err = loaded.Store.ReadFile("shared:same.sql")
	require.Error(t, err, "conflicting files must not leak into surviving readers")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = LoadRecovering(ctx, f.db, versions, func(Version, error) bool { t.Error("cancellation must not be suppressed"); return true })
	require.True(t, errors.Is(err, context.Canceled))
}
