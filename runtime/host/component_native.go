package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/viant/authz/oauth"
	"github.com/viant/datly/application"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

// NativeComponentRuntime is supplied only to trusted namespace assembly. Its
// metadata preparation carries no data-execution capability; execution independently
// enforces the caller's current component and namespace authority.
type NativeComponentRuntime interface {
	ResolveNativeComponentBinding(context.Context, ComponentReference) (NativeComponentBinding, error)
	PrepareNativeComponent(context.Context, ComponentReference, NativeComponentBinding, func(NativeComponentFamily) error) error
	ExecuteNativeComponent(context.Context, ComponentReference, NativeComponentBinding, func(context.Context, NativeComponentFamily, exec.ComponentInvoker) (any, error)) (any, error)
}

// PrepareNativeComponent validates retained metadata at process startup without
// a startup principal. This method is for embedding-host assembly only and must
// never be routed to HTTP/MCP or used as a resource-discovery endpoint. The host
// must establish retained source ownership before calling it. The inspection
// receives no invoker; the isolated generation is always closed on return.
func (s *Service) PrepareNativeComponent(ctx context.Context, ref ComponentReference, pin NativeComponentBinding, inspect func(NativeComponentFamily) error) (resultErr error) {
	if inspect == nil || pin.Kind == "" || pin.ContentFingerprint == "" {
		return fmt.Errorf("complete native metadata preparation is required")
	}
	metadata := &exactComponentMetadata{}
	manager, current, err := s.componentRuntimeInspected(ctx, ref, false, metadata)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, shutdownNativeMetadata(ctx, manager.Shutdown)) }()
	if err := validateNativeComponentBinding(ref, pin, current, metadata); err != nil {
		return err
	}
	if metadata.registered == nil {
		return fmt.Errorf("native component metadata unavailable")
	}
	family, err := newNativeComponentFamily(metadata, ref)
	if err != nil {
		return err
	}
	defer family.close()
	return errors.Join(inspect(family), ctx.Err())
}

// The manager owns resource retirement; this bounds the host's wait while
// preserving operation values even after the caller cancels its inspection.
func shutdownNativeMetadata(ctx context.Context, shutdown func(context.Context) error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return shutdown(cleanupCtx)
}

// nativeComponentDescription excludes every data-execution carrier. The spec
// passes through its ordinary wire representation, so transient native authority
// and values stored behind any cannot retain providers or invocation interfaces.
// Canonical immutable input/output plans contain binding/encoding algorithms,
// but no session, provider, data source or native invoker; they are not recompiled.
func nativeComponentDescription(registered *registry.RegisteredComponent) (*registry.RegisteredComponent, error) {
	raw, err := json.Marshal(registered.Component)
	if err != nil {
		return nil, fmt.Errorf("native component metadata: %w", err)
	}
	component := &spec.Component{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(component); err != nil {
		return nil, fmt.Errorf("native component metadata: %w", err)
	}
	return &registry.RegisteredComponent{Component: component, Input: registered.Input, Output: registered.Output, OutputType: registered.OutputType, Documentation: registered.Documentation}, nil
}

type exactComponentMetadata struct {
	registered *registry.RegisteredComponent
	family     []*registry.RegisteredComponent
	authorize  func(context.Context, *http.Request, exec.ComponentTarget) error
}

// ExecuteNativeComponent supplies the exact generation's metadata and a native
// invoker to a trusted embedding host. It is an in-process capability, not a
// transport endpoint. The host must use the supplied metadata for input/output
// preparation and must not retain or mutate it. Only explicit targets from that exact source family can be invoked;
// injected dependencies remain available only to native internal binding.
//
// The callback's result is returned only after the same source, named-release,
// policy and identity completion fences used by ExecuteComponentJSON succeed.
// No active publication is consulted or changed.
func (s *Service) ExecuteNativeComponent(ctx context.Context, ref ComponentReference, pin NativeComponentBinding, execute func(context.Context, NativeComponentFamily, exec.ComponentInvoker) (any, error)) (any, error) {
	if execute == nil {
		return nil, fmt.Errorf("native component execution is required")
	}
	return s.executeExactComponent(ctx, ref, pin.ComponentBinding, func(ctx context.Context, manager *application.Manager, metadata *exactComponentMetadata) (any, error) {
		if metadata.registered == nil || metadata.authorize == nil {
			return nil, fmt.Errorf("native component metadata unavailable")
		}
		family, err := newNativeComponentFamily(metadata, ref)
		if err != nil {
			return nil, err
		}
		defer family.close()
		invoker := &exactNativeInvoker{manager: manager, metadata: metadata, authority: ctx, targets: family.targets, active: true}
		defer invoker.close()
		return execute(ctx, family, invoker)
	}, func(current windowprotocol.ComponentBinding, metadata *exactComponentMetadata) error {
		return validateNativeComponentBinding(ref, pin, current, metadata)
	})
}

// The invoker accepts only captured family targets and cannot outlive the fenced call.
// Holding its lock also lets completion wait for any invocation the trusted host
// started concurrently before disposing of the exact generation.
type exactNativeInvoker struct {
	mu        sync.Mutex
	active    bool
	manager   *application.Manager
	metadata  *exactComponentMetadata
	authority context.Context
	targets   map[exec.ComponentTarget]bool
}

func (i *exactNativeInvoker) close() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.active = false
	i.manager = nil
	i.metadata = nil
	i.authority = nil
	i.targets = nil
}
func (i *exactNativeInvoker) InvokeComponent(ctx context.Context, request exec.ComponentRequest) (any, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.active || !i.targets[request.Target] {
		return nil, fmt.Errorf("native exact component target unavailable")
	}
	// Keep the authority selected by the outer fenced invocation. A callback's
	// child context may narrow cancellation, but cannot swap bearer or facts and
	// obtain another principal's rows under the original completion fence.
	if ctx == nil {
		return nil, fmt.Errorf("native invocation context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	executionCtx, cancel := context.WithCancel(i.authority)
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	defer cancel()
	ctx = executionCtx
	// InvokeComponent deliberately does not run transport middleware. Retain the
	// compiled target authorizer here, including entity-bounded policy checks.
	requestMetadata, err := http.NewRequestWithContext(ctx, request.Target.Route.Method, request.Target.Route.Path, nil)
	if err != nil {
		return nil, err
	}
	if bearer := oauth.Bearer(ctx); bearer != "" {
		requestMetadata.Header.Set("Authorization", "Bearer "+bearer)
	}
	if err := i.metadata.authorize(ctx, requestMetadata, request.Target); err != nil {
		return nil, err
	}
	return i.manager.InvokeComponent(ctx, request)
}
