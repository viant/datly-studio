package sdk

import (
	"context"
	"time"
)

const OperationConnectorSearch = "connectors.search"

type SchemaSearchInput struct {
	Catalog          string   `json:"catalog,omitempty"`
	Schema           string   `json:"schema"`
	Table            string   `json:"table,omitempty"`
	Search           string   `json:"search,omitempty"`
	ColumnTerms      []string `json:"columnTerms,omitempty"`
	DescriptionTerms []string `json:"descriptionTerms,omitempty"`
	Limit            int      `json:"limit,omitempty"`
	Cursor           string   `json:"cursor,omitempty"`
	// MetadataCursor continues provider discovery, not result paging. Results
	// describe only this metadata page until metadataComplete is true.
	MetadataCursor string `json:"metadataCursor,omitempty"`
	Refresh        bool   `json:"refresh,omitempty"`
}
type SchemaSearchMatch struct {
	Table       DatabaseTable `json:"table"`
	Column      string        `json:"column,omitempty"`
	Description string        `json:"description,omitempty"`
}
type SchemaSearchResult struct {
	Items              []SchemaSearchMatch `json:"items"`
	NextCursor         string              `json:"nextCursor,omitempty"`
	NextMetadataCursor string              `json:"nextMetadataCursor,omitempty"`
	MetadataComplete   bool                `json:"metadataComplete"`
	SnapshotID         string              `json:"snapshotId"`
	ExpiresAt          time.Time           `json:"expiresAt"`
	Total              int                 `json:"total"`
}

// SchemaSearchOptions are host-owned lifecycle/budget settings, never tool input.
type SchemaSearchOptions struct {
	TTL, BuildTimeout                         time.Duration
	MaxEntries, MaxBytes, MaxConcurrentBuilds int
}

// SchemaSearcher receives an already-authorized connector and verified principal.
// It never caches connector grants or decides authorization.
type SchemaSearcher interface {
	Search(context.Context, *Connector, Principal, SchemaSearchInput) (*SchemaSearchResult, error)
}
