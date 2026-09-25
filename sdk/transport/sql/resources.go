package sqltransport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/viant/datly-studio/sdk"
	filestore "github.com/viant/datly-studio/studio/report_resource_files/store_write"
	folderstore "github.com/viant/datly-studio/studio/report_resource_folders/store_write"
	skillstore "github.com/viant/datly-studio/studio/report_skill_roots/store_write"
	versiontouch "github.com/viant/datly-studio/studio/report_versions/store_touch"
	"github.com/viant/datly/spec"
	xhandler "github.com/viant/xdatly/handler"
)

func (t *Transport) resources(ctx context.Context, operation string, input, output any) error {
	var identity struct {
		ReportID  string `json:"reportId"`
		VersionNo int    `json:"versionNo"`
	}
	if err := decode(input, &identity); err != nil {
		return invalid(err)
	}
	switch operation {
	case sdk.OperationResourcesGet:
		return t.resourceSnapshot(ctx, identity.ReportID, identity.VersionNo, output)
	case sdk.OperationResourcesUpsertFile:
		var value sdk.ResourceFile
		if err := decode(input, &value); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, value.ReportID, value.VersionNo, value.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.upsertResourceFile(ctx, tx, &value) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, value.ReportID, value.VersionNo, output)
	case sdk.OperationResourcesDeleteFile:
		var in sdk.ResourceDeleteInput
		if err := decode(input, &in); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, in.ReportID, in.VersionNo, in.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.deleteResourceFile(ctx, tx, in.ReportID, in.VersionNo, in.ResourceID) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, in.ReportID, in.VersionNo, output)
	case sdk.OperationResourcesUpsertFolder:
		var value sdk.ResourceFolder
		if err := decode(input, &value); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, value.ReportID, value.VersionNo, value.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.upsertResourceFolder(ctx, tx, &value) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, value.ReportID, value.VersionNo, output)
	case sdk.OperationResourcesDeleteFolder:
		var in sdk.ResourceDeleteInput
		if err := decode(input, &in); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, in.ReportID, in.VersionNo, in.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.deleteResourceFolder(ctx, tx, in.ReportID, in.VersionNo, in.FolderID) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, in.ReportID, in.VersionNo, output)
	case sdk.OperationResourcesUpsertSkill:
		var value sdk.SkillRoot
		if err := decode(input, &value); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, value.ReportID, value.VersionNo, value.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.upsertSkillRoot(ctx, tx, &value) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, value.ReportID, value.VersionNo, output)
	case sdk.OperationResourcesDeleteSkill:
		var in sdk.ResourceDeleteInput
		if err := decode(input, &in); err != nil {
			return invalid(err)
		}
		if err := t.mutateResources(ctx, in.ReportID, in.VersionNo, in.ExpectedSourceRevision, func(tx *sql.Tx) error { return t.deleteSkillRoot(ctx, tx, in.ReportID, in.VersionNo, in.SkillID) }); err != nil {
			return err
		}
		return t.resourceSnapshot(ctx, in.ReportID, in.VersionNo, output)
	}
	return invalid(errors.New("unsupported resources operation"))
}

func (t *Transport) resourceSnapshot(ctx context.Context, reportID string, versionNo int, output any) error {
	if err := t.requireResourceVersion(ctx, reportID, versionNo); err != nil {
		return err
	}
	result := &sdk.ResourceSnapshot{}
	version, err := t.getVersionValue(ctx, reportID, versionNo)
	if err != nil {
		return err
	}
	result.Version = version
	if err := t.readResourceSnapshot(ctx, reportID, versionNo, result); err != nil {
		return err
	}
	return assign(output, result)
}

func (t *Transport) upsertResourceFile(ctx context.Context, tx *sql.Tx, value *sdk.ResourceFile) error {
	if value == nil || strings.TrimSpace(value.ReportID) == "" || value.VersionNo <= 0 || strings.TrimSpace(value.Namespace) == "" || strings.TrimSpace(value.ResourcePath) == "" {
		return invalid(errors.New("reportId, versionNo, namespace, and resourcePath are required"))
	}
	if err := t.validateResourceNamespace(ctx, tx, value.ReportID, value.Namespace); err != nil {
		return err
	}
	if err := validateResourcePath(value.ResourcePath); err != nil {
		return invalid(err)
	}
	if value.IsBinary {
		return invalid(errors.New("binary resource authoring is not available through the text SDK"))
	}
	if len(value.Content) > 1<<20 {
		return invalid(errors.New("resource content exceeds 1 MiB"))
	}
	if value.ResourceID == "" {
		id, err := generatedResourceID("rf")
		if err != nil {
			return internal(err)
		}
		value.ResourceID = id
	}
	content := []byte(value.Content)
	sum := sha256.Sum256(content)
	value.ContentSize, value.ContentSHA256 = int64(len(content)), hex.EncodeToString(sum[:])
	now := t.now()
	err := t.writeResourceFile(ctx, tx, &filestore.StoredFile{ReportId: value.ReportID,
		VersionNo: value.VersionNo, ResourceId: value.ResourceID, Namespace: value.Namespace,
		ResourcePath: value.ResourcePath, MediaType: namespaceOptionalDescription(value.MediaType),
		Content: content, ContentSize: value.ContentSize, ContentSha256: value.ContentSHA256,
		IsBinary: value.IsBinary, CreatedAt: now,
		Has: &filestore.StoredFileHas{ReportId: true, VersionNo: true, ResourceId: true,
			Namespace: true, ResourcePath: true, MediaType: true, Content: true,
			ContentSize: true, ContentSha256: true, IsBinary: true, CreatedAt: true,
			ShouldDelete: true}})
	if err != nil {
		return classify(err, "resource file", value.ResourceID)
	}
	return nil
}

func (t *Transport) deleteResourceFile(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, resourceID string) error {
	if _, err := t.readResourceFileByID(ctx, tx, reportID, versionNo, resourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource file was not found"}
		}
		return internal(err)
	}
	err := t.writeResourceFile(ctx, tx, &filestore.StoredFile{ReportId: reportID,
		VersionNo: versionNo, ResourceId: resourceID, ShouldDelete: true,
		Has: &filestore.StoredFileHas{ReportId: true, VersionNo: true, ResourceId: true, ShouldDelete: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource file was not found"}
		}
		return internal(err)
	}
	return nil
}

func (t *Transport) upsertResourceFolder(ctx context.Context, tx *sql.Tx, value *sdk.ResourceFolder) error {
	if value == nil || strings.TrimSpace(value.ReportID) == "" || value.VersionNo <= 0 || strings.TrimSpace(value.Namespace) == "" || strings.TrimSpace(value.URIPrefix) == "" {
		return invalid(errors.New("reportId, versionNo, namespace, and uriPrefix are required"))
	}
	if err := t.validateResourceNamespace(ctx, tx, value.ReportID, value.Namespace); err != nil {
		return err
	}
	if err := (spec.ResourceFolder{Namespace: value.Namespace, Root: value.RootPath, URIPrefix: value.URIPrefix}).Validate(); err != nil {
		return invalid(err)
	}
	if value.FolderID == "" {
		id, err := generatedResourceID("rd")
		if err != nil {
			return internal(err)
		}
		value.FolderID = id
	}
	err := t.writeResourceFolder(ctx, tx, &folderstore.StoredFolder{ReportId: value.ReportID,
		VersionNo: value.VersionNo, FolderId: value.FolderID, Namespace: value.Namespace,
		RootPath: value.RootPath, UriPrefix: value.URIPrefix, Ordinal: value.Ordinal,
		Has: &folderstore.StoredFolderHas{ReportId: true, VersionNo: true, FolderId: true,
			Namespace: true, RootPath: true, UriPrefix: true, Ordinal: true, ShouldDelete: true}})
	if err != nil {
		return classify(err, "resource folder", value.FolderID)
	}
	return nil
}

func (t *Transport) deleteResourceFolder(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, folderID string) error {
	if _, err := t.readResourceFolderByID(ctx, tx, reportID, versionNo, folderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource folder was not found"}
		}
		return internal(err)
	}
	err := t.writeResourceFolder(ctx, tx, &folderstore.StoredFolder{ReportId: reportID,
		VersionNo: versionNo, FolderId: folderID, ShouldDelete: true,
		Has: &folderstore.StoredFolderHas{ReportId: true, VersionNo: true, FolderId: true, ShouldDelete: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource folder was not found"}
		}
		return classify(err, "resource folder", folderID)
	}
	return nil
}

func (t *Transport) upsertSkillRoot(ctx context.Context, tx *sql.Tx, value *sdk.SkillRoot) error {
	if value == nil || strings.TrimSpace(value.ReportID) == "" || value.VersionNo <= 0 || strings.TrimSpace(value.FolderID) == "" {
		return invalid(errors.New("reportId, versionNo, and folderId are required"))
	}
	if value.SkillRoot == "" {
		value.SkillRoot = "."
	}
	if value.SkillRoot != "." && (!fs.ValidPath(value.SkillRoot) || strings.Contains(value.SkillRoot, "\\")) {
		return invalid(errors.New("skillRoot must be a relative resource path"))
	}
	folder, err := t.readResourceFolderByID(ctx, tx, value.ReportID, value.VersionNo, value.FolderID)
	if errors.Is(err, sql.ErrNoRows) {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "skill folder was not found"}
	}
	if err != nil {
		return internal(err)
	}
	skillFile := path.Join(folder.RootPath, value.SkillRoot, "SKILL.md")
	files, err := t.readResourceFilesByPath(ctx, tx, value.ReportID, value.VersionNo, folder.Namespace, skillFile)
	if err != nil {
		return internal(err)
	}
	if len(files) != 1 {
		return invalid(errors.New("skill root requires a SKILL.md resource within its folder"))
	}
	if value.SkillID == "" {
		id, err := generatedResourceID("sk")
		if err != nil {
			return internal(err)
		}
		value.SkillID = id
	}
	err = t.writeSkillRoot(ctx, tx, &skillstore.StoredSkill{ReportId: value.ReportID,
		VersionNo: value.VersionNo, SkillId: value.SkillID, FolderId: value.FolderID,
		SkillRoot: value.SkillRoot, Ordinal: value.Ordinal,
		Has: &skillstore.StoredSkillHas{ReportId: true, VersionNo: true, SkillId: true,
			FolderId: true, SkillRoot: true, Ordinal: true, ShouldDelete: true}})
	if err != nil {
		return classify(err, "skill root", value.SkillID)
	}
	return nil
}

func (t *Transport) deleteSkillRoot(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, skillID string) error {
	if _, err := t.readSkillRootByID(ctx, tx, reportID, versionNo, skillID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "skill root was not found"}
		}
		return internal(err)
	}
	err := t.writeSkillRoot(ctx, tx, &skillstore.StoredSkill{ReportId: reportID,
		VersionNo: versionNo, SkillId: skillID, ShouldDelete: true,
		Has: &skillstore.StoredSkillHas{ReportId: true, VersionNo: true, SkillId: true, ShouldDelete: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "skill root was not found"}
		}
		return internal(err)
	}
	return nil
}

func (t *Transport) requireResourceVersion(ctx context.Context, reportID string, versionNo int) error {
	if strings.TrimSpace(reportID) == "" || versionNo <= 0 {
		return invalid(errors.New("reportId and versionNo are required"))
	}
	_, err := t.getVersionValue(ctx, reportID, versionNo)
	return err
}

func (t *Transport) validateResourceNamespace(ctx context.Context, tx *sql.Tx, reportID, namespace string) error {
	reports, err := t.readReportCatalogTx(ctx, tx, reportCatalogRequest{ID: reportID, Limit: 2, Unscoped: true})
	if err != nil {
		return internal(err)
	}
	if len(reports) == 0 {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "report was not found"}
	}
	if len(reports) != 1 {
		return internal(errors.New("report catalog returned ambiguous rows"))
	}
	ownerPackage := sdk.OwnerPackageSegment(reports[0].OwnerID)
	namespace = strings.TrimSpace(namespace)
	if !strings.HasPrefix(namespace, ownerPackage+".") {
		return invalid(fmt.Errorf("resource namespace must use the owner prefix %s.", ownerPackage))
	}
	foreign, err := t.readForeignResourceNamespaceUsage(ctx, tx, namespace, reportID)
	if err != nil {
		return internal(err)
	}
	if len(foreign) > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "resource namespace is already owned by another reader"}
	}
	return nil
}

func (t *Transport) mutateResources(ctx context.Context, reportID string, versionNo int, expectedRevision int64, mutation func(*sql.Tx) error) (err error) {
	if strings.TrimSpace(reportID) == "" || versionNo <= 0 {
		return invalid(errors.New("reportId and versionNo are required"))
	}
	if expectedRevision <= 0 {
		return invalid(errors.New("expectedSourceRevision is required for resource mutations"))
	}
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	value := &versiontouch.StoredVersion{ReportId: reportID, VersionNo: versionNo, SourceRevision: &expectedRevision,
		Has: &versiontouch.StoredVersionHas{ReportId: true, VersionNo: true, SourceRevision: true}}
	if writeErr := t.writeVersionTouch(ctx, tx, value); writeErr != nil {
		var conflict *xhandler.Conflict
		if !errors.As(writeErr, &conflict) {
			return internal(writeErr)
		}
		versions, readErr := t.readVersionCatalogTx(ctx, tx, versionCatalogRequest{ReportID: reportID, VersionNo: versionNo, Limit: 2})
		if readErr != nil {
			return internal(readErr)
		}
		if len(versions) == 0 {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "reader version was not found"}
		}
		if len(versions) != 1 {
			return internal(errors.New("version catalog returned ambiguous rows"))
		}
		message := "version source revision does not match"
		if versions[0].SourceRevision == expectedRevision {
			message = "published version cannot be mutated"
		}
		return &sdk.Error{Code: sdk.ErrorConflict, Message: message, ExpectedSourceRevision: expectedRevision, CurrentSourceRevision: versions[0].SourceRevision}
	}
	if err = mutation(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return internal(err)
	}
	return nil
}

func validateResourcePath(value string) error {
	if !fs.ValidPath(value) || strings.Contains(value, "\\") {
		return fmt.Errorf("resourcePath must be a relative resource path")
	}
	return nil
}

func generatedResourceID(prefix string) (string, error) {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}
