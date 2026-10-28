package host

import (
	"context"
	"encoding/json"
	"errors"

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
	provider := ""
	if s.config.Forge != nil {
		provider = s.config.Forge.ProviderIdentity
	}
	return &primitive.NamespaceCapabilities{ProviderIdentity: provider, Namespace: request.Namespace, Sync: &primitive.NamespaceSyncResult{Catalog: raw, FetchMethod: artifactcohort.MethodFetch}}, nil
}

// Advertisement is capability metadata only. Policies and cohort contents are
// returned solely by an explicit host-authorized synchronization request.
func (s *Service) advertiseNamespaceSynchronization(listed *primitive.NamespaceListResult) {
	if listed == nil || s.config.ArtifactCohorts == nil || s.config.AuthorizeArtifactHost == nil {
		return
	}
	claims, ok := s.config.ArtifactCohorts.(interface{ ArtifactNamespaces() []string })
	if !ok {
		return
	}
	allowed := map[string]bool{}
	for _, namespace := range claims.ArtifactNamespaces() {
		allowed[namespace] = true
	}
	for index := range listed.Namespaces {
		if !allowed[listed.Namespaces[index].Name] {
			continue
		}
		listed.Namespaces[index].Synchronization = &primitive.NamespaceSyncCapability{Version: 1, Method: "namespaces/get", FetchMethod: artifactcohort.MethodFetch}
	}
}

func namespaceSyncFailure(err error) map[string]string {
	code, message := "namespace_sync_unavailable", "Namespace synchronization is denied or unavailable; verify the configured host identity and namespace."
	switch {
	case errors.Is(err, artifactcohort.ErrInvalid):
		code = "namespace_sync_request_invalid"
		message = "Invalid namespace synchronization request; use its logical namespace name and a valid knownHash."
	case errors.Is(err, artifactcohort.ErrDenied):
		code = "namespace_sync_denied"
		message = "Configured host access to this published namespace is required."
	case errors.Is(err, artifactcohort.ErrUnsupported):
		message = "Published namespace synchronization is not configured."
	}
	return map[string]string{"code": code, "message": message}
}
