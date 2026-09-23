package sdk

import "context"

const OperationVersionDownload = "versions.download"

type ComponentDownload struct {
	Filename  string   `json:"filename"`
	MediaType string   `json:"mediaType"`
	Archive   []byte   `json:"archive"`
	EntryDQL  string   `json:"entryDql"`
	Files     []string `json:"files"`
}

func (c versionClient) Download(ctx context.Context, reportID string, versionNo int) (*ComponentDownload, error) {
	return invoke[ComponentDownload](ctx, c.transport, OperationVersionDownload, versionIdentity{reportID, versionNo})
}
