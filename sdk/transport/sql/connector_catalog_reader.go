package sqltransport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/connectors/store_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type connectorCatalogRequest struct {
	Name, Query, Status, OwnerID, Driver string
	Limit, Offset                        int
}

func (t *Transport) readConnectorCatalog(ctx context.Context, request connectorCatalogRequest) (items []*sdk.Connector, err error) {
	resources := resource.New()
	if err := resources.Register(stored.ConnectorDatlyResourceNamespace, stored.ConnectorDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.ConnectorComponent{}), "store_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, runtime.Shutdown(context.Background())) }()
	principal, scoped := sdk.PrincipalFromContext(ctx)
	search := ""
	if query := strings.TrimSpace(request.Query); query != "" {
		search = "%" + strings.ToLower(query) + "%"
	}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Name: request.Name, Query: search, Status: request.Status,
			OwnerId: request.OwnerID, Driver: request.Driver, Subject: principal.Subject, Scoped: scoped,
			PageLimit: request.Limit, PageOffset: request.Offset,
			Has: &stored.InputHas{Name: true, Query: true, Status: true, OwnerId: true,
				Driver: true, Subject: true, Scoped: true, PageLimit: true, PageOffset: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok || len(output.Connectors) > request.Limit {
		return nil, fmt.Errorf("connector catalog returned %T with unexpected rows", value)
	}
	items = make([]*sdk.Connector, 0, len(output.Connectors))
	for _, row := range output.Connectors {
		if row == nil || request.Name != "" && row.Name != request.Name {
			return nil, errors.New("connector catalog returned a mismatched row")
		}
		item := &sdk.Connector{Name: row.Name, Driver: row.Driver, OwnerID: row.OwnerId,
			Status: row.Status, Options: append([]byte(nil), row.OptionsJson...),
			ETag: row.Etag, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			LastTestedAt: row.LastTestedAt}
		if row.DsnTemplate != nil {
			item.DSNTemplate = *row.DsnTemplate
			item.DSNConfigured = strings.TrimSpace(item.DSNTemplate) != ""
		}
		if row.SecretRef != nil {
			item.SecretRef = *row.SecretRef
			item.SecretConfigured = strings.TrimSpace(item.SecretRef) != ""
		}
		if row.Description != nil {
			item.Description = *row.Description
		}
		if row.LastTestStatus != nil {
			item.LastTestStatus = *row.LastTestStatus
		}
		if row.LastTestErrorCode != nil {
			item.LastTestErrorCode = *row.LastTestErrorCode
		}
		items = append(items, item)
	}
	return items, nil
}
