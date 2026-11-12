package host

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/sourcevalidation"
)

func TestAtomicPublicationRecoversIncompatibleSiblingGeneration(t *testing.T) {
	sources := map[string]string{}
	policies := map[string]authz.Policy{}
	for _, name := range []string{"first", "second", "third"} {
		sources[name] = strings.ReplaceAll(unscopedRecordsDQL, "/records", "/"+name)
		policies[name] = authz.Policy{Mode: "public"}
	}
	fixture, err := newScopedHost(t, sources, policies)
	require.NoError(t, err)
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	transport := &sqltransport.Transport{DB: fixture.service.studio, Authorizer: sqltransport.AuthorizerFunc(func(context.Context, sqltransport.AuthorizationRequest) error { return nil }), Activator: fixture.service}
	client, err := sdk.NewClient(transport)
	require.NoError(t, err)
	token := fixture.token(t, "alice", []authz.Entity{})
	for _, name := range []string{"first", "second", "third"} {
		version, err := client.Versions().Create(ctx, name, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: sources[name]})
		require.NoError(t, err)
		_, err = client.Versions().Validate(ctx, name, version.VersionNo, version.SourceRevision)
		require.NoError(t, err)
		_, err = fixture.store.Provision(ctx, authz.Document{Resource: authz.Resource{Kind: "component", ID: name, Version: "2", Tenant: fixture.service.config.Access.Tenant}, Policies: map[string]authz.Policy{"execute": {Mode: "public"}}}, "fixture")
		require.NoError(t, err)
	}
	// A native host upgrade can reject formerly valid sources. One component
	// cannot be replaced alone when the candidate still contains old siblings.
	fixture.service.config.SourceValidator = func(_ context.Context, input sourcevalidation.Input) error {
		if input.VersionNo == 1 {
			return errors.New("old fixture source is incompatible")
		}
		return nil
	}
	_, err = client.Publications().Publish(ctx, "first", 2, sdk.PublishInput{ExpectedSourceRevision: 1})
	require.ErrorContains(t, err, "runtime")
	for _, name := range []string{"first", "second", "third"} {
		p, err := client.Publications().Get(ctx, name)
		require.NoError(t, err)
		require.Equal(t, 1, p.ActiveVersionNo)
	}
	result, err := client.Publications().Publish(ctx, "first", 2, sdk.PublishInput{ExpectedSourceRevision: 1, AdditionalVersions: []sdk.PublishVersion{
		{ReportID: "second", VersionNo: 2, ExpectedSourceRevision: 1},
		{ReportID: "third", VersionNo: 2, ExpectedSourceRevision: 1},
	}})
	require.NoError(t, err)
	for _, name := range []string{"first", "second", "third"} {
		p, err := client.Publications().Get(ctx, name)
		require.NoError(t, err)
		require.Equal(t, 2, p.ActiveVersionNo)
		require.Equal(t, result.ActiveGeneration, p.ActiveGeneration)
		status, body := fixture.get(t, "/"+name, token, nil)
		require.Equal(t, http.StatusOK, status, body)
		require.Contains(t, body, "alpha-101")
	}
}
