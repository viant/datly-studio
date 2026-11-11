package host

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/jsonrpc"
)

type publicNotificationSink struct{ events []*jsonrpc.Notification }

func (s *publicNotificationSink) Notify(_ context.Context, event *jsonrpc.Notification) error {
	s.events = append(s.events, event)
	return nil
}

func TestPublicSourceReadSubscribesBeforeServingAndHidesPrivateNotificationURIs(t *testing.T) {
	ctx := context.Background()
	publisher := artifactcohort.NewPublisher()
	artifact, _, err := publisher.Publish(ctx, artifactcohort.RootRef{URI: "style://shared/base", Kind: "style", Version: 1}, json.RawMessage(`{"schemaVersion":1,"css":""}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	_, err = publisher.ReplaceCatalog(ctx, nil, []artifactcohort.Cohort{{Tenant: "tenant", Name: "public", Public: true, Roots: []artifactcohort.RootRef{artifact.Root}}}, publisher.Generation())
	require.NoError(t, err)
	source := NewPublishedArtifactCatalog(publisher)
	require.NoError(t, source.BindPublicReadScope("tenant", func(context.Context) error { return nil }))
	sink := &publicNotificationSink{}
	subscribed := false
	got, handled, err := source.ReadPublicResource(ctx, artifact.Root.URI, "", func() { source.SubscribePublicResources(sink); subscribed = true })
	require.NoError(t, err)
	require.True(t, handled)
	require.True(t, subscribed)
	require.Equal(t, artifact.Root, got.Root)
	_, _, err = source.Publish(ctx, artifactcohort.RootRef{URI: "style://shared/not-selected", Kind: "style", Version: 1}, json.RawMessage(`{"schemaVersion":1,"css":".private{}"}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	source.notify(ctx, artifactcohort.NotificationUpdated, "style://shared/not-selected")
	require.NotEmpty(t, sink.events)
	for _, event := range sink.events {
		require.Equal(t, artifactcohort.NotificationListChanged, event.Method)
		require.Empty(t, event.Params)
	}
	sink.events = nil
	policy := authz.Resource{Tenant: "tenant", Kind: "cohort", ID: "private", Version: "logical"}
	doc := authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}
	_, err = source.ReplaceCatalog(ctx, []authz.Document{doc}, []artifactcohort.Cohort{{Tenant: "tenant", Name: "private", PolicyResource: policy, Roots: []artifactcohort.RootRef{artifact.Root}}}, publisher.Generation())
	require.NoError(t, err)
	require.NotEmpty(t, sink.events)
	for _, event := range sink.events {
		require.Equal(t, artifactcohort.NotificationListChanged, event.Method)
		require.Empty(t, event.Params)
	}
	source.UnsubscribeResources(sink)
	before := len(sink.events)
	source.NotifyListChanged(ctx)
	require.Len(t, sink.events, before)
}

func TestPublicSourcePreservesUninitializedAndCanceledOutcomes(t *testing.T) {
	source := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	require.NoError(t, source.BindPublicReadScope("tenant", func(context.Context) error { return nil }))
	_, err := source.DiscoverPublic(context.Background(), nil)
	require.ErrorIs(t, err, artifactcohort.ErrUnavailable)
	require.NotErrorIs(t, err, artifactcohort.ErrDenied)
	_, handled, err := source.ReadPublicResource(context.Background(), "style://shared/base", "", nil)
	require.True(t, handled)
	require.ErrorIs(t, err, artifactcohort.ErrUnavailable)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = source.DiscoverPublic(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
}
