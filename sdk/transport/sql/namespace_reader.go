package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"reflect"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/namespaces/accesspredicate"
	read "github.com/viant/datly-studio/studio/namespaces/store_read"
	usage "github.com/viant/datly-studio/studio/namespaces/store_usage"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type namespaceStoreReader struct {
	runtime *druntime.Runtime
	read    dexec.ComponentTarget
	usage   dexec.ComponentTarget
}

// The reader is request-scoped; every caller shuts down its runtime after use.
func (t *Transport) newNamespaceStoreReader() (*namespaceStoreReader, error) {
	resources := resource.New()
	if err := resources.Register(read.NamespaceDatlyResourceNamespace, read.NamespaceDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(usage.UsageDatlyResourceNamespace, usage.UsageDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/namespaces/accesspredicate", Types: []reflect.Type{reflect.TypeFor[accesspredicate.NamespaceDirectory]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	readRegistration, readTarget, err := readercomponent.Compile(reflect.TypeOf(read.NamespaceComponent{}), "store_read",
		reflect.TypeOf(read.Input{}), reflect.TypeOf(read.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	usageRegistration, usageTarget, err := readercomponent.Compile(reflect.TypeOf(usage.UsageComponent{}), "store_usage",
		reflect.TypeOf(usage.Input{}), reflect.TypeOf(usage.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	readRegistration.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{readRegistration, usageRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &namespaceStoreReader{runtime: runtime, read: readTarget, usage: usageTarget}, nil
}

func (t *Transport) readNamespaces(ctx context.Context, name, query, status, namespaceID string, limit, offset int) ([]*sdk.Namespace, error) {
	reader, err := t.newNamespaceStoreReader()
	if err != nil {
		return nil, err
	}
	defer reader.runtime.Shutdown(context.Background())
	principal, scoped := sdk.PrincipalFromContext(ctx)
	search := ""
	if value := strings.TrimSpace(query); value != "" {
		search = "%" + strings.ToLower(value) + "%"
	}
	input := &read.Input{NamespaceId: namespaceID, Name: name, OwnerId: "", Query: search, Status: status, Subject: principal.Subject, Scoped: scoped,
		PageLimit: limit, PageOffset: offset,
		Has: &read.InputHas{NamespaceId: true, Name: true, OwnerId: true, Query: true, Status: true, Subject: true, Scoped: true, PageLimit: true, PageOffset: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.read, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*read.Output)
	if !ok {
		return nil, fmt.Errorf("namespace reader returned %T", value)
	}
	var result []*sdk.Namespace
	for _, row := range output.Namespaces {
		if row == nil || name != "" && row.Name != name || namespaceID != "" && row.NamespaceId != namespaceID {
			return nil, fmt.Errorf("namespace reader returned a mismatched row")
		}
		description := ""
		if row.Description != nil {
			description = *row.Description
		}
		roles := []string{}
		if row.AllowedRolesJson != nil {
			if err = json.Unmarshal([]byte(*row.AllowedRolesJson), &roles); err != nil {
				return nil, fmt.Errorf("invalid namespace roles")
			}
		}
		id := row.NamespaceId
		if id == "" {
			id = namespaceaccess.ID(row.OwnerId, row.Name)
		}
		visibility := row.Visibility
		if visibility == "" {
			visibility = namespaceaccess.Private
		}
		result = append(result, &sdk.Namespace{CanManage: scoped && principal.Subject == row.OwnerId, NamespaceID: id, Visibility: visibility, AllowedRoles: roles, MCPEnabled: row.McpEnabled, MCPPort: row.McpPort, OwnerID: row.OwnerId, Name: row.Name, Title: row.Title,
			Description: description, Status: row.Status, ETag: row.Etag,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return result, nil
}

func (t *Transport) ownedNamespaceStatus(ctx context.Context, ownerID, name string) (string, error) {
	reader, err := t.newNamespaceStoreReader()
	if err != nil {
		return "", err
	}
	defer reader.runtime.Shutdown(context.Background())
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.read,
		Input: &read.Input{OwnerId: ownerID, Name: name, Query: "", Status: "", Subject: "", Scoped: false,
			PageLimit: 2, PageOffset: 0,
			Has: &read.InputHas{OwnerId: true, Name: true, Query: true, Status: true, Subject: true,
				Scoped: true, PageLimit: true, PageOffset: true}}})
	if err != nil {
		return "", err
	}
	output, ok := value.(*read.Output)
	if !ok {
		return "", fmt.Errorf("owned namespace reader returned %T", value)
	}
	if len(output.Namespaces) == 0 {
		return "", sql.ErrNoRows
	}
	if len(output.Namespaces) != 1 || output.Namespaces[0] == nil ||
		output.Namespaces[0].OwnerId != ownerID || output.Namespaces[0].Name != name {
		return "", fmt.Errorf("owned namespace reader returned an ambiguous or mismatched row")
	}
	return output.Namespaces[0].Status, nil
}

func (t *Transport) namespaceUsage(ctx context.Context, ownerID, name string) (int64, error) {
	reader, err := t.newNamespaceStoreReader()
	if err != nil {
		return 0, err
	}
	defer reader.runtime.Shutdown(context.Background())
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.usage,
		Input: &usage.Input{OwnerId: ownerID, Name: name, Has: &usage.InputHas{OwnerId: true, Name: true}}})
	if err != nil {
		return 0, err
	}
	output, ok := value.(*usage.Output)
	if !ok || len(output.Usages) != 1 || output.Usages[0] == nil {
		return 0, fmt.Errorf("namespace usage reader returned %T with unexpected rows", value)
	}
	return output.Usages[0].Used, nil
}
