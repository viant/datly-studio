package store_import

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"unicode/utf8"

	xhandler "github.com/viant/xdatly/handler"
)

// ImportRules owns the persisted shape of an uploaded DQL version. The SDK
// authorizes the actor, resolves the bundle entry, allocates the version
// number and computes the spec hash. This hook derives every resource file
// identity from its content in Init and denies inconsistent or unsafe imports
// in Validate before Datly's universal writer inserts the version and its
// files in one transaction: a failed file insert rolls the version back.
type ImportRules struct{}

func ImportRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ImportRules)(nil)).Elem()
}

var (
	ImportRulesHooks = new(ImportRules)
	ImportRulesDatly = ImportRulesDatlyType()
)

const (
	importState            = "draft"
	importAuthoringMode    = "dql"
	importCompileStatus    = "pending"
	importSourceRevision   = 1
	importEmptyJSONObject  = `{}`
	importUnsafePathRunes  = "\\:\x00"
	importCurrentDirectory = "."
)

// Init fills server-owned defaults and derives file identities. Values the
// caller supplied are left intact so Validate can deny inconsistent input.
func (hooks *ImportRules) Init(_ context.Context, version *ImportedVersion, _ xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]) error {
	if version == nil {
		return fmt.Errorf("imported version is required")
	}
	if version.Has == nil {
		version.Has = &ImportedVersionHas{}
	}
	if version.State == "" {
		version.SetState(importState)
	}
	if version.AuthoringMode == "" {
		version.SetAuthoringMode(importAuthoringMode)
	}
	if version.CompileStatus == "" {
		version.SetCompileStatus(importCompileStatus)
	}
	if version.SourceRevision == 0 {
		version.SetSourceRevision(importSourceRevision)
	}
	if version.GeneratedDql == nil && version.AuthoredDql != nil {
		generated := *version.AuthoredDql
		version.SetGeneratedDql(&generated)
	}
	if len(bytes.TrimSpace(version.ComponentSpecJson)) == 0 {
		version.SetComponentSpecJson(json.RawMessage(importEmptyJSONObject))
	}
	if len(bytes.TrimSpace(version.TypeManifestJson)) == 0 {
		version.SetTypeManifestJson(json.RawMessage(importEmptyJSONObject))
	}
	for _, file := range version.File {
		if file == nil {
			return fmt.Errorf("imported resource file is required")
		}
		if file.Has == nil {
			file.Has = &ImportedResourceFileHas{}
		}
		if file.ResourceId == "" {
			file.SetResourceId(resourceIdentity(file.ResourcePath))
		}
		if file.ContentSha256 == "" {
			file.SetContentSha256(contentDigest(file.Content))
		}
		if !file.Has.ContentSize {
			file.SetContentSize(int64(len(file.Content)))
		}
		if !file.Has.IsBinary {
			file.SetIsBinary(!utf8.Valid(file.Content))
		}
		if file.CreatedAt.IsZero() {
			file.SetCreatedAt(version.CreatedAt)
		}
	}
	return nil
}

// Validate denies before any row is written. Every check is an exact rule of
// the import contract; the writer never repairs a violating row.
func (hooks *ImportRules) Validate(_ context.Context, version *ImportedVersion, _ xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]) error {
	if version == nil {
		return fmt.Errorf("imported version is required")
	}
	if strings.TrimSpace(version.ReportId) == "" {
		return fmt.Errorf("imported version report id is required")
	}
	if version.VersionNo <= 0 {
		return fmt.Errorf("imported version number must be allocated")
	}
	if version.State != importState {
		return fmt.Errorf("imported versions must be drafts, got %q", version.State)
	}
	if version.AuthoringMode != importAuthoringMode {
		return fmt.Errorf("imported versions must use dql authoring mode, got %q", version.AuthoringMode)
	}
	if version.CompileStatus != importCompileStatus {
		return fmt.Errorf("imported versions must start with pending compile status, got %q", version.CompileStatus)
	}
	if version.SourceRevision != importSourceRevision {
		return fmt.Errorf("imported versions must start at source revision %d", importSourceRevision)
	}
	for name, value := range map[string]string{"spec_format_version": version.SpecFormatVersion, "spec_hash": version.SpecHash,
		"datly_version": version.DatlyVersion, "compiler_version": version.CompilerVersion, "created_by": version.CreatedBy} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("imported version %s is required", name)
		}
	}
	if version.CreatedAt.IsZero() {
		return fmt.Errorf("imported version created_at is required")
	}
	if version.AuthoredDql == nil || strings.TrimSpace(*version.AuthoredDql) == "" {
		return fmt.Errorf("imported version requires DQL source")
	}
	if version.GeneratedDql == nil || *version.GeneratedDql != *version.AuthoredDql {
		return fmt.Errorf("imported generated DQL must equal the authored DQL")
	}
	if !jsonObject(version.ComponentSpecJson) || !jsonObject(version.TypeManifestJson) {
		return fmt.Errorf("imported version spec and type manifest must be JSON objects")
	}
	if len(version.File) == 0 {
		return fmt.Errorf("import requires at least one resource file")
	}
	seen := map[string]bool{}
	retained := false
	for _, file := range version.File {
		if file == nil {
			return fmt.Errorf("imported resource file is required")
		}
		path := file.ResourcePath
		if !fs.ValidPath(path) || path == importCurrentDirectory || strings.ContainsAny(path, importUnsafePathRunes) {
			return fmt.Errorf("unsafe resource path %q", path)
		}
		if seen[path] {
			return fmt.Errorf("duplicate resource path %s", path)
		}
		seen[path] = true
		if file.ReportId != version.ReportId || file.VersionNo != version.VersionNo {
			return fmt.Errorf("resource file %s must belong to the imported version", path)
		}
		if strings.TrimSpace(file.Namespace) == "" {
			return fmt.Errorf("resource file %s namespace is required", path)
		}
		if file.ResourceId != resourceIdentity(path) {
			return fmt.Errorf("resource file %s id must be the SHA-256 of its path", path)
		}
		if file.ContentSha256 != contentDigest(file.Content) {
			return fmt.Errorf("resource file %s content digest does not match its content", path)
		}
		if file.ContentSize != int64(len(file.Content)) {
			return fmt.Errorf("resource file %s content size does not match its content", path)
		}
		if file.IsBinary != !utf8.Valid(file.Content) {
			return fmt.Errorf("resource file %s binary marker does not match its content", path)
		}
		if file.CreatedAt.IsZero() {
			return fmt.Errorf("resource file %s created_at is required", path)
		}
		retained = retained || string(file.Content) == *version.AuthoredDql
	}
	if !retained {
		return fmt.Errorf("the imported DQL source must be retained as a resource file")
	}
	return nil
}

func (hooks *ImportRules) AfterSequence(context.Context, *ImportedVersion, xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *ImportRules) AfterQueue(context.Context, *ImportedVersion, xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *ImportRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error {
	return nil
}

func resourceIdentity(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}

func contentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func jsonObject(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	return len(trimmed) > 0 && trimmed[0] == '{' && json.Valid(trimmed)
}
