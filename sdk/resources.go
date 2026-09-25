package sdk

import "context"

const (
	OperationResourcesGet          = "resources.get"
	OperationResourcesUpsertFile   = "resources.upsert_file"
	OperationResourcesDeleteFile   = "resources.delete_file"
	OperationResourcesUpsertFolder = "resources.upsert_folder"
	OperationResourcesDeleteFolder = "resources.delete_folder"
	OperationResourcesUpsertSkill  = "resources.upsert_skill"
	OperationResourcesDeleteSkill  = "resources.delete_skill"
)

type ResourceFile struct {
	ReportID               string `json:"reportId"`
	VersionNo              int    `json:"versionNo"`
	ResourceID             string `json:"resourceId"`
	Namespace              string `json:"namespace"`
	ResourcePath           string `json:"resourcePath"`
	MediaType              string `json:"mediaType,omitempty"`
	Content                string `json:"content"`
	ContentSize            int64  `json:"contentSize"`
	ContentSHA256          string `json:"contentSha256"`
	IsBinary               bool   `json:"isBinary"`
	ExpectedSourceRevision int64  `json:"expectedSourceRevision,omitempty"`
}

type ResourceFolder struct {
	ReportID               string `json:"reportId"`
	VersionNo              int    `json:"versionNo"`
	FolderID               string `json:"folderId"`
	Namespace              string `json:"namespace"`
	RootPath               string `json:"rootPath"`
	URIPrefix              string `json:"uriPrefix"`
	Ordinal                int    `json:"ordinal"`
	ExpectedSourceRevision int64  `json:"expectedSourceRevision,omitempty"`
}

type SkillRoot struct {
	ReportID               string `json:"reportId"`
	VersionNo              int    `json:"versionNo"`
	SkillID                string `json:"skillId"`
	FolderID               string `json:"folderId"`
	SkillRoot              string `json:"skillRoot"`
	Ordinal                int    `json:"ordinal"`
	ExpectedSourceRevision int64  `json:"expectedSourceRevision,omitempty"`
}

// ResourceDeleteInput identifies a versioned resource mutation. Mutations
// require the exact ExpectedSourceRevision of the target report version.
type ResourceDeleteInput struct {
	ReportID               string `json:"reportId"`
	VersionNo              int    `json:"versionNo"`
	ResourceID             string `json:"resourceId,omitempty"`
	FolderID               string `json:"folderId,omitempty"`
	SkillID                string `json:"skillId,omitempty"`
	ExpectedSourceRevision int64  `json:"expectedSourceRevision,omitempty"`
}

type ResourceSnapshot struct {
	Files   []*ResourceFile   `json:"files"`
	Folders []*ResourceFolder `json:"folders"`
	Skills  []*SkillRoot      `json:"skills"`
	Version *ReportVersion    `json:"version,omitempty"`
}

type ResourceService interface {
	Get(context.Context, string, int) (*ResourceSnapshot, error)
	UpsertFile(context.Context, ResourceFile) (*ResourceSnapshot, error)
	// Deprecated: use DeleteFileWithRevision; resource mutations require an exact revision.
	DeleteFile(context.Context, string, int, string) (*ResourceSnapshot, error)
	DeleteFileWithRevision(context.Context, ResourceDeleteInput) (*ResourceSnapshot, error)
	UpsertFolder(context.Context, ResourceFolder) (*ResourceSnapshot, error)
	// Deprecated: use DeleteFolderWithRevision; resource mutations require an exact revision.
	DeleteFolder(context.Context, string, int, string) (*ResourceSnapshot, error)
	DeleteFolderWithRevision(context.Context, ResourceDeleteInput) (*ResourceSnapshot, error)
	UpsertSkill(context.Context, SkillRoot) (*ResourceSnapshot, error)
	// Deprecated: use DeleteSkillWithRevision; resource mutations require an exact revision.
	DeleteSkill(context.Context, string, int, string) (*ResourceSnapshot, error)
	DeleteSkillWithRevision(context.Context, ResourceDeleteInput) (*ResourceSnapshot, error)
}

type resourceClient struct{ transport Transport }

func (c resourceClient) Get(ctx context.Context, reportID string, versionNo int) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesGet, map[string]any{"reportId": reportID, "versionNo": versionNo})
}
func (c resourceClient) UpsertFile(ctx context.Context, value ResourceFile) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesUpsertFile, value)
}
func (c resourceClient) DeleteFile(ctx context.Context, reportID string, versionNo int, resourceID string) (*ResourceSnapshot, error) {
	return c.DeleteFileWithRevision(ctx, ResourceDeleteInput{ReportID: reportID, VersionNo: versionNo, ResourceID: resourceID})
}
func (c resourceClient) DeleteFileWithRevision(ctx context.Context, input ResourceDeleteInput) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesDeleteFile, input)
}
func (c resourceClient) UpsertFolder(ctx context.Context, value ResourceFolder) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesUpsertFolder, value)
}
func (c resourceClient) DeleteFolder(ctx context.Context, reportID string, versionNo int, folderID string) (*ResourceSnapshot, error) {
	return c.DeleteFolderWithRevision(ctx, ResourceDeleteInput{ReportID: reportID, VersionNo: versionNo, FolderID: folderID})
}
func (c resourceClient) DeleteFolderWithRevision(ctx context.Context, input ResourceDeleteInput) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesDeleteFolder, input)
}
func (c resourceClient) UpsertSkill(ctx context.Context, value SkillRoot) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesUpsertSkill, value)
}
func (c resourceClient) DeleteSkill(ctx context.Context, reportID string, versionNo int, skillID string) (*ResourceSnapshot, error) {
	return c.DeleteSkillWithRevision(ctx, ResourceDeleteInput{ReportID: reportID, VersionNo: versionNo, SkillID: skillID})
}
func (c resourceClient) DeleteSkillWithRevision(ctx context.Context, input ResourceDeleteInput) (*ResourceSnapshot, error) {
	return invoke[ResourceSnapshot](ctx, c.transport, OperationResourcesDeleteSkill, input)
}
