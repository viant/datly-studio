package nativeruntime

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/studio/artifact_release/bundle"
	"github.com/viant/datly-studio/studio/artifact_release/channels"
	"github.com/viant/datly-studio/studio/artifact_release/complete"
	"github.com/viant/datly-studio/studio/artifact_release/create_channel"
	"github.com/viant/datly-studio/studio/artifact_release/fail"
	"github.com/viant/datly-studio/studio/artifact_release/history"
	"github.com/viant/datly-studio/studio/artifact_release/intent"
	"github.com/viant/datly-studio/studio/artifact_release/put_bundle"
	"github.com/viant/datly-studio/studio/artifact_release/snapshot"
	"github.com/viant/datly-studio/studio/artifact_release/stage"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writer "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	"reflect"
)

// Runtime registers the stock-transcribed release components and borrows the host connector. Generated contracts own
// all read/write metadata; each invocation's native runtime owns its transaction.
type Runtime struct {
	registrations []*registry.RegisteredComponent
	resources     *resource.Store
}

// New composes native release graphs without owning the database pool, executing
// application SQL, or replacing native transaction management.
func New(ctx context.Context, db *sql.DB) (*Runtime, error) {
	if ctx == nil || ctx.Err() != nil || db == nil {
		return nil, fmt.Errorf("native release connector is required")
	}
	holders := []any{stage.StageComponent{}, complete.CompleteComponent{}, fail.FailComponent{}, snapshot.SnapshotComponent{}, bundle.BundleComponent{}, intent.IntentComponent{}, channels.ChannelsComponent{}, history.HistoryComponent{}, create_channel.CreateChannelComponent{}, put_bundle.PutBundleComponent{}}
	reflected, err := bootstrap.ReflectPackages(bootstrap.LinkedPackagePaths(holders))
	if err != nil {
		return nil, err
	}
	if err = reflected.Types.LinkRuntimeAll(typecatalog.TypeOriginPackage,
		x.NewType(reflect.TypeFor[stage.StageRules]()), x.NewType(reflect.TypeFor[stage.BindingRules]()),
		x.NewType(reflect.TypeFor[complete.Rules]()), x.NewType(reflect.TypeFor[fail.Rules]()),
		x.NewType(reflect.TypeFor[create_channel.Rules]()), x.NewType(reflect.TypeFor[put_bundle.Rules]())); err != nil {
		return nil, err
	}
	if len(reflected.Components) != len(holders) {
		return nil, fmt.Errorf("native release component set is incomplete")
	}
	resources := resource.New()
	for _, holder := range holders {
		embedded := holder.(bootstrap.EmbeddedResource)
		if err = resources.Register(embedded.EmbedNamespace(), *embedded.EmbedFS()); err != nil {
			return nil, err
		}
	}
	sqlConnector := &dsql.SQLComponent{DB: db}
	if err = sqlConnector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	if err = sqlConnector.RegisterConnector("catalog", db); err != nil {
		return nil, err
	}
	result := &Runtime{resources: resources}
	for _, source := range reflected.Components {
		component, err := source.Resolve(source.LinkedInputType, source.LinkedOutputType)
		if err != nil {
			return nil, err
		}
		input := bootstrap.ArtifactInput{Component: component, InputType: source.LinkedInputType, OutputType: source.LinkedOutputType, Types: reflected.Types, Resources: resources, IdentifierProducts: sqlConnector.IdentifierProduct}
		mutation := component.Settings.Mutation
		if mutation != "" {
			input.Handler, err = writer.NewWithIdentifierProducts(component, source.LinkedInputType, source.LinkedOutputType, mutation, sqlConnector.IdentifierProduct)
			if err != nil {
				return nil, err
			}
			input.HandlerOwnedOutput = true
		}
		artifact, err := bootstrap.BuildArtifact(input)
		if err != nil {
			return nil, err
		}
		runtime := registry.RegisteredComponent{DataSource: dml.Source{DB: db}}
		runtime.Capabilities.Connector = sqlConnector
		var providers []locator.Provider
		if view, err := artifact.NewViewProvider(bootstrap.ViewRuntimeConfig{SQL: sqlConnector}); err != nil {
			return nil, err
		} else if view != nil {
			providers = append(providers, view)
		}
		runtime.Providers = providers
		if mutation == "" {
			compiled := artifact.ReaderCompilation()
			if compiled == nil {
				return nil, fmt.Errorf("native release reader is unavailable")
			}
			runtime.Reader, err = compiled.NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sqlConnector})
			if err != nil {
				return nil, err
			}
		}
		registration, err := artifact.Registration(runtime)
		if err != nil {
			return nil, err
		}
		result.registrations = append(result.registrations, registration)
	}
	return result, nil
}

func (i *Runtime) InvokeComponent(ctx context.Context, request exec.ComponentRequest) (any, error) {
	if i == nil || ctx == nil || ctx.Err() != nil {
		return nil, fmt.Errorf("native release context is required")
	}
	runtime, err := druntime.NewRuntime(i.registrations, druntime.WithResources(i.resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.WithoutCancel(ctx))
	return runtime.InvokeComponent(ctx, request)
}
