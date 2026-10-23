package host

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
)

// PublishedArtifactCatalog adapts a trusted, hydrated Forge publisher to the
// Datly Studio MCP. It owns no user grant cache: discover/fetch are authorized
// once at the MCP handler using Config.AuthorizeArtifactHost. The publisher
// stores producer-closed top-level versions and their complete policy catalog.
type PublishedArtifactCatalog struct {
	Publisher *artifactcohort.Publisher
	mu        sync.Mutex
	next      uint64
	sinks     map[uint64]transport.Notifier
}

func NewPublishedArtifactCatalog(p *artifactcohort.Publisher) *PublishedArtifactCatalog {
	return &PublishedArtifactCatalog{Publisher: p}
}
func (s *PublishedArtifactCatalog) Discover(ctx context.Context) (artifactcohort.Manifest, error) {
	if s == nil || s.Publisher == nil || ctx == nil || ctx.Err() != nil {
		return artifactcohort.Manifest{}, artifactcohort.ErrUnavailable
	}
	manifest, err := s.Publisher.Snapshot()
	if ctx.Err() != nil {
		return artifactcohort.Manifest{}, artifactcohort.ErrUnavailable
	}
	return manifest, err
}
func (s *PublishedArtifactCatalog) Fetch(ctx context.Context, input artifactcohort.FetchInput) (*artifactcohort.Artifact, error) {
	if s == nil || s.Publisher == nil || input.Tenant == "" || ctx == nil || ctx.Err() != nil {
		return nil, artifactcohort.ErrUnavailable
	}
	manifest, err := s.Publisher.Snapshot()
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, cohort := range manifest.Cohorts {
		if cohort.Tenant != input.Tenant {
			continue
		}
		// Revision overrides select bytes for an already-authorized root; they
		// do not grant a user access, but the trusted host must be able to stage
		// the exact retained version named by the active manifest.
		for _, root := range append(append([]artifactcohort.RootRef(nil), cohort.Roots...), cohort.RevisionOverrides...) {
			if root == input.Root {
				allowed = true
				break
			}
		}
		if allowed {
			break
		}
	}
	if !allowed {
		return nil, artifactcohort.ErrDenied
	}
	artifact, err := s.Publisher.Get(input.Root)
	if ctx.Err() != nil {
		return nil, artifactcohort.ErrUnavailable
	}
	return artifact, err
}
func (s *PublishedArtifactCatalog) Publish(ctx context.Context, root artifactcohort.RootRef, payload json.RawMessage, validate func(context.Context, json.RawMessage) error) (*artifactcohort.Artifact, uint64, error) {
	if s == nil || s.Publisher == nil {
		return nil, 0, artifactcohort.ErrUnavailable
	}
	before, beforeErr := s.Publisher.Snapshot()
	artifact, generation, err := s.Publisher.Publish(ctx, root, payload, validate)
	if err == nil {
		after, afterErr := s.Publisher.Snapshot()
		if afterErr == nil && (beforeErr != nil || after.Hash != before.Hash) {
			s.notify(ctx, artifactcohort.NotificationUpdated, artifact.Root.URI)
		}
	}
	return artifact, generation, err
}
func (s *PublishedArtifactCatalog) ReplaceCatalog(ctx context.Context, policies []authz.Document, cohorts []artifactcohort.Cohort, expectedGeneration uint64) (artifactcohort.Manifest, error) {
	if s == nil || s.Publisher == nil {
		return artifactcohort.Manifest{}, artifactcohort.ErrUnavailable
	}
	manifest, err := s.Publisher.ReplaceCatalog(ctx, policies, cohorts, expectedGeneration)
	if err == nil {
		s.notify(ctx, artifactcohort.NotificationListChanged, "")
	}
	return manifest, err
}

// RemoveRoot withdraws a published top-level URI from every cohort after the
// producer has committed its archive decision. Immutable bytes may remain in
// Publisher storage, but Fetch can no longer serve them without an active
// cohort reference. Root-only removal changes only the URI's partition.
func (s *PublishedArtifactCatalog) RemoveRoot(ctx context.Context, uri string) error {
	if s == nil || s.Publisher == nil || ctx == nil || ctx.Err() != nil || uri == "" {
		return artifactcohort.ErrInvalid
	}
	for attempt := 0; attempt < 8; attempt++ {
		manifest, err := s.Publisher.Snapshot()
		if err != nil {
			return err
		}
		removed := false
		for i := range manifest.Cohorts {
			roots := manifest.Cohorts[i].Roots
			kept := roots[:0:0]
			for _, root := range roots {
				if root.URI == uri {
					removed = true
					continue
				}
				kept = append(kept, root)
			}
			manifest.Cohorts[i].Roots = kept
			overrides := manifest.Cohorts[i].RevisionOverrides
			keptOverrides := overrides[:0:0]
			for _, selected := range overrides {
				if selected.URI == uri {
					removed = true
					continue
				}
				keptOverrides = append(keptOverrides, selected)
			}
			manifest.Cohorts[i].RevisionOverrides = keptOverrides
			denied := manifest.Cohorts[i].DeniedURIs
			keptDenied := denied[:0:0]
			for _, selected := range denied {
				if selected == uri {
					removed = true
					continue
				}
				keptDenied = append(keptDenied, selected)
			}
			manifest.Cohorts[i].DeniedURIs = keptDenied
			follows := manifest.Cohorts[i].FollowPublished
			keptFollows := follows[:0:0]
			for _, selected := range follows {
				if selected == uri {
					removed = true
					continue
				}
				keptFollows = append(keptFollows, selected)
			}
			manifest.Cohorts[i].FollowPublished = keptFollows
		}
		if !removed {
			return nil
		}
		_, err = s.Publisher.ReplaceCatalog(ctx, manifest.Policies, manifest.Cohorts, manifest.Generation)
		if errors.Is(err, artifactcohort.ErrConflict) {
			continue
		}
		if err != nil {
			return err
		}
		s.notify(ctx, artifactcohort.NotificationUpdated, uri)
		return nil
	}
	return artifactcohort.ErrConflict
}
func (s *PublishedArtifactCatalog) SubscribeResources(sink transport.Notifier) func() {
	if s == nil || sink == nil {
		return func() {}
	}
	s.mu.Lock()
	s.next++
	id := s.next
	if s.sinks == nil {
		s.sinks = map[uint64]transport.Notifier{}
	}
	s.sinks[id] = sink
	s.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { s.mu.Lock(); delete(s.sinks, id); s.mu.Unlock() }) }
}

// UnsubscribeResources is called by the MCP final-session hook when the
// optional catalog is used with an injected Forge provider rather than the
// stock LocalProvider's own resource-change subscription.
func (s *PublishedArtifactCatalog) UnsubscribeResources(sink transport.Notifier) {
	if s == nil || sink == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, current := range s.sinks {
		if current == sink {
			delete(s.sinks, id)
		}
	}
}
func (s *PublishedArtifactCatalog) notify(ctx context.Context, method, uri string) {
	s.mu.Lock()
	sinks := make([]transport.Notifier, 0, len(s.sinks))
	for _, sink := range s.sinks {
		sinks = append(sinks, sink)
	}
	s.mu.Unlock()
	var params json.RawMessage
	if uri != "" {
		params, _ = json.Marshal(map[string]string{"uri": uri})
	}
	for _, sink := range sinks {
		_ = sink.Notify(ctx, &jsonrpc.Notification{Jsonrpc: jsonrpc.Version, Method: method, Params: params})
	}
}
