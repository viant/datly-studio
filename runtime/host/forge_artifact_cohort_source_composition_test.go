package host

import (
	"context"
	"errors"
	"testing"

	"encoding/json"
	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/types"
)

func TestPublishedCatalogRemoveRootClearsComposedReferences(t *testing.T) {
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	uri := "window://datastudio/connectors"
	validate := func(_ context.Context, raw json.RawMessage) error {
		var value types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		return value.Validate()
	}
	first, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 1}, closedWindowForCohort(t, "v1"), validate)
	if err != nil {
		t.Fatal(err)
	}
	second, generation, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 2}, closedWindowForCohort(t, "v2"), validate)
	if err != nil {
		t.Fatal(err)
	}
	policies := []authz.Document{}
	cohorts := []artifactcohort.Cohort{}
	for _, name := range []string{"standard", "preview", "restricted"} {
		policy := authz.Resource{Tenant: "verified-issuer", Kind: "cohort", ID: "datastudio/" + name, Version: "logical"}
		policies = append(policies, authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "verified-user"}}}})
		cohort := artifactcohort.Cohort{Tenant: "verified-issuer", Name: name, PolicyResource: policy}
		switch name {
		case "standard":
			cohort.Roots = []artifactcohort.RootRef{{URI: uri, Kind: "window"}}
		case "preview":
			cohort.Priority = 5
			cohort.RevisionOverrides = []artifactcohort.RootRef{{URI: uri, Kind: "window", Version: 1}}
		case "restricted":
			cohort.Priority = 10
			cohort.DeniedURIs = []string{uri}
		}
		cohorts = append(cohorts, cohort)
	}
	manifest, err := catalog.ReplaceCatalog(ctx, policies, cohorts, generation)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cohorts) != 3 {
		t.Fatal("composed source catalog was not published")
	}
	if err := catalog.RemoveRoot(ctx, uri); err != nil {
		t.Fatal(err)
	}
	removed, err := catalog.Discover(ctx)
	if err != nil || artifactcohort.ValidateManifest(removed) != nil {
		t.Fatalf("removed manifest invalid: %v", err)
	}
	for _, cohort := range removed.Cohorts {
		if len(cohort.Roots) != 0 || len(cohort.RevisionOverrides) != 0 || len(cohort.DeniedURIs) != 0 || len(cohort.FollowPublished) != 0 {
			t.Fatalf("removed URI retained in cohort %s", cohort.Name)
		}
	}
	for _, ref := range []artifactcohort.RootRef{first.Root, second.Root} {
		if _, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "verified-issuer", Root: ref}); !errors.Is(err, artifactcohort.ErrDenied) {
			t.Fatalf("removed revision remained fetchable: %v", err)
		}
	}
}

func TestPublishedCatalogFetchesExactOverrideOnlyRevisionForTrustedHost(t *testing.T) {
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	uri := "window://datastudio/preview-only"
	validate := func(_ context.Context, raw json.RawMessage) error {
		var value types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		return value.Validate()
	}
	first, _, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 1}, closedWindowForCohort(t, "v1"), validate)
	if err != nil {
		t.Fatal(err)
	}
	second, generation, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: uri, Kind: "window", Version: 2}, closedWindowForCohort(t, "v2"), validate)
	if err != nil {
		t.Fatal(err)
	}
	policy := authz.Resource{Tenant: "verified-issuer", Kind: "cohort", ID: "datastudio/preview", Version: "logical"}
	document := authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "verified-user"}}}}
	manifest, err := catalog.ReplaceCatalog(ctx, []authz.Document{document}, []artifactcohort.Cohort{{Tenant: "verified-issuer", Name: "preview", PolicyResource: policy, RevisionOverrides: []artifactcohort.RootRef{{URI: uri, Kind: "window", Version: 1}}}}, generation)
	if err != nil || len(manifest.Cohorts) != 1 || len(manifest.Cohorts[0].Roots) != 0 || len(manifest.Cohorts[0].RevisionOverrides) != 1 {
		t.Fatalf("override-only catalog: %+v %v", manifest.Cohorts, err)
	}
	fetched, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "verified-issuer", Root: first.Root})
	if err != nil || fetched.Root != first.Root || string(fetched.Payload) != string(first.Payload) {
		t.Fatalf("exact retained override was not fetchable: %+v %v", fetched, err)
	}
	badHash := first.Root
	badHash.Hash = second.Root.Hash
	for _, input := range []artifactcohort.FetchInput{{Tenant: "verified-issuer", Root: second.Root}, {Tenant: "verified-issuer", Root: badHash}, {Tenant: "other", Root: first.Root}} {
		if _, err := catalog.Fetch(ctx, input); !errors.Is(err, artifactcohort.ErrDenied) {
			t.Fatalf("undeclared root became fetchable: %+v %v", input, err)
		}
	}
}
