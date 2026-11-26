// Package resources materializes versioned Studio resource rows as immutable
// Datly resource stores and explicitly declared MCP folder/skill plans.
package resources

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing/fstest"

	bindresource "github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/internal/resourceartifact"
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
	Fingerprints    map[Version]string
	Store           *bindresource.Store
	ByVersion       map[Version]*bindresource.Store
	Folders         []mcpresource.Folder
	ResourceReports map[string]string
}

// Load reads one immutable resource generation. Namespaces are application-wide
// authority: two active components cannot register different files under the
// same namespace.
func Load(ctx context.Context, db *sql.DB, versions []Version) (*Loaded, error) {
	return load(ctx, db, versions, nil, nil)
}

// Loader retains only actor-independent native read plans for one DB lifetime.
// Every Load invokes them with fresh context/inputs and reads current rows.
type Loader struct {
	db     *sql.DB
	reader *snapshotReader
}

func NewLoader(db *sql.DB) (*Loader, error) {
	if db == nil {
		return nil, fmt.Errorf("Studio database is required")
	}
	reader, err := newSnapshotReader(db)
	if err != nil {
		return nil, err
	}
	return &Loader{db: db, reader: reader}, nil
}
func (l *Loader) Close(ctx context.Context) error {
	if l == nil || l.reader == nil {
		return nil
	}
	return l.reader.runtime.Shutdown(ctx)
}
func (l *Loader) Load(ctx context.Context, versions []Version) (*Loaded, error) {
	if l == nil || l.reader == nil {
		return nil, fmt.Errorf("resource metadata reader is unavailable")
	}
	return load(ctx, l.db, versions, l.reader, nil)
}

// LoadRecovering reuses immutable snapshot loading while allowing the host to
// omit invalid version-owned resources. Database and identity failures remain fatal.
// The callback is preparation-only and must not perform an access decision.
func LoadRecovering(ctx context.Context, db *sql.DB, versions []Version, invalid func(Version, error) bool) (*Loaded, error) {
	return load(ctx, db, versions, nil, invalid)
}

type invalidResource struct{ cause error }

func (e *invalidResource) Error() string { return e.cause.Error() }
func (e *invalidResource) Unwrap() error { return e.cause }

func load(ctx context.Context, db *sql.DB, versions []Version, reader *snapshotReader, invalid func(Version, error) bool) (*Loaded, error) {
	if db == nil {
		return nil, fmt.Errorf("Studio database is required")
	}
	for _, version := range versions {
		if strings.TrimSpace(version.ReportID) == "" || version.VersionNo <= 0 {
			return nil, fmt.Errorf("resource version identity is required")
		}
	}
	result := &Loaded{Fingerprints: map[Version]string{}, Store: bindresource.New(), ByVersion: map[Version]*bindresource.Store{}, ResourceReports: map[string]string{}}
	if len(versions) == 0 {
		return result, nil
	}
	if reader == nil {
		var err error
		reader, err = newSnapshotReader(db)
		if err != nil {
			return nil, err
		}
		defer reader.runtime.Shutdown(context.Background())
	}
	// Read each immutable version once, then merge only accepted resources. A bad
	// version must never leave partially registered files in another reader's store.
	type snapshot struct {
		files   []resourceFingerprintFile
		folders []mcpresource.Folder
	}
	snapshots := map[Version]*snapshot{}
	rejected := map[Version]bool{}
	reject := func(version Version, cause error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if rejected[version] {
			return nil
		}
		if invalid == nil || !invalid(version, cause) {
			return cause
		}
		rejected[version] = true
		return nil
	}
	fileOwners := map[string]Version{}
	folderOwners := map[string]Version{}
	for _, version := range versions {
		files, err := reader.files(ctx, version)
		if err != nil {
			return nil, err
		}
		local := map[string]bool{}
		item := &snapshot{}
		var invalidErr error
		for _, file := range files {
			if file.Namespace == resourceartifact.Namespace {
				continue
			}
			if err = validFile(file.Namespace, file.ResourcePath); err != nil {
				invalidErr = err
				break
			}
			if local[file.ResourcePath] {
				invalidErr = fmt.Errorf("duplicate default resource path %q for report %s", file.ResourcePath, version.ReportID)
				break
			}
			local[file.ResourcePath] = true
			item.files = append(item.files, resourceFingerprintFile{Namespace: file.Namespace, Path: file.ResourcePath, Content: append([]byte(nil), file.Content...)})
		}
		if invalidErr != nil {
			if err = reject(version, invalidErr); err != nil {
				return nil, err
			}
			continue
		}
		item.folders, err = reader.folders(ctx, version)
		if err != nil {
			var malformed *invalidResource
			if !errors.As(err, &malformed) {
				return nil, err
			}
			if err = reject(version, err); err != nil {
				return nil, err
			}
			continue
		}
		snapshots[version] = item
		for _, file := range item.files {
			key := file.Namespace + ":" + file.Path
			if owner, exists := fileOwners[key]; exists {
				cause := fmt.Errorf("duplicate resource namespace/path %s", key)
				if err = reject(version, cause); err != nil {
					return nil, err
				}
				if err = reject(owner, cause); err != nil {
					return nil, err
				}
			} else {
				fileOwners[key] = version
			}
		}
		for _, folder := range item.folders {
			if owner, exists := folderOwners[folder.URIPrefix]; exists && owner.ReportID != version.ReportID {
				cause := fmt.Errorf("resource URI prefix %q is owned by both reports %s and %s", folder.URIPrefix, owner.ReportID, version.ReportID)
				if err = reject(version, cause); err != nil {
					return nil, err
				}
				if err = reject(owner, cause); err != nil {
					return nil, err
				}
			} else {
				folderOwners[folder.URIPrefix] = version
			}
		}
	}
	namespaces := map[string]fstest.MapFS{}
	for _, version := range versions {
		if rejected[version] {
			continue
		}
		item := snapshots[version]
		local := fstest.MapFS{}
		for _, file := range item.files {
			entry := &fstest.MapFile{Data: file.Content}
			local[file.Path] = entry
			if namespaces[file.Namespace] == nil {
				namespaces[file.Namespace] = fstest.MapFS{}
			}
			namespaces[file.Namespace][file.Path] = entry
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
		for _, folder := range item.folders {
			result.ResourceReports[folder.URIPrefix] = version.ReportID
		}
		result.Folders = append(result.Folders, item.folders...)
		sort.Slice(item.files, func(i, j int) bool {
			if item.files[i].Namespace == item.files[j].Namespace {
				return item.files[i].Path < item.files[j].Path
			}
			return item.files[i].Namespace < item.files[j].Namespace
		})
		sort.Slice(item.folders, func(i, j int) bool { return item.folders[i].URIPrefix < item.folders[j].URIPrefix })
		raw, err := json.Marshal(struct {
			Files   []resourceFingerprintFile
			Folders []mcpresource.Folder
		}{item.files, item.folders})
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256(raw)
		result.Fingerprints[version] = hex.EncodeToString(hash[:])
	}
	for namespace, files := range namespaces {
		if err := result.Store.Register(namespace, files); err != nil {
			return nil, err
		}
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
			return nil, &invalidResource{fmt.Errorf("invalid skill root %q", root)}
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
			return nil, &invalidResource{fmt.Errorf("resource folder %s: %w", row.FolderId, err)}
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

type resourceFingerprintFile struct {
	Namespace, Path string
	Content         []byte
}
