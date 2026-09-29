package preview

import (
	"context"
	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/sdk"
	access "github.com/viant/authz"
	"github.com/viant/datly/spec"
	"testing"
	"time"
)

type previewPolicyStore struct{ resource access.Resource }

func (s previewPolicyStore) Get(_ context.Context, r access.Resource) (access.Document, error) {
	if r != s.resource {
		return access.Document{}, access.ErrDenied
	}
	return access.Document{Resource: r, Revision: 1, Policies: map[string]access.Policy{"execute": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reader"}, EntityType: "publisher"}}}, nil
}
func (previewPolicyStore) Replace(context.Context, access.Document, int64, string) (access.Document, error) {
	return access.Document{}, access.ErrDenied
}

type previewFacts struct{ facts access.Facts }

func (p previewFacts) Resolve(context.Context) (access.Facts, error) { return p.facts, nil }
func TestPreviewContextPinsVersionAndVerifiedSubject(t *testing.T) {
	resource := access.Resource{Kind: "component", ID: "report", Version: "11", Tenant: "tenant"}
	facts := access.Facts{Subject: "alice", Tenant: "tenant", Issuer: "trusted", Roles: []string{"reader"}, Entities: []access.Entity{{Type: "publisher", ID: "127"}}, ValidUntil: time.Now().Add(time.Minute)}
	service := &access.Service{Store: previewPolicyStore{resource}, Provider: previewFacts{facts}}
	component := &spec.Component{Parameters: []*spec.Parameter{{Name: "Auth", Source: spec.BindSource{Kind: "component", Name: "GET:" + accesscontext.RoutePrefix + "report/publisher"}}}}
	ctx := sdk.WithVerifiedCredential(sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "alice"}), sdk.VerifiedCredential{Bearer: "verified-by-host", Claims: struct{}{}})
	d := Dynamic{Access: service, AccessTenant: "tenant"}
	registrations, err := d.previewContexts(ctx, &definition{ReportID: "report", VersionNo: 11}, component)
	if err != nil || len(registrations) != 1 {
		t.Fatalf("contexts=%d err=%v", len(registrations), err)
	}
	if _, err = d.previewContexts(ctx, &definition{ReportID: "report", VersionNo: 10}, component); err == nil {
		t.Fatal("wrong version authorized")
	}
	mismatch := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "bob"})
	if _, err = d.previewContexts(mismatch, &definition{ReportID: "report", VersionNo: 11}, component); err == nil {
		t.Fatal("different actor authorized")
	}
	if _, err = d.previewContexts(context.Background(), &definition{ReportID: "report", VersionNo: 11}, component); err == nil {
		t.Fatal("unverified actor authorized")
	}
	component.Parameters[0].Source.Name = "GET:" + accesscontext.RoutePrefix + "other/publisher"
	if _, err = d.previewContexts(ctx, &definition{ReportID: "report", VersionNo: 11}, component); err == nil {
		t.Fatal("foreign context authorized")
	}
}
