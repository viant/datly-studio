package preview

import (
	"context"

	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
)

// DependencyRuntime supplies host-owned implementations of a reader's declared
// component dependencies. It does not replace the reader's predicates or decide
// business row access. The preview owns Shutdown after a successful resolution.
type DependencyRuntime struct {
	Components []*registry.RegisteredComponent
	Options    []druntime.Option
	Shutdown   func(context.Context) error
}

type DependencyResolver func(context.Context, *spec.Component) (*DependencyRuntime, error)

func (d Dynamic) prepareContexts(ctx context.Context, definition *definition, component *spec.Component) (*DependencyRuntime, error) {
	contexts, err := d.previewContexts(ctx, definition, component)
	if err != nil {
		return nil, err
	}
	result := &DependencyRuntime{Components: contexts}
	if d.Dependencies == nil {
		return result, nil
	}
	extra, err := d.Dependencies(ctx, component)
	if err != nil {
		return nil, err
	}
	if extra == nil {
		return result, nil
	}
	result.Components = append(result.Components, extra.Components...)
	result.Options = extra.Options
	result.Shutdown = extra.Shutdown
	return result, nil
}
func (d *DependencyRuntime) close() {
	if d != nil && d.Shutdown != nil {
		_ = d.Shutdown(context.Background())
	}
}
