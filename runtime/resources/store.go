// Package resources materializes versioned Studio resource rows as immutable
// Datly resource stores and explicitly declared MCP folder/skill plans.
package resources

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing/fstest"

	bindresource "github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	storedfiles "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	storedfolders "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	storedskills "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	dexec "github.com/viant/datly/exec"
	mcpresource "github.com/viant/datly/mcp/resource"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
)

type Version struct {
	ReportID  string
	VersionNo int
}

type Loaded struct {
	Store           *bindresource.Store
	ByVersion       map[Version]*bindresource.Store
	Folders         []mcpresource.Folder
	ResourceReports map[string]string
}

// Load reads one immutable resource generation. Namespaces are application-wide
// authority: two active components cannot register different files under the
// same namespace.
func Load(ctx context.Context, db *sql.DB, versions []Version) (*Loaded, error) {
	if db == nil {
		return nil, fmt.Errorf("Studio database is required")
	}
	for _, version := range versions {
		if strings.TrimSpace(version.ReportID) == "" || version.VersionNo <= 0 {
			return nil, fmt.Errorf("resource version identity is required")
		}
	}
	result := &Loaded{Store: bindresource.New(), ByVersion: map[Version]*bindresource.Store{}, ResourceReports: map[string]string{}}
	if len(versions) == 0 {
		return result, nil
	}
	reader, err := newSnapshotReader(db)
	if err != nil {
		return nil, err
	}
	defer reader.runtime.Shutdown(context.Background())
	namespaces := map[string]fstest.MapFS{}
	for _, version := range versions {
		local := fstest.MapFS{}
		files, err := reader.files(ctx, version)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if err = validFile(file.Namespace, file.ResourcePath); err != nil {
				return nil, err
			}
			if _, exists := local[file.ResourcePath]; exists {
				return nil, fmt.Errorf("duplicate default resource path %q for report %s", file.ResourcePath, version.ReportID)
			}
			item := &fstest.MapFile{Data: append([]byte(nil), file.Content...)}
			local[file.ResourcePath] = item
			group, registered := namespaces[file.Namespace]
			if !registered {
				group = fstest.MapFS{}
			}
			if _, exists := group[file.ResourcePath]; exists {
				return nil, fmt.Errorf("duplicate resource namespace/path %s:%s", file.Namespace, file.ResourcePath)
			}
			group[file.ResourcePath] = item
			namespaces[file.Namespace] = group
		}
		if len(local) > 0 {
			scoped, err := result.Store.WithDefault(local)
			if err != nil {
				return nil, err
			}
			result.ByVersion[version] = scoped
		} else {
			result.ByVersion[version] = result.Store
		}
	}
	for namespace, files := range namespaces {
		if err := result.Store.Register(namespace, files); err != nil {
			return nil, err
		}
	}
	for _, version := range versions {
		folders, err := reader.folders(ctx, version)
		if err != nil {
			return nil, err
		}
		for _, folder := range folders {
			if owner, ok := result.ResourceReports[folder.URIPrefix]; ok && owner != version.ReportID {
				return nil, fmt.Errorf("resource URI prefix %q is owned by both reports %s and %s", folder.URIPrefix, owner, version.ReportID)
			}
			result.ResourceReports[folder.URIPrefix] = version.ReportID
		}
		result.Folders = append(result.Folders, folders...)
	}
	return result, nil
}

type snapshotReader struct {
	runtime                                  *druntime.Runtime
	filesTarget, foldersTarget, skillsTarget dexec.ComponentTarget
}

func newSnapshotReader(db *sql.DB) (*snapshotReader, error) {
	resources := bindresource.New()
	for _, item := range []struct {
		name  string
		files fs.FS
	}{
		{storedfiles.FileDatlyResourceNamespace, storedfiles.FileDatlyResources},
		{storedfolders.FolderDatlyResourceNamespace, storedfolders.FolderDatlyResources},
		{storedskills.SkillDatlyResourceNamespace, storedskills.SkillDatlyResources},
	} {
		if err := resources.Register(item.name, item.files); err != nil {
			return nil, err
		}
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	var registrations []*registry.RegisteredComponent
	var targets [3]dexec.ComponentTarget
	for i, item := range []struct {
		holder, input, output reflect.Type
	}{
		{reflect.TypeOf(storedfiles.FileComponent{}), reflect.TypeOf(storedfiles.Input{}), reflect.TypeOf(storedfiles.Output{})},
		{reflect.TypeOf(storedfolders.FolderComponent{}), reflect.TypeOf(storedfolders.Input{}), reflect.TypeOf(storedfolders.Output{})},
		{reflect.TypeOf(storedskills.SkillComponent{}), reflect.TypeOf(storedskills.Input{}), reflect.TypeOf(storedskills.Output{})},
	} {
		registration, target, err := readercomponent.Compile(item.holder, "store_snapshot", item.input, item.output, resources, connector)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, registration)
		targets[i] = target
	}
	runtime, err := druntime.NewRuntime(registrations, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &snapshotReader{runtime: runtime, filesTarget: targets[0], foldersTarget: targets[1], skillsTarget: targets[2]}, nil
}

func (r *snapshotReader) files(ctx context.Context, version Version) ([]*storedfiles.SnapshotFile, error) {
	input := &storedfiles.Input{ReportId: version.ReportID, VersionNo: version.VersionNo,
		Has: &storedfiles.InputHas{ReportId: true, VersionNo: true}}
	value, err := r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.filesTarget, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedfiles.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("resource files reader returned %T", value)
	}
	for _, file := range output.Files {
		if file == nil || file.ReportId != version.ReportID || file.VersionNo != version.VersionNo {
			return nil, fmt.Errorf("resource files reader returned an incomplete or mismatched version")
		}
	}
	return output.Files, nil
}

func (r *snapshotReader) folders(ctx context.Context, version Version) ([]mcpresource.Folder, error) {
	skills := map[string][]string{}
	skillInput := &storedskills.Input{ReportId: version.ReportID, VersionNo: version.VersionNo,
		Has: &storedskills.InputHas{ReportId: true, VersionNo: true}}
	value, err := r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.skillsTarget, Input: skillInput})
	if err != nil {
		return nil, err
	}
	skillOutput, ok := value.(*storedskills.Output)
	if !ok || skillOutput == nil {
		return nil, fmt.Errorf("skill roots reader returned %T", value)
	}
	for _, skill := range skillOutput.Skills {
		if skill == nil || skill.ReportId != version.ReportID || skill.VersionNo != version.VersionNo {
			return nil, fmt.Errorf("skill roots reader returned an incomplete or mismatched version")
		}
	}
	sort.Slice(skillOutput.Skills, func(i, j int) bool {
		a, b := skillOutput.Skills[i], skillOutput.Skills[j]
		if a.Ordinal != b.Ordinal {
			return a.Ordinal < b.Ordinal
		}
		return a.SkillId < b.SkillId
	})
	for _, skill := range skillOutput.Skills {
		root := skill.SkillRoot
		if root == "" {
			root = "."
		}
		if !fs.ValidPath(root) || strings.Contains(root, "\\") {
			return nil, fmt.Errorf("invalid skill root %q", root)
		}
		skills[skill.FolderId] = append(skills[skill.FolderId], root)
	}
	folderInput := &storedfolders.Input{ReportId: version.ReportID, VersionNo: version.VersionNo,
		Has: &storedfolders.InputHas{ReportId: true, VersionNo: true}}
	value, err = r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.foldersTarget, Input: folderInput})
	if err != nil {
		return nil, err
	}
	folderOutput, ok := value.(*storedfolders.Output)
	if !ok || folderOutput == nil {
		return nil, fmt.Errorf("resource folders reader returned %T", value)
	}
	for _, folder := range folderOutput.Folders {
		if folder == nil || folder.ReportId != version.ReportID || folder.VersionNo != version.VersionNo {
			return nil, fmt.Errorf("resource folders reader returned an incomplete or mismatched version")
		}
	}
	sort.Slice(folderOutput.Folders, func(i, j int) bool {
		a, b := folderOutput.Folders[i], folderOutput.Folders[j]
		if a.Ordinal != b.Ordinal {
			return a.Ordinal < b.Ordinal
		}
		return a.FolderId < b.FolderId
	})
	var result []mcpresource.Folder
	for _, row := range folderOutput.Folders {
		folder := mcpresource.Folder{Namespace: row.Namespace, Root: row.RootPath, URIPrefix: row.UriPrefix, Skills: skills[row.FolderId]}
		if err = (spec.ResourceFolder{Namespace: row.Namespace, Root: row.RootPath, URIPrefix: row.UriPrefix}).Validate(); err != nil {
			return nil, fmt.Errorf("resource folder %s: %w", row.FolderId, err)
		}
		result = append(result, folder)
	}
	return result, nil
}

func validFile(namespace, resourcePath string) error {
	if strings.TrimSpace(namespace) == "" || strings.ContainsAny(namespace, ":/\\") {
		return fmt.Errorf("invalid resource namespace %q", namespace)
	}
	if resourcePath == "." || resourcePath != strings.TrimSpace(resourcePath) || resourcePath != path.Clean(resourcePath) || !fs.ValidPath(resourcePath) || strings.Contains(resourcePath, "\\") {
		return fmt.Errorf("invalid resource path %q", resourcePath)
	}
	return nil
}
