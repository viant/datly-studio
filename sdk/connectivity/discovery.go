package connectivity

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/sqlx/io/config"
	"github.com/viant/sqlx/metadata"
	"github.com/viant/sqlx/metadata/info"
)

// Discover uses the same authorized connector and driver metadata as table
// browsing. Unsupported catalog enumeration retains the actual session catalog.
func (c SQLCatalog) Discover(ctx context.Context, connector *sdk.Connector, input sdk.DatabaseDiscoveryInput) (*sdk.DatabaseDiscovery, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	scope, err := resolveDiscoveryScope(connector, input.Catalog, input.Schema)
	if err != nil {
		return nil, err
	}
	db, err := openDiscoveryDB(ctx, connector, c.Secrets, c.Const, scope)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	meta := metadata.New()
	product, err := meta.DetectProduct(ctx, db)
	if err != nil {
		return nil, err
	}
	session, err := config.Session(ctx, db)
	if err != nil {
		return nil, err
	}
	scope.apply(session)
	result := &sdk.DatabaseDiscovery{Driver: connector.Driver, Product: product.Name, Catalog: session.Catalog, Schema: session.Schema, Catalogs: []sdk.DatabaseCatalog{}, Schemas: []sdk.DatabaseSchema{}}
	if !strings.EqualFold(connector.Driver, "bigquery") {
		_ = meta.Info(ctx, db, info.KindVersion, &result.Version)
	}
	var catalogs []struct {
		Name string `sqlx:"CATALOG_NAME"`
	}
	if err = meta.Info(ctx, db, info.KindCatalogs, &catalogs); err != nil {
		result.Warnings = append(result.Warnings, "Catalog enumeration is unavailable; add projects/catalogs to the lookup list.")
	}
	seen := map[string]bool{}
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			result.Catalogs = append(result.Catalogs, sdk.DatabaseCatalog{Name: name})
		}
	}
	add(session.Catalog)
	for _, item := range catalogs {
		add(item.Name)
	}
	schemas, err := c.Schemas(ctx, connector, sdk.SchemaCatalogInput{Catalog: scope.Catalog})
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		result.Warnings = append(result.Warnings, "Schema discovery is unavailable for this catalog; select or add a schema/dataset manually.")
	} else {
		result.Schemas = schemas.Items
		for _, item := range schemas.Items {
			add(item.Catalog)
		}
	}
	sort.Slice(result.Catalogs, func(i, j int) bool { return result.Catalogs[i].Name < result.Catalogs[j].Name })
	return result, nil
}
