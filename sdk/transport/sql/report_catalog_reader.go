package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/reports/catalogpredicate"
	stored "github.com/viant/datly-studio/studio/reports/store_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type reportCatalogRequest struct {
	ID, Query, Namespace, Status, OwnerID, ConnectorName string
	Limit, Offset                                        int
	// Unscoped is only for server-owned mutations already authorized by the
	// SDK operation; public get/list requests always keep principal scope.
	Unscoped bool
}

func (t *Transport) readReportCatalog(ctx context.Context, request reportCatalogRequest) ([]*sdk.Report, error) {
	return t.readReportCatalogTx(ctx, nil, request)
}

func (t *Transport) readReportCatalogTx(ctx context.Context, tx *sql.Tx, request reportCatalogRequest) ([]*sdk.Report, error) {
	resources := resource.New()
	if err := resources.Register(stored.ReportDatlyResourceNamespace, stored.ReportDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/reports/catalogpredicate",
		Types: []reflect.Type{reflect.TypeFor[catalogpredicate.ReportCatalogRead]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.ReportComponent{}), "store_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	principal, scoped := sdk.PrincipalFromContext(ctx)
	if request.Unscoped {
		scoped = false
	}
	input := &stored.Input{Id: request.ID, Query: request.Query, Namespace: request.Namespace,
		Status: request.Status, OwnerId: request.OwnerID, ConnectorName: request.ConnectorName,
		Subject: principal.Subject, Scoped: scoped, Limit: request.Limit, Offset: request.Offset,
		Has: &stored.InputHas{Id: request.ID != "", Query: request.Query != "", Namespace: request.Namespace != "",
			Status: request.Status != "", OwnerId: request.OwnerID != "", ConnectorName: request.ConnectorName != "",
			Subject: true, Scoped: true, Limit: true, Offset: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("report catalog returned %T", value)
	}
	if len(output.Reports) > request.Limit {
		return nil, fmt.Errorf("report catalog exceeded requested limit")
	}
	result := make([]*sdk.Report, 0, len(output.Reports))
	for _, row := range output.Reports {
		if row == nil || request.ID != "" && row.Id != request.ID {
			return nil, fmt.Errorf("report catalog returned a mismatched row")
		}
		item := &sdk.Report{ID: row.Id, Namespace: row.Namespace, Slug: row.Slug, Title: row.Title,
			OwnerID: row.OwnerId, OwnerPackage: sdk.OwnerPackageSegment(row.OwnerId), Status: row.Status,
			DefaultConnectorName: row.DefaultConnectorName, ComponentScope: row.ComponentScope,
			ComponentName: row.ComponentName, CurrentDraftVersion: row.CurrentDraftVersion,
			ETag: row.Etag, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		if row.Description != nil {
			item.Description = *row.Description
		}
		result = append(result, item)
	}
	return result, nil
}
