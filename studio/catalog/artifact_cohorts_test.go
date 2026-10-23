package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/host"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/scy/auth/jwt"
)

type artifactHostGateKey struct{}

func TestNativeArtifactToolsRequireVerifiedConfiguredHost(t *testing.T) {
	defer ConfigurePublishedArtifactCohorts(nil, nil)
	ctx := context.Background()
	source := host.NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	root := artifactcohort.RootRef{URI: "window://datastudio/connectors", Kind: "window", Version: 1}
	published, generation, err := source.Publish(ctx, root, json.RawMessage(`{"window":"closed"}`), func(_ context.Context, raw json.RawMessage) error {
		if !json.Valid(raw) {
			return artifactcohort.ErrInvalid
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	policy := authz.Resource{Tenant: "issuer", Kind: "cohort", ID: "verified-management", Version: "logical"}
	manifest, err := source.ReplaceCatalog(ctx, []authz.Document{{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "*"}}}}}, []artifactcohort.Cohort{{Tenant: "issuer", Name: "verified-management", PolicyResource: policy, Roots: []artifactcohort.RootRef{{URI: root.URI, Kind: root.Kind}}}}, generation)
	if err != nil {
		t.Fatal(err)
	}
	if err := ConfigurePublishedArtifactCohorts(source, func(ctx context.Context) error {
		if ctx.Value(artifactHostGateKey{}) != "allowed-host" {
			return errors.New("host denied")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	claims := &jwt.Claims{}
	claims.Subject = "allowed-host"
	verified := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "allowed-host"}}
	input := &ArtifactDiscoverInput{Jwt: claims, Auth: verified}
	out := new(ArtifactDiscoverOutput)
	if err := runArtifactDiscover(ctx, nil, input, out); err == nil {
		t.Fatal("unconfigured caller received entire cohort index")
	}
	actor := context.WithValue(ctx, artifactHostGateKey{}, "allowed-host")
	if err := runArtifactDiscover(actor, nil, input, out); err != nil || out.Response.Hash != manifest.Hash {
		t.Fatalf("trusted host discovery: %v", err)
	}
	input.KnownHash = manifest.Hash
	if err := runArtifactDiscover(actor, nil, input, out); err != nil || !out.Response.Unchanged || len(out.Response.Cohorts) != 0 {
		t.Fatalf("conditional trusted discovery: %v", err)
	}
	if err := runArtifactDiscover(actor, nil, &ArtifactDiscoverInput{Jwt: claims, Auth: &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "other"}}}, out); err == nil {
		t.Fatal("mismatched verified principal accepted")
	}
	fetched := new(ArtifactFetchOutput)
	if err := runArtifactFetch(actor, nil, &ArtifactFetchInput{Jwt: claims, Auth: verified, Tenant: "issuer", Root: published.Root}, fetched); err != nil || fetched.Response == nil || fetched.Response.Root != published.Root {
		t.Fatalf("trusted exact fetch: %v", err)
	}
	wrong := published.Root
	wrong.Version++
	if err := runArtifactFetch(actor, nil, &ArtifactFetchInput{Jwt: claims, Auth: verified, Tenant: "issuer", Root: wrong}, fetched); err == nil {
		t.Fatal("unknown published revision fetched")
	}
}
