package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/connectors/store_runtime_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type ConnectorCatalog struct{ db *sql.DB }

func NewConnectorCatalog(db *sql.DB) *ConnectorCatalog { return &ConnectorCatalog{db: db} }

func (c *ConnectorCatalog) LoadActiveConnectors(ctx context.Context) (result []*sdk.Connector, err error) {
	if c == nil || c.db == nil {
		return nil, fmt.Errorf("active connector catalog database is unavailable")
	}
	resources := resource.New()
	if err := resources.Register(stored.ConnectorDatlyResourceNamespace, stored.ConnectorDatlyResources); err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	connector := &dsql.SQLComponent{DB: c.db}
	if err := connector.RegisterConnector("studio", c.db); err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.ConnectorComponent{}), "store_runtime_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	defer func() { err = errors.Join(err, runtime.Shutdown(context.Background())) }()
	input := &stored.Input{Status: "active", Live: true, Has: &stored.InputHas{Status: true, Live: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("load active connectors: unexpected output %T", value)
	}
	for _, row := range output.Connectors {
		if row == nil || row.Name == "" || row.Status != "active" || !row.IsLive {
			return nil, fmt.Errorf("load active connectors: invalid row")
		}
		item := &sdk.Connector{
			Name: row.Name, Driver: row.Driver, OwnerID: row.OwnerId, Status: row.Status,
			Options: append([]byte(nil), row.OptionsJson...), LastTestedAt: row.LastTestedAt,
			ETag: row.Etag, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
		if row.DsnTemplate != nil {
			item.DSNTemplate = *row.DsnTemplate
			item.DSNConfigured = strings.TrimSpace(item.DSNTemplate) != ""
		}
		if row.SecretRef != nil {
			item.SecretRef = *row.SecretRef
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
		result = append(result, item)
	}
	return result, nil
}
