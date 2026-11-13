package connectivity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/viant/afs"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/internal/schemasearch"
	"github.com/viant/datly-studio/sdk"
)

// SchemaSearch owns immutable snapshots for one host. Its caller authorizes
// connector access on every operation before passing server-held configuration.
type SchemaSearch struct {
	catalog SQLCatalog
	cache   *schemasearch.Cache
}

func NewSchemaSearch(catalog SQLCatalog, options sdk.SchemaSearchOptions) *SchemaSearch {
	return &SchemaSearch{catalog: catalog, cache: schemasearch.NewCache(schemasearch.CacheOptions{TTL: options.TTL, BuildTimeout: options.BuildTimeout, MaxEntries: options.MaxEntries, MaxBytes: options.MaxBytes, MaxConcurrentBuilds: options.MaxConcurrentBuilds})}
}
func (s *SchemaSearch) Close() { s.cache.Close() }
func (s *SchemaSearch) Search(ctx context.Context, connector *sdk.Connector, principal sdk.Principal, input sdk.SchemaSearchInput) (*sdk.SchemaSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if principal.Subject == "" || connector == nil {
		return nil, fmt.Errorf("schema search requires an authorized connector and verified principal")
	}
	scope, err := resolveDiscoveryScope(connector, input.Catalog, input.Schema)
	if err != nil {
		return nil, err
	}
	if scope.Schema == "" {
		return nil, fmt.Errorf("schema search requires a database/dataset")
	}
	resolved, err := connectorsecret.Resolve(ctx, s.catalog.Secrets, connector.DSNTemplate, connector.SecretRef, s.catalog.Const)
	if err != nil {
		return nil, err
	}
	fingerprint, err := schemaCredentialFingerprint(ctx, connector.Driver, resolved)
	if err != nil {
		return nil, err
	}
	prepared := *connector
	prepared.DSNTemplate = resolved
	prepared.SecretRef = ""
	identity, _ := json.Marshal([]string{connector.OwnerID, connector.Name, connector.Driver, string(connector.Options)})
	key := schemasearch.Scope{CallerIdentity: principal.Subject, ConnectorIdentity: string(identity), CredentialIdentity: fingerprint, ConnectorRevision: connector.ETag, Catalog: scope.Catalog, Schema: scope.Schema, Table: input.Table, MetadataCursor: input.MetadataCursor}
	snapshot, err := s.cache.Get(ctx, key, input.Refresh, func(loadCtx context.Context) (*schemasearch.Metadata, error) {
		page, e := s.catalog.ColumnMetadataPage(loadCtx, &prepared, SchemaMetadataInput{Catalog: scope.Catalog, Schema: scope.Schema, Table: input.Table, PageToken: input.MetadataCursor})
		if e != nil {
			return nil, e
		}
		return &schemasearch.Metadata{Tables: page.Tables, Complete: input.MetadataCursor == "" && page.Complete, Continuation: page.NextPageToken}, nil
	})
	if err != nil {
		return nil, err
	}
	query := schemasearch.Query{Search: input.Search, ColumnTerms: input.ColumnTerms, DescriptionTerms: input.DescriptionTerms, Limit: input.Limit, Cursor: input.Cursor}
	if input.Table != "" {
		query.Table = &sdk.DatabaseTable{Catalog: scope.Catalog, Schema: scope.Schema, Name: input.Table}
	}
	page, err := snapshot.Index.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return &sdk.SchemaSearchResult{Items: page.Items, NextCursor: page.NextCursor, Total: page.Total, NextMetadataCursor: snapshot.Continuation, MetadataComplete: snapshot.Complete, SnapshotID: snapshot.ID, ExpiresAt: snapshot.ExpiresAt}, nil
}

// The resolved DSN changes on connector-secret rotation. Drivers with an external
// credential file also fingerprint that current resource through the existing
// filesystem owner, rather than using its unchanged URL as credential identity.
func schemaCredentialFingerprint(ctx context.Context, driver, dsn string) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write([]byte(driver))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(dsn))
	if resolveDriver(strings.ToLower(strings.TrimSpace(driver))) == "bigquery" {
		parsed, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("invalid metadata connector URL")
		}
		reference := parsed.Query().Get("credURL")
		if reference == "" && parsed.Query().Get("credJSON") == "" {
			reference = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if reference == "" {
				return "", fmt.Errorf("cached schema discovery requires an identifiable source credential")
			}
		}
		if reference != "" {
			data, err := afs.New().DownloadWithURL(ctx, reference)
			if err != nil {
				return "", fmt.Errorf("read metadata credential identity: %w", err)
			}
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write(data)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
