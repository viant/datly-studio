package artifactrelease_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/authz"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"testing"
)

func BenchmarkAudienceCatalogScaling(b *testing.B) {
	for _, size := range []int{1, 100, 1000} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			ctx := context.Background()
			p := artifactcohort.NewPublisher()
			root, _, err := p.Publish(ctx, artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1}, json.RawMessage(`{"title":"Home"}`), func(context.Context, json.RawMessage) error { return nil })
			if err != nil {
				b.Fatal(err)
			}
			key := contract.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo", Name: "main"}
			ref := authz.NamedReference{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}, Revision: 1}
			channel := contract.Channel{Key: key, Revision: 1}
			name := contract.ReleaseCohortName(key, ref)
			docs := make([]authz.Document, 0, size)
			cohorts := make([]artifactcohort.Cohort, 0, size)
			for i := 0; i < size; i++ {
				n := name
				if i > 0 {
					n = fmt.Sprintf("unrelated-%d", i)
				}
				r := authz.Resource{Tenant: "tenant", Kind: "cohort", ID: n, Version: "logical"}
				docs = append(docs, authz.Document{Resource: r, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}}}})
				cohorts = append(cohorts, artifactcohort.Cohort{Tenant: "tenant", Name: n, PolicyResource: r, Roots: []artifactcohort.RootRef{root.Root}})
			}
			if _, err = p.ReplaceCatalog(ctx, docs, cohorts, p.Generation()); err != nil {
				b.Fatal(err)
			}
			evaluate := sdkrelease.AudienceEvaluator(p, audienceFacts{role: "operator"}, nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ok, err := evaluate(ctx, channel, ref)
				if err != nil || !ok {
					b.Fatal(ok, err)
				}
			}
		})
	}
}
