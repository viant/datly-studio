package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/constant"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/datly/typecatalog"
)

// A generation owns Datly's connector set. Readers borrow its named pools;
// retiring a reader must not close a pool still used by another reader.
type runtimeConnectorSet struct {
	native      []*connector.Set
	connections column.Connections
}

func (p *runtimeConnectorSet) Close() error {
	if p == nil || p.native == nil {
		return nil
	}
	var result error
	for _, native := range p.native {
		result = errors.Join(result, native.Close())
	}
	return result
}
func (p *runtimeConnectorSet) sources(ctx context.Context, name string) (*runtimeSources, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	db, ok := p.connections[name]
	if !ok || db == nil {
		return nil, fmt.Errorf("dynamic reader connector %q is unavailable", name)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	for name, connection := range p.connections {
		if err := sqlComponent.RegisterConnector(name, connection); err != nil {
			return nil, err
		}
	}
	return &runtimeSources{sql: sqlComponent, connections: p.connections}, nil
}
func openRuntimeConnectorSet(ctx context.Context, definitions []definition, active map[string]connectorDefinition, resolver connectorsecret.Resolver, resources func(definition) fs.FS, types *typecatalog.Catalog, predicates *predicatecatalog.Catalog, constants ...*constant.Values) (*runtimeConnectorSet, error) {
	var instance *constant.Values
	if len(constants) > 0 {
		instance = constants[0]
	}
	return openRuntimeConnectorSetObserved(ctx, definitions, active, resolver, resources, types, predicates, instance, nil)
}

func openRuntimeConnectorSetObserved(ctx context.Context, definitions []definition, active map[string]connectorDefinition, resolver connectorsecret.Resolver, resources func(definition) fs.FS, types *typecatalog.Catalog, predicates *predicatecatalog.Catalog, instance *constant.Values, isolate func(definition, string, error) bool) (*runtimeConnectorSet, error) {
	available := make([]string, 0, len(active))
	for name := range active {
		available = append(available, name)
	}
	sort.Strings(available)
	required := map[string]bool{}
	owners := map[string][]definition{}
	for _, definition := range definitions {
		var resourceFS fs.FS
		if resources != nil {
			resourceFS = resources(definition)
		}
		catalog, err := predicatepolicy.Types(types, definition.policy, predicates)
		if err != nil {
			if isolate != nil && isolate(definition, "source_policy", err) {
				continue
			}
			return nil, err
		}
		names, err := referencedRuntimeConnectors(ctx, definition, available, resourceFS, catalog, instance)
		if err != nil {
			if isolate != nil && isolate(definition, "contract", err) {
				continue
			}
			return nil, err
		}
		for _, name := range names {
			owners[name] = append(owners[name], definition)
			required[name] = true
		}
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	configs := make([]connector.Config, 0, len(names))
	rejectConnector := func(name string, err error) bool {
		if isolate == nil {
			return false
		}
		for _, definition := range owners[name] {
			if !isolate(definition, "connector", err) {
				return false
			}
		}
		return true
	}
	for _, name := range names {
		item, ok := active[name]
		if !ok {
			err := fmt.Errorf("dynamic readers require active Studio connector %q", name)
			if rejectConnector(name, err) {
				continue
			}
			return nil, err
		}
		dsn, err := connectorsecret.Resolve(ctx, resolver, item.dsn, item.secretRef, instance)
		if err != nil {
			cause := fmt.Errorf("resolve Studio connector %q secret: %w", name, err)
			if rejectConnector(name, cause) {
				continue
			}
			return nil, cause
		}
		configs = append(configs, connector.Config{Name: name, Driver: normalizeDriver(item.driver), DSN: dsn})
	}
	result := &runtimeConnectorSet{connections: column.Connections{}}
	// Each configured connector owns one native pool shared by every dependent
	// reader. Opening a failed connector cannot close another connector's pool.
	for _, config := range configs {
		name := config.Name
		native, err := connector.Open(ctx, []connector.Config{config}, name)
		if err != nil {
			if rejectConnector(name, err) {
				continue
			}
			_ = result.Close()
			return nil, err
		}
		db, err := native.ResolveDB(ctx, name)
		if err == nil {
			item := active[name]
			err = connectorinit.Configure(ctx, db, item.driver, item.options)
		}
		if err != nil {
			_ = native.Close()
			if rejectConnector(name, err) {
				continue
			}
			_ = result.Close()
			return nil, err
		}
		result.native = append(result.native, native)
		result.connections[name] = db
	}

	return result, nil
}

// referencedRuntimeConnectors is the same declared DQL connector closure used
// both to open execution pools and to partition compiled artifact identities.
// Its exact resources and type authority must match native compilation; a partial
// inspection cannot safely define either closure.
func referencedRuntimeConnectors(ctx context.Context, definition definition, available []string, resources fs.FS, types *typecatalog.Catalog, constants *constant.Values) ([]string, error) {
	required := map[string]bool{definition.connector: true}
	inspection := readerbuilder.New(readerbuilder.Config{Const: constants, Scope: definition.scope, Name: definition.name, Resources: resources, Types: types, AvailableConnectors: available}).Apply(ctx, readerbuilder.Request{DQL: definition.dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.IsError() {
			return nil, fmt.Errorf("inspect connectors for %s version %d: %w", definition.reportID, definition.versionNo, &transcribe.CompileError{Diagnostics: inspection.Diagnostics})
		}
	}
	if inspection.Structure == nil || inspection.Structure.Component == nil {
		return nil, fmt.Errorf("inspect connectors for %s version %d: native component unavailable", definition.reportID, definition.versionNo)
	}
	if inspection.Structure != nil {
		// Independent input views retain their connector in the native component
		// graph rather than an outer use_connector function.
		if component := inspection.Structure.Component; component != nil {
			// The authored default can override the component record's fallback.
			// Native transcription uses this setting even without use_connector.
			if component.Settings != nil {
				if name := strings.TrimSpace(component.Settings.DefaultConnector); name != "" {
					required[name] = true
				}
			}
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
	return names, nil
}
