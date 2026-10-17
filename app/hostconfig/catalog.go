package hostconfig

import (
	"context"
	"errors"
	"strings"

	"github.com/viant/datly/bootstrap/connector"
)

// CatalogConnector selects the single infrastructure connection. Data-source
// connections for authored components remain database-managed. The conventional
// name is supplied when omitted, without changing the caller's configuration.
func (c *Config) CatalogConnector() (connector.Config, error) {
	if c == nil || len(c.Connectors) != 1 {
		return connector.Config{}, errors.New("shared host requires one catalog connector")
	}
	result := c.Connectors[0]
	result.Name = strings.TrimSpace(result.Name)
	if result.Name == "" {
		result.Name = "studio"
	}
	if result.AliasOf != "" || result.Driver == "" || result.DSN == "" {
		return connector.Config{}, errors.New("catalog connector requires Driver and DSN")
	}
	if c.Connector != "" && c.Connector != result.Name {
		return connector.Config{}, errors.New("catalog connector selection differs from its definition")
	}
	return result, nil
}

// OpenCatalog reuses Datly's SCY resolution, pooling, and connection ownership.
// The returned set is owned by the parent host, which must close it after all
// borrowed catalog users stop. No second credential or DSN format is introduced.
func (c *Config) OpenCatalog(ctx context.Context) (*connector.Set, error) {
	catalog, err := c.CatalogConnector()
	if err != nil {
		return nil, err
	}
	return connector.Open(ctx, []connector.Config{catalog}, catalog.Name)
}
