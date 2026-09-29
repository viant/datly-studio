package namespacemcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/namespaces/accesspredicate"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type Definition struct {
	OwnerID      string
	Visibility   string
	AllowedRoles []string
	Active       bool
	NamespaceID  string
	Enabled      bool
	Port         int
}

type Definitions interface {
	Load(context.Context) ([]Definition, error)
}

// SQLDefinitions uses a linked, private Datly reader. It is a trusted control-
// plane source; it must not be exposed as an unscoped public SDK endpoint.
type SQLDefinitions struct{ DB *sql.DB }

func (source SQLDefinitions) Load(ctx context.Context) ([]Definition, error) {
	return source.load(ctx, "")
}
func (source SQLDefinitions) LoadNamespace(ctx context.Context, id string) ([]Definition, error) {
	if id == "" {
		return nil, fmt.Errorf("namespace ID is required")
	}
	return source.load(ctx, id)
}
func (source SQLDefinitions) load(ctx context.Context, namespaceID string) ([]Definition, error) {
	if source.DB == nil {
		return nil, fmt.Errorf("namespace database is required")
	}
	tx, err := source.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	resources := resource.New()
	if err = resources.Register(stored.NamespaceDatlyResourceNamespace, stored.NamespaceDatlyResources); err != nil {
		return nil, err
	}
	connectors := &dsql.SQLComponent{DB: source.DB, Tx: tx}
	if err = connectors.RegisterConnector("studio", source.DB); err != nil {
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
	registration, target, err := readercomponent.Compile(reflect.TypeFor[stored.NamespaceComponent](), "store_read", reflect.TypeFor[stored.Input](), reflect.TypeFor[stored.Output](), resources, connectors, types)
	if err != nil {
		return nil, err
	}
	registration.Capabilities.Connector = connectors
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	var result []Definition
	const pageSize = 500
	for offset := 0; ; offset += pageSize {
		input := &stored.Input{NamespaceId: namespaceID, PageLimit: pageSize, PageOffset: offset, Scoped: false, Has: &stored.InputHas{NamespaceId: true, Name: true, OwnerId: true, Query: true, Status: true, Subject: true, Scoped: true, PageLimit: true, PageOffset: true}}
		value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
		if err != nil {
			return nil, err
		}
		output, ok := value.(*stored.Output)
		if !ok {
			return nil, fmt.Errorf("namespace reader returned %T", value)
		}
		for _, row := range output.Namespaces {
			if row == nil || row.NamespaceId == "" {
				return nil, fmt.Errorf("namespace identity is missing")
			}
			port := 0
			if row.McpPort != nil {
				port = *row.McpPort
			}
			roles := []string{}
			if row.AllowedRolesJson != nil {
				if err := json.Unmarshal([]byte(*row.AllowedRolesJson), &roles); err != nil {
					return nil, fmt.Errorf("namespace roles are invalid")
				}
			}
			result = append(result, Definition{NamespaceID: row.NamespaceId, OwnerID: row.OwnerId, Visibility: row.Visibility, AllowedRoles: roles, Active: row.Status == "active", Enabled: row.McpEnabled && row.Status == "active", Port: port})
		}
		if len(output.Namespaces) < pageSize {
			break
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
