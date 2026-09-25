package store_write

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestFileStoreRulesValidateContentAndPreserveCreation(t *testing.T) {
	content := []byte("hello")
	sum := sha256.Sum256(content)
	now := time.Now().UTC()
	previousCreated := now.Add(-time.Hour)
	row := func() *StoredFile {
		return &StoredFile{ReportId: "report", VersionNo: 2, ResourceId: "file",
			Namespace: "owner.docs", ResourcePath: "guide/SKILL.md", Content: content,
			ContentSize: int64(len(content)), ContentSha256: hex.EncodeToString(sum[:]),
			CreatedAt: now, Has: &StoredFileHas{ReportId: true, VersionNo: true,
				ResourceId: true, Namespace: true, ResourcePath: true, MediaType: true,
				Content: true, ContentSize: true, ContentSha256: true, IsBinary: true,
				CreatedAt: true, ShouldDelete: true}}
	}
	rules := &FileStoreRules{}
	previous := &StoredFile{ReportId: "report", VersionNo: 2, ResourceId: "file", CreatedAt: previousCreated}
	state := xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredFile, xhandler.NoParent]{Previous: previous}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || !valid.CreatedAt.Equal(previousCreated) {
		t.Fatalf("updated file=%+v err=%v", valid, err)
	}
	badSize := row()
	badSize.ContentSize++
	if err := rules.Init(context.Background(), badSize, state); err == nil {
		t.Fatal("resource file accepted incorrect size")
	}
	badDigest := row()
	badDigest.ContentSha256 = "forged"
	if err := rules.Init(context.Background(), badDigest, state); err == nil {
		t.Fatal("resource file accepted incorrect digest")
	}
	missing := &StoredFile{ReportId: "report", VersionNo: 2, ResourceId: "missing", ShouldDelete: true,
		Has: &StoredFileHas{ReportId: true, VersionNo: true, ResourceId: true, ShouldDelete: true}}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), missing, xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{}); !errors.As(err, &conflict) {
		t.Fatalf("missing file delete error=%v", err)
	}
}
