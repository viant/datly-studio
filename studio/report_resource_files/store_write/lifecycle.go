package store_write

import (
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/resourceartifact"
	xhandler "github.com/viant/xdatly/handler"
	"io/fs"
	reflect "reflect"
	"strings"
)

// FileStoreRules customizes role Input.Files.
type FileStoreRules struct{}

func FileStoreRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*FileStoreRules)(nil)).Elem()
}

var (
	FileStoreRulesHooks = new(FileStoreRules)
	FileStoreRulesDatly = FileStoreRulesDatlyType()
)

func (hooks *FileStoreRules) Init(ctx context.Context, entity *StoredFile, state xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		strings.TrimSpace(entity.ResourceId) == "" || entity.Has == nil ||
		!entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.ResourceId ||
		!entity.Has.ShouldDelete {
		return fmt.Errorf("resource file mutation requires complete identity and an explicit delete marker")
	}
	if !entity.Has.NamespaceId {
		return fmt.Errorf("resource namespace ownership is required")
	}
	previousNamespace := ""
	if state.Previous != nil {
		previousNamespace = state.Previous.NamespaceId
	}
	if err := namespaceaccess.ValidateResourceOwnership(entity.NamespaceId, previousNamespace); err != nil {
		return err
	}
	if entity.Namespace == resourceartifact.Namespace || state.Previous != nil && state.Previous.Namespace == resourceartifact.Namespace {
		if !resourceartifact.TrustedWrite(ctx) {
			return fmt.Errorf("compiled resource namespace is host-owned")
		}
		if entity.ShouldDelete {
			return fmt.Errorf("compiled resource artifacts are immutable")
		}
		if state.Previous != nil {
			previous := state.Previous
			previousDigest := sha256.Sum256(previous.Content)
			previousHash := hex.EncodeToString(previousDigest[:])
			corrupt := previous.ContentSize != int64(len(previous.Content)) || previous.ContentSha256 != previousHash
			if previous.Namespace != entity.Namespace || previous.ResourcePath != entity.ResourcePath ||
				(corrupt || previous.ContentSha256 != entity.ContentSha256 || string(previous.Content) != string(entity.Content)) && resourceartifact.RepairDigest(ctx) != previousHash {
				return &xhandler.Conflict{Entity: "compiled_artifact", Field: "resource_id", Reason: "published bytes cannot change"}
			}
		}
	}
	if entity.ShouldDelete {
		if state.Previous == nil {
			return &xhandler.Conflict{Entity: "resource_file", Field: "resource_id", Reason: "resource file does not exist"}
		}
		return nil
	}
	if !entity.Has.Namespace || !entity.Has.ResourcePath || !entity.Has.MediaType ||
		!entity.Has.Content || !entity.Has.ContentSize || !entity.Has.ContentSha256 ||
		!entity.Has.IsBinary || !entity.Has.CreatedAt ||
		strings.TrimSpace(entity.Namespace) == "" ||
		!fs.ValidPath(entity.ResourcePath) || strings.Contains(entity.ResourcePath, "\\") ||
		entity.IsBinary || len(entity.Content) > 1<<20 ||
		entity.ContentSize != int64(len(entity.Content)) {
		return fmt.Errorf("resource file requires a complete valid text snapshot")
	}
	sum := sha256.Sum256(entity.Content)
	if entity.ContentSha256 != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("resource file SHA-256 does not match content")
	}
	if state.Previous != nil {
		entity.SetCreatedAt(state.Previous.CreatedAt)
	} else if entity.CreatedAt.IsZero() {
		return fmt.Errorf("new resource file requires createdAt")
	}
	return nil
}
func (hooks *FileStoreRules) Validate(ctx context.Context, entity *StoredFile, state xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FileStoreRules) AfterSequence(ctx context.Context, entity *StoredFile, state xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FileStoreRules) AfterQueue(ctx context.Context, entity *StoredFile, state xhandler.LifecycleContext[StoredFile, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *FileStoreRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
