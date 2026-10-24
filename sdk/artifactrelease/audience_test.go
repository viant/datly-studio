package artifactrelease_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/viant/authz"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type audienceFacts struct{ role string }

func (p audienceFacts) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "recipient", Tenant: "tenant", Issuer: "tenant", Roles: []string{p.role}, ValidUntil: time.Now().Add(time.Hour)}, nil
}

func TestAudienceEvaluatorUsesActiveCompiledAuthzJoin(t *testing.T) {
	ctx := context.Background()
	publisher := artifactcohort.NewPublisher()
	if _, err := publisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	root, _, err := publisher.Publish(ctx, artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1}, json.RawMessage(`{"title":"Home"}`), func(context.Context, json.RawMessage) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := contract.NewBundle(ctx, "demo", []artifactcohort.RootRef{root.Root}, func(_ context.Context, ref artifactcohort.RootRef) (*artifactcohort.Artifact, error) {
		return publisher.Get(ref)
	})
	if err != nil {
		t.Fatal(err)
	}
	policy := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	ref := authz.NamedReference{Resource: policy, Revision: 1}
	definition := authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
	}}}}}
	key := contract.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo", Name: "main"}
	channel, err := contract.NewChannel(key, "", []authz.NamedReference{ref})
	if err != nil {
		t.Fatal(err)
	}
	channel, err = contract.PreparePromotion(channel, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := contract.CompileCohorts(ctx, channel, bundle, []authz.Document{definition})
	if err != nil {
		t.Fatal(err)
	}
	before, err := publisher.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publisher.ReplaceCatalog(ctx, compiled.Policies, compiled.Cohorts, before.Generation); err != nil {
		t.Fatal(err)
	}
	operator := sdkrelease.AudienceEvaluator(publisher, audienceFacts{role: "operator"}, nil)
	allowed, err := operator(ctx, channel, ref)
	if err != nil || !allowed {
		t.Fatalf("named recipient denied: %t %v", allowed, err)
	}
	viewer := sdkrelease.AudienceEvaluator(publisher, audienceFacts{role: "viewer"}, nil)
	allowed, err = viewer(ctx, channel, ref)
	if err != nil || allowed {
		t.Fatalf("unassigned user allowed: %t %v", allowed, err)
	}
	before, err = publisher.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publisher.ReplaceCatalog(ctx, nil, nil, before.Generation); err != nil {
		t.Fatal(err)
	}
	allowed, err = operator(ctx, channel, ref)
	if err != nil || allowed {
		t.Fatalf("removed release still allowed: %t %v", allowed, err)
	}
}
