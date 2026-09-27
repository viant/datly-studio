package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/sdk"
	resourcestore "github.com/viant/datly-studio/sdk/transport/sql/internal/resources"
	filesnapshot "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	filewrite "github.com/viant/datly-studio/studio/report_resource_files/store_write"
	foldersnapshot "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	folderwrite "github.com/viant/datly-studio/studio/report_resource_folders/store_write"
	skillsnapshot "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	skillwrite "github.com/viant/datly-studio/studio/report_skill_roots/store_write"
	versioncatalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	versiontouch "github.com/viant/datly-studio/studio/report_versions/store_touch"
	reportcatalog "github.com/viant/datly-studio/studio/reports/store_catalog"
	claimwrite "github.com/viant/datly-studio/studio/resource_namespace_claims/store_write"
	namespacepresence "github.com/viant/datly-studio/studio/resource_namespaces/store_presence"
	namespaceusage "github.com/viant/datly-studio/studio/resource_namespaces/store_usage"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

// The direct SDK path builds a standalone Datly runtime around an SDK-owned
// transaction. A native endpoint supplies ComponentInvoker instead; every
// reader and generic writer then joins the endpoint's invocation-owned unit.
func (t *Transport) invokeResource(ctx context.Context, holder reflect.Type, name, method, path string, input any) (any, error) {
	if t.ComponentInvoker == nil {
		return nil, fmt.Errorf("native resource component invoker is unavailable")
	}
	return t.ComponentInvoker.InvokeComponent(ctx, exec.ComponentRequest{
		Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}},
		Input:  input,
	})
}

func (t *Transport) resourceTouchVersion(ctx context.Context, tx *sql.Tx, row *versiontouch.StoredVersion) error {
	if t.ComponentInvoker == nil {
		return t.writeVersionTouch(ctx, tx, row)
	}
	expected := *row.SourceRevision
	in := &versiontouch.Input{}
	in.SetVersions([]*versiontouch.StoredVersion{row})
	value, err := t.invokeResource(ctx, reflect.TypeFor[versiontouch.VersionComponent](), "version", "PATCH", "/_studio/report-version-store/touch", in)
	if err != nil {
		return err
	}
	result, ok := value.(*versiontouch.Output)
	if !ok || result == nil || len(result.Data) != 1 || result.Data[0] == nil || result.Data[0].SourceRevision == nil || *result.Data[0].SourceRevision != expected+1 {
		return fmt.Errorf("version touch returned %T without next source revision", value)
	}
	return nil
}

func (t *Transport) resourceVersionCatalog(ctx context.Context, tx *sql.Tx, request versionCatalogRequest) ([]*sdk.ReportVersion, error) {
	if t.ComponentInvoker == nil {
		return t.readVersionCatalogTx(ctx, tx, request)
	}
	in := &versioncatalog.Input{ReportId: request.ReportID, VersionNo: request.VersionNo, Limit: request.Limit, Offset: request.Offset,
		Has: &versioncatalog.InputHas{ReportId: true, VersionNo: request.VersionNo > 0, Limit: true, Offset: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[versioncatalog.VersionComponent](), "version", "GET", "/_studio/report-version-store/catalog", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*versioncatalog.Output)
	if !ok || result == nil || len(result.Versions) > request.Limit {
		return nil, fmt.Errorf("version catalog returned invalid %T", value)
	}
	versions := make([]*sdk.ReportVersion, 0, len(result.Versions))
	for _, row := range result.Versions {
		if row == nil || row.ReportId != request.ReportID || row.VersionNo != request.VersionNo {
			return nil, fmt.Errorf("version catalog returned a mismatched row")
		}
		versions = append(versions, &sdk.ReportVersion{ReportID: row.ReportId, VersionNo: row.VersionNo, State: row.State, SourceRevision: row.SourceRevision})
	}
	return versions, nil
}

func (t *Transport) resourceReportCatalog(ctx context.Context, tx *sql.Tx, request reportCatalogRequest) ([]*sdk.Component, error) {
	if t.ComponentInvoker == nil {
		return t.readReportCatalogTx(ctx, tx, request)
	}
	in := &reportcatalog.Input{Id: request.ID, Scoped: false, Limit: request.Limit, Offset: request.Offset,
		Has: &reportcatalog.InputHas{Id: request.ID != "", Subject: true, Scoped: true, Limit: true, Offset: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[reportcatalog.ReportComponent](), "report", "GET", "/_studio/report-store/catalog", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*reportcatalog.Output)
	if !ok || result == nil || len(result.Reports) > request.Limit {
		return nil, fmt.Errorf("report catalog returned invalid %T", value)
	}
	reports := make([]*sdk.Component, 0, len(result.Reports))
	for _, row := range result.Reports {
		if row == nil || row.Id != request.ID || row.OwnerId == "" {
			return nil, fmt.Errorf("report catalog returned a mismatched row")
		}
		reports = append(reports, &sdk.Component{ID: row.Id, OwnerID: row.OwnerId})
	}
	return reports, nil
}

func (t *Transport) resourceWriteFile(ctx context.Context, tx *sql.Tx, row *filewrite.StoredFile) error {
	if t.ComponentInvoker == nil {
		return resourcestore.WriteFile(ctx, t.DB, tx, row)
	}
	in := &filewrite.Input{}
	in.SetFiles([]*filewrite.StoredFile{row})
	value, err := t.invokeResource(ctx, reflect.TypeFor[filewrite.FileComponent](), "file", "PATCH", "/_studio/resource-file-store/write", in)
	if err != nil {
		return err
	}
	result, ok := value.(*filewrite.Output)
	if !ok || result == nil {
		return fmt.Errorf("file writer returned %T", value)
	}
	return nil
}

func (t *Transport) resourceWriteFolder(ctx context.Context, tx *sql.Tx, row *folderwrite.StoredFolder) error {
	if t.ComponentInvoker == nil {
		return resourcestore.WriteFolder(ctx, t.DB, tx, row)
	}
	in := &folderwrite.Input{}
	in.SetFolders([]*folderwrite.StoredFolder{row})
	value, err := t.invokeResource(ctx, reflect.TypeFor[folderwrite.FolderComponent](), "folder", "PATCH", "/_studio/resource-folder-store/write", in)
	if err != nil {
		return err
	}
	result, ok := value.(*folderwrite.Output)
	if !ok || result == nil {
		return fmt.Errorf("folder writer returned %T", value)
	}
	return nil
}

func (t *Transport) resourceWriteSkill(ctx context.Context, tx *sql.Tx, row *skillwrite.StoredSkill) error {
	if t.ComponentInvoker == nil {
		return resourcestore.WriteSkill(ctx, t.DB, tx, row)
	}
	in := &skillwrite.Input{}
	in.SetSkills([]*skillwrite.StoredSkill{row})
	value, err := t.invokeResource(ctx, reflect.TypeFor[skillwrite.SkillComponent](), "skill", "PATCH", "/_studio/skill-root-store/write", in)
	if err != nil {
		return err
	}
	result, ok := value.(*skillwrite.Output)
	if !ok || result == nil {
		return fmt.Errorf("skill writer returned %T", value)
	}
	return nil
}

func (t *Transport) resourceWriteNamespaceClaim(ctx context.Context, tx *sql.Tx, operation string, row *claimwrite.StoredClaim) error {
	if t.ComponentInvoker == nil {
		return resourcestore.WriteNamespaceClaim(ctx, t.DB, tx, operation, row)
	}
	in := &claimwrite.Input{}
	in.SetOperation(operation)
	in.SetClaims([]*claimwrite.StoredClaim{row})
	value, err := t.invokeResource(ctx, reflect.TypeFor[claimwrite.ClaimComponent](), "claim", "PATCH", "/_studio/resource-namespace-claim-store/write", in)
	if err != nil {
		return err
	}
	result, ok := value.(*claimwrite.Output)
	if !ok || result == nil {
		return fmt.Errorf("namespace claim writer returned %T", value)
	}
	return nil
}

func (t *Transport) resourceFileByID(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, resourceID string) (*filesnapshot.SnapshotFile, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.ReadFileByID(ctx, t.DB, tx, reportID, versionNo, resourceID)
	}
	in := &filesnapshot.Input{ReportId: reportID, VersionNo: versionNo, ResourceId: resourceID,
		Has: &filesnapshot.InputHas{ReportId: true, VersionNo: true, ResourceId: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[filesnapshot.FileComponent](), "file", "GET", "/_studio/resource-snapshot/files", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*filesnapshot.Output)
	if !ok || result == nil || len(result.Files) > 1 {
		return nil, fmt.Errorf("resource file lookup returned invalid %T", value)
	}
	if len(result.Files) == 0 {
		return nil, sql.ErrNoRows
	}
	row := result.Files[0]
	if row == nil || row.ReportId != reportID || row.VersionNo != versionNo || row.ResourceId != resourceID {
		return nil, fmt.Errorf("resource file lookup returned a mismatched row")
	}
	return row, nil
}

func (t *Transport) resourceFilesByPath(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, namespace, resourcePath string) ([]*filesnapshot.SnapshotFile, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.ReadFilesByPath(ctx, t.DB, tx, reportID, versionNo, namespace, resourcePath)
	}
	in := &filesnapshot.Input{ReportId: reportID, VersionNo: versionNo, Namespace: namespace, ResourcePath: resourcePath,
		Has: &filesnapshot.InputHas{ReportId: true, VersionNo: true, Namespace: true, ResourcePath: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[filesnapshot.FileComponent](), "file", "GET", "/_studio/resource-snapshot/files", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*filesnapshot.Output)
	if !ok || result == nil {
		return nil, fmt.Errorf("resource file path lookup returned %T", value)
	}
	for _, row := range result.Files {
		if row == nil || row.ReportId != reportID || row.VersionNo != versionNo || row.Namespace != namespace || row.ResourcePath != resourcePath {
			return nil, fmt.Errorf("resource file path lookup returned a mismatched row")
		}
	}
	return result.Files, nil
}

func (t *Transport) resourceFolderByID(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, folderID string) (*foldersnapshot.SnapshotFolder, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.ReadFolderByID(ctx, t.DB, tx, reportID, versionNo, folderID)
	}
	in := &foldersnapshot.Input{ReportId: reportID, VersionNo: versionNo, FolderId: folderID,
		Has: &foldersnapshot.InputHas{ReportId: true, VersionNo: true, FolderId: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[foldersnapshot.FolderComponent](), "folder", "GET", "/_studio/resource-snapshot/folders", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*foldersnapshot.Output)
	if !ok || result == nil || len(result.Folders) > 1 {
		return nil, fmt.Errorf("resource folder lookup returned invalid %T", value)
	}
	if len(result.Folders) == 0 {
		return nil, sql.ErrNoRows
	}
	row := result.Folders[0]
	if row == nil || row.ReportId != reportID || row.VersionNo != versionNo || row.FolderId != folderID {
		return nil, fmt.Errorf("resource folder lookup returned a mismatched row")
	}
	return row, nil
}

func (t *Transport) resourceSkillByID(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, skillID string) (*skillsnapshot.SnapshotSkill, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.ReadSkillByID(ctx, t.DB, tx, reportID, versionNo, skillID)
	}
	in := &skillsnapshot.Input{ReportId: reportID, VersionNo: versionNo, SkillId: skillID,
		Has: &skillsnapshot.InputHas{ReportId: true, VersionNo: true, SkillId: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[skillsnapshot.SkillComponent](), "skill", "GET", "/_studio/resource-snapshot/skills", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*skillsnapshot.Output)
	if !ok || result == nil || len(result.Skills) > 1 {
		return nil, fmt.Errorf("skill root lookup returned invalid %T", value)
	}
	if len(result.Skills) == 0 {
		return nil, sql.ErrNoRows
	}
	row := result.Skills[0]
	if row == nil || row.ReportId != reportID || row.VersionNo != versionNo || row.SkillId != skillID {
		return nil, fmt.Errorf("skill root lookup returned a mismatched row")
	}
	return row, nil
}

func (t *Transport) resourceNamespaceHasResources(ctx context.Context, tx *sql.Tx, reportID, namespace string) (bool, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.NamespaceHasResources(ctx, t.DB, tx, reportID, namespace)
	}
	in := &namespacepresence.Input{ReportId: reportID, Namespace: namespace,
		Has: &namespacepresence.InputHas{ReportId: true, Namespace: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[namespacepresence.UsageComponent](), "usage", "GET", "/_studio/resource-namespace-store/presence", in)
	if err != nil {
		return false, err
	}
	result, ok := value.(*namespacepresence.Output)
	if !ok || result == nil || len(result.Usages) > 1 {
		return false, fmt.Errorf("resource namespace presence returned invalid %T", value)
	}
	for _, row := range result.Usages {
		if row == nil || row.ReportId != reportID || row.Namespace != namespace {
			return false, fmt.Errorf("resource namespace presence returned mismatched row")
		}
	}
	return len(result.Usages) == 1, nil
}

func (t *Transport) resourceForeignNamespaceUsage(ctx context.Context, tx *sql.Tx, namespace, excludeReportID string) ([]*namespaceusage.NamespaceUsage, error) {
	if t.ComponentInvoker == nil {
		return resourcestore.ReadForeignNamespaceUsage(ctx, t.DB, tx, namespace, excludeReportID)
	}
	in := &namespaceusage.Input{Namespace: namespace, ExcludeReportId: excludeReportID,
		Has: &namespaceusage.InputHas{Namespace: true, ExcludeReportId: true}}
	value, err := t.invokeResource(ctx, reflect.TypeFor[namespaceusage.UsageComponent](), "usage", "GET", "/_studio/resource-namespace-store/usage", in)
	if err != nil {
		return nil, err
	}
	result, ok := value.(*namespaceusage.Output)
	if !ok || result == nil || len(result.Usages) > 2 {
		return nil, fmt.Errorf("resource namespace usage returned invalid %T", value)
	}
	for _, row := range result.Usages {
		if row == nil || row.Namespace != namespace || row.ReportId == "" || row.ReportId == excludeReportID {
			return nil, fmt.Errorf("resource namespace usage returned a mismatched row")
		}
	}
	return result.Usages, nil
}
