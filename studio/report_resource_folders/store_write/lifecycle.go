package store_write

import (
	context "context"
	"fmt"
	"strings"

	"github.com/viant/datly/spec"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// FolderStoreRules customizes role Input.Folders.
type FolderStoreRules struct{}

func FolderStoreRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*FolderStoreRules)(nil)).Elem()
}

var (
	FolderStoreRulesHooks = new(FolderStoreRules)
	FolderStoreRulesDatly = FolderStoreRulesDatlyType()
)

func (hooks *FolderStoreRules) Init(_ context.Context, entity *StoredFolder, state xhandler.LifecycleContext[StoredFolder, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		strings.TrimSpace(entity.FolderId) == "" || entity.Has == nil ||
		!entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.FolderId ||
		!entity.Has.ShouldDelete {
		return fmt.Errorf("resource folder mutation requires complete identity and explicit delete marker")
	}
	if entity.ShouldDelete {
		if state.Previous == nil {
			return &xhandler.Conflict{Entity: "resource_folder", Field: "folder_id", Reason: "resource folder does not exist"}
		}
		return nil
	}
	if !entity.Has.Namespace || !entity.Has.RootPath || !entity.Has.UriPrefix ||
		!entity.Has.Ordinal {
		return fmt.Errorf("resource folder requires a complete snapshot")
	}
	if err := (spec.ResourceFolder{Namespace: entity.Namespace, Root: entity.RootPath,
		URIPrefix: entity.UriPrefix}).Validate(); err != nil {
		return fmt.Errorf("resource folder: %w", err)
	}
	return nil
}
func (hooks *FolderStoreRules) Validate(ctx context.Context, entity *StoredFolder, state xhandler.LifecycleContext[StoredFolder, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FolderStoreRules) AfterSequence(ctx context.Context, entity *StoredFolder, state xhandler.LifecycleContext[StoredFolder, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FolderStoreRules) AfterQueue(ctx context.Context, entity *StoredFolder, state xhandler.LifecycleContext[StoredFolder, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FolderStoreRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
