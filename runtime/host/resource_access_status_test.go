package host

import (
	"context"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accesscontext"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	xresponse "github.com/viant/xdatly/response"
	"strings"
	"testing"
)

type statusResourceStore struct {
	resourceStore
	err error
}

func (s statusResourceStore) Get(ctx context.Context, r authz.Resource) (authz.Document, error) {
	if s.err != nil {
		return authz.Document{}, s.err
	}
	return s.resourceStore.Get(ctx, r)
}

type statusResourceProvider struct{ err error }

func (p statusResourceProvider) Resolve(ctx context.Context) (authz.Facts, error) {
	if p.err != nil {
		return authz.Facts{}, p.err
	}
	return runtimeFacts{}.Resolve(ctx)
}
func TestResourceAuthorizationStatus(t *testing.T) {
	resource := authz.Resource{Kind: "component", ID: "tasks", Version: "1", Tenant: "one"}
	for _, tc := range []struct {
		name                  string
		storeErr, providerErr error
		role                  string
		status                int
	}{
		{name: "store outage", storeErr: errors.New("PRIVATE database failure"), role: "reader", status: 503},
		{name: "provider outage", providerErr: errors.New("PRIVATE authority failure"), role: "reader", status: 503},
		{name: "invalid identity", providerErr: authz.ErrDenied, role: "reader", status: 401},
		{name: "missing policy", storeErr: authz.ErrDenied, role: "reader", status: 403},
		{name: "policy denial", role: "administrator", status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{"execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: tc.role}, EntityType: "project"}}}
			service := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "one"}}, resourceAccess: &authz.Service{Store: statusResourceStore{resourceStore: resourceStore{documents: map[authz.Resource]authz.Document{resource: document}}, err: tc.storeErr}, Provider: statusResourceProvider{err: tc.providerErr}}}
			assertStatus := func(err error) {
				t.Helper()
				if xresponse.ErrorStatusCode(err, 0) != tc.status || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatalf("status=%d error=%v", xresponse.ErrorStatusCode(err, 0), err)
				}
			}
			assertStatus(service.authorizeResourcePolicy(context.Background(), resource, "execute"))
			assertStatus(service.authorizeComponentExecution(context.Background(), resource, true))
			component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Name: "Tasks"}, Parameters: []*spec.Parameter{{Name: "Auth", Source: spec.BindSource{Kind: string(spec.KindComponent), Name: "GET:/_studio/access/context/tasks/project"}}}}
			contexts, _, err := service.accessContexts([]*registry.RegisteredComponent{{Component: component}}, map[spec.Key]string{component.Key: "tasks"}, map[string]int{"tasks": 1})
			if err != nil {
				t.Fatal(err)
			}
			_, err = contexts[0].Handler.Execute(context.Background(), rhandler.Invocation{Input: &accesscontext.Input{}})
			assertStatus(err)
		})
	}
}
