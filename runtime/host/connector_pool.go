package host

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/constant"
	"github.com/viant/datly/spec"
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
func openRuntimeConnectorSet(ctx context.Context, definitions []definition, active map[string]connectorDefinition, resolver connectorsecret.Resolver, resources func(definition) fs.FS, constants ...*constant.Values) (*runtimeConnectorSet, error) {
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
		var resourceFS fs.FS
		if resources != nil {
			resourceFS = resources(definition)
		}
		for _, name := range referencedRuntimeConnectors(ctx, definition, available, resourceFS, instance) {
			required[name] = true
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

// referencedRuntimeConnectors is the same declared DQL connector closure used
// both to open execution pools and to partition compiled artifact identities.
func referencedRuntimeConnectors(ctx context.Context, definition definition, available []string, resources fs.FS, constants *constant.Values) []string {
	required := map[string]bool{definition.connector: true}
	inspection := readerbuilder.New(readerbuilder.Config{Const: constants, Scope: definition.scope, Name: definition.name, Resources: resources, AvailableConnectors: available}).Apply(ctx, readerbuilder.Request{DQL: definition.dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if inspection.Structure != nil {
		// Independent input views retain their connector in the native component
		// graph rather than an outer use_connector function.
		if component := inspection.Structure.Component; component != nil {
			views := append([]*spec.View{component.RootView}, component.Views...)
			seen := map[*spec.View]bool{}
			for len(views) > 0 {
				view := views[0]
				views = views[1:]
				if view == nil || seen[view] {
					continue
				}
				seen[view] = true
				if source := view.RuntimeSource(); source != nil && source.Bindings != nil {
					if name := strings.TrimSpace(source.Bindings.Connector); name != "" {
						required[name] = true
					}
				}
				for _, relation := range view.Relations {
					if relation != nil {
						views = append(views, relation.View)
					}
				}
			}
		}
		for _, function := range inspection.Structure.Functions {
			if strings.EqualFold(function.Name, "use_connector") && len(function.Args) > 1 {
				required[strings.Trim(strings.TrimSpace(function.Args[1]), "'\"")] = true
			}
		}
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
