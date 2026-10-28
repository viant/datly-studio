package host

import (
	"context"
	"encoding/json"

	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/forge/backend/artifactcohort"
	primitive "github.com/viant/forge/backend/provider/contract"
)

func (s *Service) synchronizeNamespaceArtifacts(ctx context.Context, request primitive.NamespaceGetRequest) (*primitive.NamespaceCapabilities, error) {
	if request.Namespace == "" || request.Sync == nil || request.Sync.KnownHash != "" && !validArtifactKnownHash(request.Sync.KnownHash) {
		return nil, artifactcohort.ErrInvalid
	}
	if s.config.ArtifactCohorts == nil || s.config.AuthorizeArtifactHost == nil {
		return nil, artifactcohort.ErrUnsupported
	}
	if err := s.config.AuthorizeArtifactHost(ctx); err != nil {
		return nil, artifactcohort.ErrDenied
	}
	if s.config.NamespaceID != "" {
		rows, err := (namespacemcp.SQLDefinitions{DB: s.studio}).LoadNamespace(ctx, s.config.NamespaceID)
		if err != nil || len(rows) != 1 || rows[0].Name != request.Namespace || !rows[0].Active || !rows[0].Enabled {
			return nil, artifactcohort.ErrDenied
		}
	}
	manifest, err := s.config.ArtifactCohorts.Discover(ctx)
	if err != nil {
		return nil, err
	}
	if artifactcohort.ValidateManifest(manifest) != nil || validateArtifactManifestScope(ctx, manifest, request.Namespace) != nil {
		return nil, artifactcohort.ErrInvalid
	}
	var catalog any = manifest
	if request.Sync.KnownHash != "" && request.Sync.KnownHash == manifest.Hash {
		catalog = map[string]any{"unchanged": true, "hash": manifest.Hash}
	}
	raw, err := json.Marshal(catalog)
	if err != nil {
		return nil, err
	}
	return &primitive.NamespaceCapabilities{Namespace: request.Namespace, Sync: &primitive.NamespaceSyncResult{Catalog: raw, FetchMethod: artifactcohort.MethodFetch}}, nil
}
