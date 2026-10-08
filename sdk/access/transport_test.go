package access

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk"
)

type failureStore struct {
	document          authz.Document
	readErr, writeErr error
}

func (s failureStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	return s.document, s.readErr
}
func (s failureStore) Replace(context.Context, authz.Document, int64, string) (authz.Document, error) {
	return authz.Document{}, s.writeErr
}

type failureProvider struct {
	facts authz.Facts
	err   error
}

func (p failureProvider) Resolve(context.Context) (authz.Facts, error) { return p.facts, p.err }

func TestPolicyTransportSharedFailureCategories(t *testing.T) {
	resource := authz.Resource{Kind: "component", ID: "orders", Version: "1", Tenant: "tenant"}
	document := authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{
		"viewAccess":   {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
		"manageAccess": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
	}}
	facts := authz.Facts{Subject: "owner", Issuer: "trusted", Tenant: "tenant", ValidUntil: time.Now().Add(time.Hour)}
	privateErr := errors.New("private storage detail")
	for _, tc := range []struct {
		name       string
		store      failureStore
		provider   failureProvider
		operations []string
		want       sdk.ErrorCode
	}{
		{"read unavailable", failureStore{document: document, readErr: privateErr}, failureProvider{facts: facts}, []string{OperationGet, OperationContext, OperationReplace}, sdk.ErrorUnavailable},
		{"identity denied", failureStore{document: document}, failureProvider{err: authz.ErrDenied}, []string{OperationGet, OperationContext}, sdk.ErrorUnauthorized},
		{"provider unavailable", failureStore{document: document}, failureProvider{err: privateErr}, []string{OperationGet, OperationContext, OperationReplace}, sdk.ErrorUnavailable},
		{"write unavailable", failureStore{document: document, writeErr: privateErr}, failureProvider{facts: facts}, []string{OperationReplace}, sdk.ErrorUnavailable},
		{"conflict", failureStore{document: document, writeErr: authz.ErrConflict}, failureProvider{facts: facts}, []string{OperationReplace}, sdk.ErrorConflict},
		{"view denied", failureStore{document: document}, failureProvider{facts: authz.Facts{Subject: "other", Issuer: "trusted", Tenant: "tenant", ValidUntil: facts.ValidUntil}}, []string{OperationGet, OperationContext, OperationReplace}, sdk.ErrorForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, operation := range tc.operations {
				transport := Transport{Service: &authz.Service{Store: tc.store, Provider: tc.provider}}
				var input any = resource
				var output any = &authz.Document{}
				if operation == OperationContext {
					output = &authz.EditorContext{}
				}
				if operation == OperationReplace {
					input = document
				}
				err := transport.Invoke(context.Background(), operation, input, output)
				var public *sdk.Error
				if !errors.As(err, &public) || public.Code != tc.want {
					t.Fatalf("%s want %s got %v", operation, tc.want, err)
				}
				if public.Message == privateErr.Error() {
					t.Fatal("private detail leaked into public message")
				}
			}
		})
	}
}
