// Package resources materializes versioned Studio resource rows as immutable
// Datly resource stores and explicitly declared MCP folder/skill plans.
package resources

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing/fstest"

	bindresource "github.com/viant/bindly/resource"
	mcpresource "github.com/viant/datly/mcp/resource"
	"github.com/viant/datly/spec"
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
	result := &Loaded{Store: bindresource.New(), ByVersion: map[Version]*bindresource.Store{}, ResourceReports: map[string]string{}}
	namespaces := map[string]fstest.MapFS{}
	for _, version := range versions {
		if strings.TrimSpace(version.ReportID) == "" || version.VersionNo <= 0 {
			return nil, fmt.Errorf("resource version identity is required")
		}
		local := fstest.MapFS{}
		rows, err := db.QueryContext(ctx, `SELECT namespace,resource_path,content FROM report_resource_files WHERE report_id=? AND version_no=? ORDER BY namespace,resource_path`, version.ReportID, version.VersionNo)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var namespace, resourcePath string
			var content []byte
			if err = rows.Scan(&namespace, &resourcePath, &content); err != nil {
				rows.Close()
				return nil, err
			}
			if err = validFile(namespace, resourcePath); err != nil {
				rows.Close()
				return nil, err
			}
			if _, exists := local[resourcePath]; exists {
				rows.Close()
				return nil, fmt.Errorf("duplicate default resource path %q for report %s", resourcePath, version.ReportID)
			}
			item := &fstest.MapFile{Data: append([]byte(nil), content...)}
			local[resourcePath] = item
			group, registered := namespaces[namespace]
			if !registered {
				group = fstest.MapFS{}
			}
			if _, exists := group[resourcePath]; exists {
				rows.Close()
				return nil, fmt.Errorf("duplicate resource namespace/path %s:%s", namespace, resourcePath)
			}
			group[resourcePath] = item
			namespaces[namespace] = group
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
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
		folders, err := loadFolders(ctx, db, version)
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

func loadFolders(ctx context.Context, db *sql.DB, version Version) ([]mcpresource.Folder, error) {
	skills := map[string][]string{}
	skillRows, err := db.QueryContext(ctx, `SELECT folder_id,skill_root FROM report_skill_roots WHERE report_id=? AND version_no=? ORDER BY ordinal,skill_id`, version.ReportID, version.VersionNo)
	if err != nil {
		return nil, err
	}
	for skillRows.Next() {
		var folderID, root string
		if err = skillRows.Scan(&folderID, &root); err != nil {
			skillRows.Close()
			return nil, err
		}
		if root == "" {
			root = "."
		}
		if !fs.ValidPath(root) || strings.Contains(root, "\\") {
			skillRows.Close()
			return nil, fmt.Errorf("invalid skill root %q", root)
		}
		skills[folderID] = append(skills[folderID], root)
	}
	if err = skillRows.Err(); err != nil {
		skillRows.Close()
		return nil, err
	}
	skillRows.Close()
	folderRows, err := db.QueryContext(ctx, `SELECT folder_id,namespace,root_path,uri_prefix FROM report_resource_folders WHERE report_id=? AND version_no=? ORDER BY ordinal,folder_id`, version.ReportID, version.VersionNo)
	if err != nil {
		return nil, err
	}
	defer folderRows.Close()
	var result []mcpresource.Folder
	for folderRows.Next() {
		var id, namespace, root, uri string
		if err = folderRows.Scan(&id, &namespace, &root, &uri); err != nil {
			return nil, err
		}
		folder := mcpresource.Folder{Namespace: namespace, Root: root, URIPrefix: uri, Skills: skills[id]}
		if err = (spec.ResourceFolder{Namespace: namespace, Root: root, URIPrefix: uri}).Validate(); err != nil {
			return nil, fmt.Errorf("resource folder %s: %w", id, err)
		}
		result = append(result, folder)
	}
	return result, folderRows.Err()
}

func validFile(namespace, resourcePath string) error {
	if strings.TrimSpace(namespace) == "" || strings.ContainsAny(namespace, ":/\\") {
		return fmt.Errorf("invalid resource namespace %q", namespace)
	}
	resourcePath = path.Clean(strings.TrimSpace(resourcePath))
	if resourcePath == "." || !fs.ValidPath(resourcePath) || strings.Contains(resourcePath, "\\") {
		return fmt.Errorf("invalid resource path %q", resourcePath)
	}
	return nil
}
