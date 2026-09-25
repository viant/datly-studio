// Package exact assembles one caller-supplied Datly v1 runtime contract for
// an exact component version. It owns no catalog lookup or connector metadata
// SQL; applications supply those trusted inputs and keep authorization current.
package exact

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/report"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
)

// Config supplies an already compiled, exact-version contract and its trusted
// connector. Decide must independently re-evaluate the current execute policy
// whenever a declared access-context dependency is invoked. It must not return
// a caller-supplied or previously cached authorization decision.
type Config struct {
	Contract   *transcribe.RuntimeContract
	SQL        *dsql.SQLComponent
	ResourceID string
	Decide     func(accesscontext.Dependency) accesscontext.Decider
	// RequiredEntityTypes comes from the current trusted execute decision.
	// Every bounded dimension must have a declared native context dependency.
	RequiredEntityTypes []string
}

// Native owns a temporary Datly runtime for one exact component contract.
// Request fields are supplied by the caller, but component/param dependencies
// are rebound by Datly's native engine before SQL execution.
type Native struct {
	runtime      *druntime.Runtime
	target       dexec.ComponentTarget
	inputType    reflect.Type
	dependencies []accesscontext.Dependency
}

func New(ctx context.Context, config Config) (*Native, error) {
	contract := config.Contract
	if contract == nil || contract.Component == nil || contract.InputType == nil || contract.OutputType == nil || config.SQL == nil || config.ResourceID == "" {
		return nil, fmt.Errorf("exact contract, types, SQL connector, and resource id are required")
	}
	if contract.Component.Key.Kind != spec.KindComponent || len(contract.Component.Routes) != 1 || contract.Component.Routes[0] == nil {
		return nil, fmt.Errorf("exact source must declare one Datly component route")
	}
	dependencies, err := accesscontext.DependsOn(contract.Component)
	if err != nil {
		return nil, err
	}
	deciders := make(map[accesscontext.Dependency]accesscontext.Decider, len(dependencies))
	declared := make(map[string]bool, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.ComponentID != config.ResourceID {
			return nil, fmt.Errorf("exact source binds access context for another resource")
		}
		if config.Decide == nil {
			return nil, fmt.Errorf("exact source access-context decision is unavailable")
		}
		decider := config.Decide(dependency)
		if decider == nil {
			return nil, fmt.Errorf("exact source access-context decision is unavailable")
		}
		deciders[dependency] = decider
		declared[dependency.EntityType] = true
	}
	for _, entityType := range config.RequiredEntityTypes {
		if strings.TrimSpace(entityType) == "" || !declared[entityType] {
			return nil, fmt.Errorf("exact source does not bind required entity dimension %q", entityType)
		}
	}
	compilation, err := report.NewProjectCompiler(report.ProjectConfig{Types: contract.Types}).CompileArtifacts([]bootstrap.ArtifactInput{{
		Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType,
		Types: contract.Types, Resources: contract.Resources, CodecFactory: accesscontext.Codecs(),
	}})
	if err != nil {
		return nil, fmt.Errorf("compile exact source: %w", err)
	}
	registrations, err := compilation.RuntimeComponents(ctx, report.RuntimeConfigureFunc(func(_ context.Context, artifact *report.ComponentArtifact) (report.RuntimeCapabilities, error) {
		if artifact.IsReport() {
			return report.RuntimeCapabilities{}, nil
		}
		compiled := artifact.ReaderCompilation()
		if compiled == nil {
			return report.RuntimeCapabilities{}, fmt.Errorf("exact source reader compilation is unavailable")
		}
		reader, err := compiled.NewExecution(bootstrap.ReaderRuntimeConfig{SQL: config.SQL})
		return report.RuntimeCapabilities{Reader: reader}, err
	}))
	if err != nil {
		return nil, err
	}
	for _, dependency := range dependencies {
		contextRegistration, err := accesscontext.Register(dependency, deciders[dependency])
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, contextRegistration)
	}
	if !containsSource(registrations, contract.Component.Key) {
		return nil, fmt.Errorf("exact source registration is absent")
	}
	runtime, err := druntime.NewRuntime(registrations)
	if err != nil {
		return nil, err
	}
	route := contract.Component.Routes[0]
	return &Native{runtime: runtime, target: dexec.ComponentTarget{Component: contract.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}, inputType: contract.InputType, dependencies: append([]accesscontext.Dependency(nil), dependencies...)}, nil
}

func containsSource(registrations []*registry.RegisteredComponent, key spec.Key) bool {
	for _, item := range registrations {
		if item != nil && item.Component != nil && item.Component.Key == key {
			return true
		}
	}
	return false
}

// Dependencies reports the declared server-owned context dimensions. An
// application must deny a bounded decision if these cannot enforce it all.
func (n *Native) Dependencies() []accesscontext.Dependency {
	if n == nil {
		return nil
	}
	return append([]accesscontext.Dependency(nil), n.dependencies...)
}

func (n *Native) InputType() reflect.Type {
	if n == nil {
		return nil
	}
	return n.inputType
}

func (n *Native) Invoke(ctx context.Context, input any) (any, error) {
	if n == nil || n.runtime == nil {
		return nil, fmt.Errorf("exact runtime is unavailable")
	}
	if input != nil {
		value := reflect.ValueOf(input)
		if value.Kind() != reflect.Pointer || value.IsNil() || value.Type().Elem() != n.inputType {
			return nil, fmt.Errorf("exact input must be *%s", n.inputType)
		}
	}
	return n.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: n.target, Input: input})
}

func (n *Native) Close(ctx context.Context) error {
	if n == nil || n.runtime == nil {
		return nil
	}
	return n.runtime.Shutdown(ctx)
}
