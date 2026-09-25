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
	updated, err := tx.ExecContext(ctx, `UPDATE report_resource_files SET namespace=?,resource_path=?,media_type=?,content=?,content_size=?,content_sha256=?,is_binary=? WHERE report_id=? AND version_no=? AND resource_id=?`, value.Namespace, value.ResourcePath, nullable(value.MediaType), content, value.ContentSize, value.ContentSHA256, value.IsBinary, value.ReportID, value.VersionNo, value.ResourceID)
	if err != nil {
		return classify(err, "resource file", value.ResourceID)
	}
	if count, _ := updated.RowsAffected(); count == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,media_type,content,content_size,content_sha256,is_binary,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, value.ReportID, value.VersionNo, value.ResourceID, value.Namespace, value.ResourcePath, nullable(value.MediaType), content, value.ContentSize, value.ContentSHA256, value.IsBinary, now); err != nil {
			return classify(err, "resource file", value.ResourceID)
		}
	}
	return nil
}

func (t *Transport) deleteResourceFile(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, resourceID string) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM report_resource_files WHERE report_id=? AND version_no=? AND resource_id=?`, reportID, versionNo, resourceID)
	if err != nil {
		return internal(err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource file was not found"}
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
	updated, err := tx.ExecContext(ctx, `UPDATE report_resource_folders SET namespace=?,root_path=?,uri_prefix=?,ordinal=? WHERE report_id=? AND version_no=? AND folder_id=?`, value.Namespace, value.RootPath, value.URIPrefix, value.Ordinal, value.ReportID, value.VersionNo, value.FolderID)
	if err != nil {
		return classify(err, "resource folder", value.FolderID)
	}
	if count, _ := updated.RowsAffected(); count == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal) VALUES(?,?,?,?,?,?,?)`, value.ReportID, value.VersionNo, value.FolderID, value.Namespace, value.RootPath, value.URIPrefix, value.Ordinal); err != nil {
			return classify(err, "resource folder", value.FolderID)
		}
	}
	return nil
}

func (t *Transport) deleteResourceFolder(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, folderID string) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM report_resource_folders WHERE report_id=? AND version_no=? AND folder_id=?`, reportID, versionNo, folderID)
	if err != nil {
		return classify(err, "resource folder", folderID)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "resource folder was not found"}
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
	var namespace, folderRoot string
	err := tx.QueryRowContext(ctx, `SELECT namespace,root_path FROM report_resource_folders WHERE report_id=? AND version_no=? AND folder_id=?`, value.ReportID, value.VersionNo, value.FolderID).Scan(&namespace, &folderRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "skill folder was not found"}
	}
	if err != nil {
		return internal(err)
	}
	skillFile := path.Join(folderRoot, value.SkillRoot, "SKILL.md")
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_resource_files WHERE report_id=? AND version_no=? AND namespace=? AND resource_path=?`, value.ReportID, value.VersionNo, namespace, skillFile).Scan(&count); err != nil || count != 1 {
		return invalid(errors.New("skill root requires a SKILL.md resource within its folder"))
	}
	if value.SkillID == "" {
		id, err := generatedResourceID("sk")
		if err != nil {
			return internal(err)
		}
		value.SkillID = id
	}
	updated, err := tx.ExecContext(ctx, `UPDATE report_skill_roots SET folder_id=?,skill_root=?,ordinal=? WHERE report_id=? AND version_no=? AND skill_id=?`, value.FolderID, value.SkillRoot, value.Ordinal, value.ReportID, value.VersionNo, value.SkillID)
	if err != nil {
		return classify(err, "skill root", value.SkillID)
	}
	if count, _ := updated.RowsAffected(); count == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO report_skill_roots(report_id,version_no,skill_id,folder_id,skill_root,ordinal) VALUES(?,?,?,?,?,?)`, value.ReportID, value.VersionNo, value.SkillID, value.FolderID, value.SkillRoot, value.Ordinal); err != nil {
			return classify(err, "skill root", value.SkillID)
		}
	}
	return nil
}

func (t *Transport) deleteSkillRoot(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, skillID string) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM report_skill_roots WHERE report_id=? AND version_no=? AND skill_id=?`, reportID, versionNo, skillID)
	if err != nil {
		return internal(err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "skill root was not found"}
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

type resourceQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (t *Transport) validateResourceNamespace(ctx context.Context, db resourceQueryer, reportID, namespace string) error {
	var ownerID string
	err := db.QueryRowContext(ctx, `SELECT owner_id FROM reports WHERE id=?`, reportID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "report was not found"}
	}
	if err != nil {
		return internal(err)
	}
	ownerPackage := sdk.OwnerPackageSegment(ownerID)
	namespace = strings.TrimSpace(namespace)
	if !strings.HasPrefix(namespace, ownerPackage+".") {
		return invalid(fmt.Errorf("resource namespace must use the owner prefix %s.", ownerPackage))
	}
	var foreign int
	err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_resource_files WHERE namespace=? AND report_id<>?`, namespace, reportID).Scan(&foreign)
	if err != nil {
		return internal(err)
	}
	if foreign > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "resource namespace is already owned by another reader"}
	}
	if err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_resource_folders WHERE namespace=? AND report_id<>?`, namespace, reportID).Scan(&foreign); err != nil {
		return internal(err)
	}
	if foreign > 0 {
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
