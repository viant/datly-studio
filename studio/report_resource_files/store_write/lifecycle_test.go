package store_write

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/resourceartifact"
	xhandler "github.com/viant/xdatly/handler"
)

func TestFileStoreRulesValidateContentAndPreserveCreation(t *testing.T) {
	content := []byte("hello")
	sum := sha256.Sum256(content)
	now := time.Now().UTC()
	previousCreated := now.Add(-time.Hour)
	row := func() *StoredFile {
		return &StoredFile{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 2, ResourceId: "file",
			Namespace: "owner.docs", ResourcePath: "guide/SKILL.md", Content: content,
			ContentSize: int64(len(content)), ContentSha256: hex.EncodeToString(sum[:]),
			CreatedAt: now, Has: &StoredFileHas{NamespaceId: true, ReportId: true, VersionNo: true,
				ResourceId: true, Namespace: true, ResourcePath: true, MediaType: true,
				Content: true, ContentSize: true, ContentSha256: true, IsBinary: true,
				CreatedAt: true, ShouldDelete: true}}
	}
	rules := &FileStoreRules{}
	previous := &StoredFile{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 2, ResourceId: "file", CreatedAt: previousCreated}
	state := xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredFile, xhandler.NoParent]{Previous: previous}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || !valid.CreatedAt.Equal(previousCreated) {
		t.Fatalf("updated file=%+v err=%v", valid, err)
	}
	foreignState := xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredFile, xhandler.NoParent]{Previous: &StoredFile{NamespaceId: strings.Repeat("b", 64)}}}
	if err := rules.Init(context.Background(), row(), foreignState); err == nil {
		t.Fatal("resource ownership move was accepted")
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
	missing := &StoredFile{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 2, ResourceId: "missing", ShouldDelete: true,
		Has: &StoredFileHas{NamespaceId: true, ReportId: true, VersionNo: true, ResourceId: true, ShouldDelete: true}}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), missing, xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{}); !errors.As(err, &conflict) {
		t.Fatalf("missing file delete error=%v", err)
	}
}

func TestCompiledResourceNamespaceRequiresHostAndImmutableBytes(t *testing.T) {
	content := []byte(`{"version":"fixture"}`)
	sum := sha256.Sum256(content)
	now := time.Now().UTC()
	row := func() *StoredFile {
		return &StoredFile{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 1,
			ResourceId: strings.Repeat("b", 64), Namespace: resourceartifact.Namespace,
			ResourcePath: "compiled/artifact.json", Content: append([]byte(nil), content...),
			ContentSize: int64(len(content)), ContentSha256: hex.EncodeToString(sum[:]), CreatedAt: now,
			Has: &StoredFileHas{NamespaceId: true, ReportId: true, VersionNo: true, ResourceId: true,
				Namespace: true, ResourcePath: true, MediaType: true, Content: true, ContentSize: true,
				ContentSha256: true, IsBinary: true, CreatedAt: true, ShouldDelete: true}}
	}
	rules := &FileStoreRules{}
	if err := rules.Init(context.Background(), row(), xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{}); err == nil {
		t.Fatal("browser-authored compiled artifact was accepted")
	}
	trusted := resourceartifact.WithTrustedWrite(context.Background())
	if err := rules.Init(trusted, row(), xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{}); err != nil {
		t.Fatal(err)
	}
	previous := row()
	state := xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{EntityState: xhandler.EntityState[StoredFile, xhandler.NoParent]{Previous: previous}}
	if err := rules.Init(trusted, row(), state); err != nil {
		t.Fatalf("idempotent artifact write: %v", err)
	}
	changed := row()
	changed.Content = []byte(`{"version":"changed"}`)
	if err := rules.Init(trusted, changed, state); err == nil {
		t.Fatal("same artifact identity accepted different bytes")
	}
	corrupt := row()
	corrupt.ContentSha256 = "damaged"
	corruptState := xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]{EntityState: xhandler.EntityState[StoredFile, xhandler.NoParent]{Previous: corrupt}}
	if err := rules.Init(trusted, row(), corruptState); err == nil {
		t.Fatal("corrupt artifact repaired without an expected previous digest")
	}
	if err := rules.Init(resourceartifact.WithTrustedRepair(trusted, hex.EncodeToString(sum[:])), row(), corruptState); err != nil {
		t.Fatalf("trusted compiler could not repair digest-corrupt artifact: %v", err)
	}
	if err := rules.Init(resourceartifact.WithTrustedRepair(trusted, strings.Repeat("0", 64)), changed, state); err == nil {
		t.Fatal("wrong repair digest changed a valid artifact")
	}
	deleted := row()
	deleted.ShouldDelete = true
	if err := rules.Init(trusted, deleted, state); err == nil {
		t.Fatal("compiled artifact was deleted")
	}
}
