package sdk

import "context"

const (
	OperationVersionLoadDQL     = "versions.load_dql"
	OperationVersionLoadArchive = "versions.load_archive"
)

type LoadDQLInput struct {
	DQL   string `json:"dql"`
	Notes string `json:"notes,omitempty"`
}

// LoadArchiveInput selects one root DQL as the component source. All files,
// including the other root DQL documents, are retained as version resources.
// EntryDQL may be omitted only when the archive has one root DQL document.
type LoadArchiveInput struct {
	Archive  []byte `json:"archive"`
	Format   string `json:"format"`
	EntryDQL string `json:"entryDql,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type DQLLoadResult struct {
	Version  *ReportVersion `json:"version"`
	EntryDQL string         `json:"entryDql"`
	Entries  []string       `json:"entries"`
	Files    []string       `json:"files"`
}

func (c versionClient) LoadDQL(ctx context.Context, reportID string, input LoadDQLInput) (*DQLLoadResult, error) {
	return invoke[DQLLoadResult](ctx, c.transport, OperationVersionLoadDQL, struct {
		ReportID string       `json:"reportId"`
		Input    LoadDQLInput `json:"input"`
	}{reportID, input})
}
func (c versionClient) LoadArchive(ctx context.Context, reportID string, input LoadArchiveInput) (*DQLLoadResult, error) {
	return invoke[DQLLoadResult](ctx, c.transport, OperationVersionLoadArchive, struct {
		ReportID string           `json:"reportId"`
		Input    LoadArchiveInput `json:"input"`
	}{reportID, input})
}
