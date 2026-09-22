// Package sdk is the only application API intended for Forge and other Studio
// clients. It deliberately exposes no Datly, SQL, store, or control types.
package sdk

import (
	"context"
	"fmt"
)

// Transport executes one stable SDK operation. Implementations may be HTTP,
// in-process, or test transports; operation payloads are SDK DTOs only.
type Transport interface {
	Invoke(ctx context.Context, operation string, input, output any) error
}

type Client interface {
	Connectors() ConnectorService
	Namespaces() NamespaceService
	AuthorizationPredicates() AuthorizationPredicateService
	Reports() ReportService
	Versions() VersionService
	Preview() PreviewService
	Catalog() CatalogService
	Publications() PublicationService
	Resources() ResourceService
	ACL() ACLService
	Runtime() RuntimeService
}

type client struct{ transport Transport }

func NewClient(transport Transport) (Client, error) {
	if transport == nil {
		return nil, fmt.Errorf("studio SDK transport is required")
	}
	return &client{transport: transport}, nil
}

func (c *client) Connectors() ConnectorService     { return connectorClient{c.transport} }
func (c *client) Namespaces() NamespaceService     { return namespaceClient{c.transport} }
func (c *client) AuthorizationPredicates() AuthorizationPredicateService { return authorizationPredicateClient{c.transport} }
func (c *client) Reports() ReportService           { return reportClient{c.transport} }
func (c *client) Versions() VersionService         { return versionClient{c.transport} }
func (c *client) Preview() PreviewService          { return previewClient{c.transport} }
func (c *client) Catalog() CatalogService          { return catalogClient{c.transport} }
func (c *client) Publications() PublicationService { return publicationClient{c.transport} }
func (c *client) Resources() ResourceService       { return resourceClient{c.transport} }
func (c *client) ACL() ACLService                  { return aclClient{c.transport} }
func (c *client) Runtime() RuntimeService          { return runtimeClient{c.transport} }

func invoke[T any](ctx context.Context, transport Transport, operation string, input any) (*T, error) {
	result := new(T)
	if err := transport.Invoke(ctx, operation, input, result); err != nil {
		return nil, err
	}
	return result, nil
}

func invokeEmpty(ctx context.Context, transport Transport, operation string, input any) error {
	return transport.Invoke(ctx, operation, input, nil)
}
