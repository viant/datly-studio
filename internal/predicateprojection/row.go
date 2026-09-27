package predicateprojection

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/sdk"
	storedreader "github.com/viant/datly-studio/studio/authorization_predicates/store_read"
)

type Links interface {
	Contains(packagePath, typeName string) bool
}

// FromRow projects the same governed SDK DTO for the generic and native
// transports, including the deployment-linked predicate flag.
func FromRow(row *storedreader.StoredAuthorizationPredicate, links Links) (*sdk.AuthorizationPredicate, error) {
	if row == nil || row.CreatedAt == nil || row.UpdatedAt == nil {
		return nil, fmt.Errorf("authorization predicate reader returned an incomplete row")
	}
	value := &sdk.AuthorizationPredicate{
		Name: row.Name, Title: row.Title, PackagePath: row.PackagePath, TypeName: row.TypeName,
		OwnerID: row.OwnerId, Status: row.Status, ETag: row.Etag,
		CreatedAt: *row.CreatedAt, UpdatedAt: *row.UpdatedAt,
	}
	if row.Description != nil {
		value.Description = *row.Description
	}
	if row.SqlScopeJson != nil && strings.TrimSpace(*row.SqlScopeJson) != "" {
		var metadata sdk.AuthorizationPredicateSQLMetadata
		if err := json.Unmarshal([]byte(*row.SqlScopeJson), &metadata); err != nil {
			return nil, err
		}
		value.Alias, value.Columns = metadata.Alias, metadata.Columns
	}
	if links != nil {
		value.Linked = links.Contains(value.PackagePath, value.TypeName)
	}
	return value, nil
}
