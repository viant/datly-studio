package host

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/constant"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe/column"
)

// A generation owns Datly's connector set. Readers borrow its named pools;
// retiring a reader must not close a pool still used by another reader.
type runtimeConnectorSet struct {
	native      *connector.Set
	connections column.Connections
}

func (p *runtimeConnectorSet) Close() error {
	if p == nil || p.native == nil {
		return nil
	}
	return p.native.Close()
}
func (p *runtimeConnectorSet) sources(ctx context.Context, name string) (*runtimeSources, error) {
	db, err := p.native.ResolveDB(ctx, name)
	if err != nil {
		return nil, err
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	for name, connection := range p.connections {
		if err = sqlComponent.RegisterConnector(name, connection); err != nil {
			return nil, err
		}
	}
	return &runtimeSources{sql: sqlComponent, connections: p.connections}, nil
}
func openRuntimeConnectorSet(ctx context.Context, definitions []definition, active map[string]connectorDefinition, resolver connectorsecret.Resolver, constants ...*constant.Values) (*runtimeConnectorSet, error) {
	var instance *constant.Values
	if len(constants) > 0 {
		instance = constants[0]
	}
	available := make([]string, 0, len(active))
	for name := range active {
		available = append(available, name)
	}
	sort.Strings(available)
	required := map[string]bool{}
	for _, definition := range definitions {
		required[definition.connector] = true
		inspection := readerbuilder.New(readerbuilder.Config{Const: instance, Scope: definition.scope, Name: definition.name, AvailableConnectors: available}).Apply(ctx, readerbuilder.Request{DQL: definition.dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
		if inspection.Structure != nil {
			for _, function := range inspection.Structure.Functions {
				if strings.EqualFold(function.Name, "use_connector") && len(function.Args) > 1 {
					required[strings.Trim(strings.TrimSpace(function.Args[1]), "'\"")] = true
				}
			}
		}
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	configs := make([]connector.Config, 0, len(names))
	for _, name := range names {
		item, ok := active[name]
		if !ok {
			return nil, fmt.Errorf("dynamic readers require active Studio connector %q", name)
		}
		dsn, err := connectorsecret.Resolve(ctx, resolver, item.dsn, item.secretRef, instance)
		if err != nil {
			return nil, fmt.Errorf("resolve Studio connector %q secret: %w", name, err)
		}
		configs = append(configs, connector.Config{Name: name, Driver: normalizeDriver(item.driver), DSN: dsn})
	}
	defaultName := ""
	if len(names) > 0 {
		defaultName = names[0]
	}
	native, err := connector.Open(ctx, configs, defaultName)
	if err != nil {
		return nil, fmt.Errorf("open dynamic reader connector set: %w", err)
	}
	result := &runtimeConnectorSet{native: native, connections: column.Connections{}}
	for _, name := range names {
		db, resolveErr := native.ResolveDB(ctx, name)
		if resolveErr != nil {
			native.Close()
			return nil, resolveErr
		}
		item := active[name]
		if err = connectorinit.Configure(ctx, db, item.driver, item.options); err != nil {
			native.Close()
			return nil, fmt.Errorf("configure Studio connector %q: %w", name, err)
		}
		result.connections[name] = db
	}
	return result, nil
}
