// Package access adapts the generic ACL policy store contract to the Datly v1
// resource policy components. Every read and write executes the generated
// studio/resource_policy reader and writer in-process; this package owns no
// SQL and opens no transactions of its own.
package access

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	acl "github.com/viant/datly-studio/sdk/access"
	policyreader "github.com/viant/datly-studio/studio/resource_policy/reader"
	policywriter "github.com/viant/datly-studio/studio/resource_policy/writer"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// Store hosts the resource policy components against one Studio database.
// DB is the "studio" connector the generated components declare.
type Store struct {
	DB *sql.DB

	mu      sync.Mutex
	runtime *druntime.Runtime
	read    dexec.ComponentTarget
	write   dexec.ComponentTarget
}

var _ acl.Store = (*Store)(nil)

const studioConnector = "studio"

// Close releases the hosted component runtime.
func (s *Store) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime == nil {
		return nil
	}
	err := s.runtime.Shutdown(ctx)
	s.runtime = nil
	return err
}

// components builds the reader and writer registrations from the generated
// packages exactly as the Datly standalone host would, bound to Store.DB.
func (s *Store) components() (*druntime.Runtime, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime != nil {
		return s.runtime, nil
	}
	if s.DB == nil {
		return nil, errors.New("resource policy store requires a Studio database")
	}
	resources := resource.New()
	if err := resources.Register(policyreader.PolicyDatlyResourceNamespace, policyreader.PolicyDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(policywriter.PolicyDatlyResourceNamespace, policywriter.PolicyDatlyResources); err != nil {
		return nil, err
	}
	sqlComponent := &dsql.SQLComponent{DB: s.DB}
	if err := sqlComponent.RegisterConnector(studioConnector, s.DB); err != nil {
		return nil, err
	}
	readerComponent, err := resolveComponent(reflect.TypeOf(policyreader.PolicyComponent{}), "reader", reflect.TypeOf(policyreader.Input{}), reflect.TypeOf(policyreader.Output{}))
	if err != nil {
		return nil, err
	}
	readerArtifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: readerComponent, InputType: reflect.TypeOf(policyreader.Input{}), OutputType: reflect.TypeOf(policyreader.Output{}), Resources: resources})
	if err != nil {
		return nil, fmt.Errorf("resource policy reader: %w", err)
	}
	execution, err := readerArtifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sqlComponent})
	if err != nil {
		return nil, err
	}
	readerRegistration, err := readerArtifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		return nil, err
	}
	writerComponent, err := resolveComponent(reflect.TypeOf(policywriter.PolicyComponent{}), "writer", reflect.TypeOf(policywriter.Input{}), reflect.TypeOf(policywriter.Output{}))
	if err != nil {
		return nil, err
	}
	writerArtifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: writerComponent, InputType: reflect.TypeOf(policywriter.Input{}), OutputType: reflect.TypeOf(policywriter.Output{}), Resources: resources})
	if err != nil {
		return nil, fmt.Errorf("resource policy writer: %w", err)
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: writerArtifact.ViewDependencies, Input: writerArtifact.Input, SQL: sqlComponent})
	if err != nil {
		return nil, err
	}
	handler, err := writerhandler.New(writerComponent, reflect.TypeOf(policywriter.Input{}), reflect.TypeOf(policywriter.Output{}), "patch")
	if err != nil {
		return nil, err
	}
	writerRegistration := &registry.RegisteredComponent{Component: writerArtifact.Component, Input: writerArtifact.Input, Output: writerArtifact.Output, OutputType: reflect.TypeOf(policywriter.Output{}), Handler: handler, Providers: []locator.Provider{views}, DataSource: dml.Source{DB: s.DB}}
	writerRegistration.Capabilities.Connector = sqlComponent
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{readerRegistration, writerRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	s.read = componentTarget(readerRegistration.Component)
	s.write = componentTarget(writerRegistration.Component)
	s.runtime = runtime
	return runtime, nil
}

func resolveComponent(holder reflect.Type, packageName string, inputType, outputType reflect.Type) (*spec.Component, error) {
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, fmt.Errorf("%s has no component contract", holder)
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, fmt.Errorf("%s has no component metadata", holder)
	}
	return (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: packageName, PackagePath: holder.PkgPath(), Tag: metadata, InputType: inputType.Name(), OutputType: outputType.Name()}).Resolve(inputType, outputType)
}

func componentTarget(component *spec.Component) dexec.ComponentTarget {
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	return target
}

// Get executes the generated reader for one exact resource identity. Every
// key dimension is a required predicate, so a missing head yields no row.
func (s *Store) Get(ctx context.Context, r acl.Resource) (acl.Document, error) {
	if r.Tenant == "" || r.Kind == "" || r.ID == "" || r.Version == "" {
		return acl.Document{}, sql.ErrNoRows
	}
	runtime, err := s.components()
	if err != nil {
		return acl.Document{}, err
	}
	input := &policyreader.Input{
		TenantId: r.Tenant, ResourceKind: r.Kind, ResourceId: r.ID, ResourceVersion: r.Version,
		Has: &policyreader.InputHas{TenantId: true, ResourceKind: true, ResourceId: true, ResourceVersion: true},
	}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.read, Input: input})
	if err != nil {
		return acl.Document{}, err
	}
	output, ok := value.(*policyreader.Output)
	if !ok || len(output.Policies) == 0 || output.Policies[0] == nil {
		return acl.Document{}, sql.ErrNoRows
	}
	if len(output.Policies) != 1 {
		return acl.Document{}, fmt.Errorf("resource policy identity resolved %d heads", len(output.Policies))
	}
	row := output.Policies[0]
	if row.Revision == nil {
		return acl.Document{}, errors.New("resource policy head has no revision")
	}
	d := acl.Document{Resource: r, Revision: int64(*row.Revision)}
	if err = json.Unmarshal(row.PoliciesJson, &d.Policies); err != nil {
		return acl.Document{}, err
	}
	return d, nil
}

// Provision is a server bootstrap operation, intentionally absent from the ACL
// service and client API. It cannot replace an existing policy.
func (s *Store) Provision(ctx context.Context, d acl.Document, actor string) (acl.Document, error) {
	return s.activate(ctx, d, 0, actor)
}

// Replace activates a new revision only when the caller's expected revision is
// the current head.
func (s *Store) Replace(ctx context.Context, d acl.Document, expected int64, actor string) (acl.Document, error) {
	if expected < 1 || d.Revision != expected {
		return acl.Document{}, acl.ErrConflict
	}
	return s.activate(ctx, d, expected, actor)
}

// activate maps the generic document onto the generated writer input: the head
// carries the expected revision as its concurrency token and one history row
// records the next revision. The writer's PolicyRules lifecycle and managed
// transaction own atomicity, compare-and-swap and bootstrap protection.
func (s *Store) activate(ctx context.Context, d acl.Document, expected int64, actor string) (acl.Document, error) {
	r := d.Resource
	if r.Tenant == "" || r.Kind == "" || r.ID == "" || r.Version == "" || actor == "" || len(d.Policies) == 0 {
		return acl.Document{}, acl.ErrDenied
	}
	for action, p := range d.Policies {
		if err := acl.ValidatePolicy(action, p); err != nil {
			return acl.Document{}, err
		}
	}
	body, err := json.Marshal(d.Policies)
	if err != nil {
		return acl.Document{}, err
	}
	runtime, err := s.components()
	if err != nil {
		return acl.Document{}, err
	}
	token := int(expected)
	next := token + 1
	occurred := time.Now().UTC()
	history := &policywriter.ResourcePolicyRevision{
		TenantId: r.Tenant, ResourceKind: r.Kind, ResourceId: r.ID, ResourceVersion: r.Version,
		Revision: &next, PoliciesJson: json.RawMessage(body), ActorId: actor, OccurredAt: &occurred,
		Has: &policywriter.ResourcePolicyRevisionHas{TenantId: true, ResourceKind: true, ResourceId: true, ResourceVersion: true, Revision: true, PoliciesJson: true, ActorId: true, OccurredAt: true},
	}
	head := &policywriter.ResourcePolicyHead{
		TenantId: r.Tenant, ResourceKind: r.Kind, ResourceId: r.ID, ResourceVersion: r.Version,
		Revision: &token, History: []*policywriter.ResourcePolicyRevision{history},
		Has: &policywriter.ResourcePolicyHeadHas{TenantId: true, ResourceKind: true, ResourceId: true, ResourceVersion: true, Revision: true, History: true},
	}
	input := &policywriter.Input{Policies: []*policywriter.ResourcePolicyHead{head}, Has: &policywriter.InputHas{Policies: true}}
	if _, err = runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.write, Input: input}); err != nil {
		return acl.Document{}, classify(err)
	}
	// Return a detached copy: callers cannot mutate committed policy state.
	committed := acl.Document{Resource: r, Revision: int64(next)}
	if err = json.Unmarshal(body, &committed.Policies); err != nil {
		return acl.Document{}, errors.New("decode committed policy")
	}
	return committed, nil
}

// classify maps writer outcomes onto the generic store contract without
// leaking component internals: revision conflicts are ErrConflict, rejected
// activations are ErrDenied, everything else is reported as is.
func classify(err error) error {
	var conflict *xhandler.Conflict
	if errors.As(err, &conflict) {
		return fmt.Errorf("%w: %s", acl.ErrConflict, conflict.Reason)
	}
	switch xresponse.ErrorStatusCode(err, 0) {
	case 409:
		return fmt.Errorf("%w: %v", acl.ErrConflict, err)
	case 403, 422:
		return fmt.Errorf("%w: %v", acl.ErrDenied, err)
	}
	return err
}
