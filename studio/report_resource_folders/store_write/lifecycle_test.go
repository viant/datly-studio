package store_write

import (
	"context"
	"errors"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestFolderStoreRulesRequireValidConfigurationAndDeleteIdentity(t *testing.T) {
	rules := &FolderStoreRules{}
	row := func() *StoredFolder {
		return &StoredFolder{ReportId: "report", VersionNo: 2, FolderId: "folder",
			Namespace: "owner.docs", RootPath: "guide", UriPrefix: "skill://owner-guide/",
			Has: &StoredFolderHas{ReportId: true, VersionNo: true, FolderId: true,
				Namespace: true, RootPath: true, UriPrefix: true, Ordinal: true, ShouldDelete: true}}
	}
	state := xhandler.LifecycleContext[StoredFolder, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row(), state); err != nil {
		t.Fatalf("valid folder rejected: %v", err)
	}
	invalid := row()
	invalid.UriPrefix = "not-a-uri"
	if err := rules.Init(context.Background(), invalid, state); err == nil {
		t.Fatal("invalid folder URI prefix accepted")
	}
	missing := &StoredFolder{ReportId: "report", VersionNo: 2, FolderId: "missing", ShouldDelete: true,
		Has: &StoredFolderHas{ReportId: true, VersionNo: true, FolderId: true, ShouldDelete: true}}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), missing, state); !errors.As(err, &conflict) {
		t.Fatalf("missing folder delete error=%v", err)
	}
}
