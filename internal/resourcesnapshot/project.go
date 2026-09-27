package resourcesnapshot

import (
	"fmt"

	"github.com/viant/datly-studio/sdk"
	files "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	folders "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	skills "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
)

// Files preserves the SDK snapshot's text content and metadata wire contract.
func Files(reportID string, versionNo int, rows []*files.SnapshotFile) ([]*sdk.ResourceFile, error) {
	var result []*sdk.ResourceFile
	for _, row := range rows {
		if row == nil || row.ReportId != reportID || row.VersionNo != versionNo {
			return nil, fmt.Errorf("resource file reader returned a mismatched row")
		}
		item := &sdk.ResourceFile{ReportID: row.ReportId, VersionNo: row.VersionNo, ResourceID: row.ResourceId,
			Namespace: row.Namespace, ResourcePath: row.ResourcePath, Content: string(row.Content),
			ContentSize: row.ContentSize, ContentSHA256: row.ContentSha256, IsBinary: row.IsBinary}
		if row.MediaType != nil {
			item.MediaType = *row.MediaType
		}
		result = append(result, item)
	}
	return result, nil
}

func Folders(reportID string, versionNo int, rows []*folders.SnapshotFolder) ([]*sdk.ResourceFolder, error) {
	var result []*sdk.ResourceFolder
	for _, row := range rows {
		if row == nil || row.ReportId != reportID || row.VersionNo != versionNo {
			return nil, fmt.Errorf("resource folder reader returned a mismatched row")
		}
		result = append(result, &sdk.ResourceFolder{ReportID: row.ReportId, VersionNo: row.VersionNo,
			FolderID: row.FolderId, Namespace: row.Namespace, RootPath: row.RootPath, URIPrefix: row.UriPrefix, Ordinal: row.Ordinal})
	}
	return result, nil
}

func Skills(reportID string, versionNo int, rows []*skills.SnapshotSkill) ([]*sdk.SkillRoot, error) {
	var result []*sdk.SkillRoot
	for _, row := range rows {
		if row == nil || row.ReportId != reportID || row.VersionNo != versionNo {
			return nil, fmt.Errorf("skill root reader returned a mismatched row")
		}
		result = append(result, &sdk.SkillRoot{ReportID: row.ReportId, VersionNo: row.VersionNo,
			SkillID: row.SkillId, FolderID: row.FolderId, SkillRoot: row.SkillRoot, Ordinal: row.Ordinal})
	}
	return result, nil
}
