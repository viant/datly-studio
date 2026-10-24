package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/datly-studio/sdk/artifactrelease"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/exec"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	"github.com/viant/xdatly/response"
)

type releaseTestSession struct{ binder xhandler.Binder }

func (s releaseTestSession) Binder() xhandler.Binder   { return s.binder }
func (s releaseTestSession) Response() response.Writer { return nil }

type releaseTestBinder struct{ invoker exec.ComponentInvoker }

func (b releaseTestBinder) Bind(context.Context, any) error { return nil }
func (b releaseTestBinder) Lookup(_ context.Context, key xhandler.ValueKey) (any, bool, error) {
	if key == exec.ComponentInvokerKey {
		return b.invoker, true, nil
	}
	return nil, false, nil
}

type releaseTestInvoker struct {
	rows []*stored.StoredNamespace
	err  error
}

func (i releaseTestInvoker) InvokeComponent(_ context.Context, request exec.ComponentRequest) (any, error) {
	if i.err != nil {
		return nil, i.err
	}
	input, ok := request.Input.(*stored.Input)
	if !ok || input.NamespaceId == "" || request.Target.Route.Path != "/_studio/namespace-store/read" {
		return nil, errors.New("release namespace lookup did not use the scoped reader")
	}
	return &stored.Output{Namespaces: i.rows}, nil
}

type catalogReleaseFixture struct{ namespace string }

var _ artifactrelease.Manager = (*catalogReleaseFixture)(nil)

func (f *catalogReleaseFixture) List(_ context.Context, in contract.ListInput) (contract.ListOutput, error) {
	f.namespace = in.Namespace
	return contract.ListOutput{}, nil
}
func (*catalogReleaseFixture) Candidates(context.Context, contract.CandidatesInput) (contract.CandidatesOutput, error) {
	return contract.CandidatesOutput{}, nil
}
func (*catalogReleaseFixture) Publish(context.Context, contract.PublishInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (*catalogReleaseFixture) Promote(context.Context, contract.PromoteInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (*catalogReleaseFixture) History(context.Context, contract.HistoryInput) (contract.HistoryOutput, error) {
	return contract.HistoryOutput{}, nil
}

func TestNativeReleaseToolsNeedInstalledManagerAndVerifiedPrincipal(t *testing.T) {
	defer ConfigureReleaseManagement(nil)
	ctx := context.Background()
	claims := &jwt.Claims{}
	claims.Subject = "publisher"
	verified := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "publisher"}}
	input := &ReleaseListInput{ReleaseIdentity: ReleaseIdentity{Jwt: claims, Auth: verified}, Request: contract.ListInput{Namespace: "demo"}}
	if ReleaseManagementAvailable() {
		t.Fatal("unconfigured release advertised")
	}
	if err := runReleaseList(ctx, nil, input, new(ReleaseListOutput)); err == nil {
		t.Fatal("unconfigured release operation succeeded")
	}
	manager := &catalogReleaseFixture{}
	ConfigureReleaseManagement(manager)
	if !ReleaseManagementAvailable() {
		t.Fatal("configured release hidden")
	}
	if err := runReleaseList(ctx, nil, input, new(ReleaseListOutput)); err != nil || manager.namespace != "demo" {
		t.Fatalf("verified release list: %v namespace=%q", err, manager.namespace)
	}
	input.Auth = &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "other"}}
	if err := runReleaseList(ctx, nil, input, new(ReleaseListOutput)); err == nil {
		t.Fatal("mismatched ingress identity reached manager")
	}
}

func TestNativeReleaseResolverRequiresVisibleMatchingNamespace(t *testing.T) {
	defer ConfigureReleaseManagement(nil)
	const namespaceID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	ctx := context.Background()
	claims := &jwt.Claims{}
	claims.Subject = "publisher"
	verified := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "publisher"}}
	manager := &catalogReleaseFixture{}
	resolved := ""
	ConfigureReleaseManagementResolver(func(_ context.Context, id string) (artifactrelease.Manager, error) {
		resolved = id
		return manager, nil
	})
	input := &ReleaseListInput{ReleaseIdentity: ReleaseIdentity{Jwt: claims, Auth: verified, NamespaceID: namespaceID}, Request: contract.ListInput{Namespace: "demo"}}
	session := releaseTestSession{binder: releaseTestBinder{invoker: releaseTestInvoker{rows: []*stored.StoredNamespace{{NamespaceId: namespaceID, Name: "demo"}}}}}
	if err := runReleaseList(ctx, session, input, new(ReleaseListOutput)); err != nil || resolved != namespaceID || manager.namespace != "demo" {
		t.Fatalf("visible selected namespace was not routed: %v id=%q name=%q", err, resolved, manager.namespace)
	}
	resolved = ""
	session.binder = releaseTestBinder{invoker: releaseTestInvoker{rows: []*stored.StoredNamespace{{NamespaceId: namespaceID, Name: "other"}}}}
	if err := runReleaseList(ctx, session, input, new(ReleaseListOutput)); err == nil || resolved != "" {
		t.Fatalf("mismatched namespace name reached resolver: %v id=%q", err, resolved)
	}
	session.binder = releaseTestBinder{invoker: releaseTestInvoker{rows: nil}}
	if err := runReleaseList(ctx, session, input, new(ReleaseListOutput)); err == nil || resolved != "" {
		t.Fatalf("invisible namespace reached resolver: %v id=%q", err, resolved)
	}
}
