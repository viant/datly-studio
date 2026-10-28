package catalog

import (
	"context"
	"encoding/json"
	"os"

	"github.com/viant/datly-studio/internal/artifactnamespace"
	"github.com/viant/forge/backend/artifactcohort"
	primitive "github.com/viant/forge/backend/provider/contract"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// Explicit wrappers extend namespace discovery without changing its ordinary
// URI, resource-summary, or pagination contract. No policy catalog appears on
// an ordinary list or get.
type namespaceIndexEntry struct {
	namespaceext.Namespace
	Synchronization *primitive.NamespaceSyncCapability `json:"synchronization,omitempty"`
}
type namespaceListWire struct {
	Namespaces []namespaceIndexEntry `json:"namespaces"`
	NextCursor *string               `json:"nextCursor,omitempty"`
}
type namespaceGetWire struct {
	*namespaceext.GetResult
	ProviderIdentity string                         `json:"providerIdentity,omitempty"`
	Sync             *primitive.NamespaceSyncResult `json:"sync,omitempty"`
}

func namespaceSynchronizationClaims() map[string]bool {
	publishedArtifactSource.RLock()
	source, authorize := publishedArtifactSource.source, publishedArtifactSource.authorize
	publishedArtifactSource.RUnlock()
	result := map[string]bool{}
	if source == nil || authorize == nil {
		return result
	}
	if claims, ok := source.(interface{ ArtifactNamespaces() []string }); ok {
		for _, namespace := range claims.ArtifactNamespaces() {
			result[namespace] = true
		}
	}
	return result
}

func marshalNamespaceList(result *namespaceext.ListResult) ([]byte, error) {
	if result == nil {
		return []byte("null"), nil
	}
	wire := namespaceListWire{Namespaces: make([]namespaceIndexEntry, 0, len(result.Namespaces)), NextCursor: result.NextCursor}
	claims := namespaceSynchronizationClaims()
	for _, namespace := range result.Namespaces {
		entry := namespaceIndexEntry{Namespace: namespace}
		if claims[namespace.Name] {
			entry.Synchronization = &primitive.NamespaceSyncCapability{Version: 1, Method: "namespaces/get", FetchMethod: artifactcohort.MethodFetch}
		}
		wire.Namespaces = append(wire.Namespaces, entry)
	}
	return json.Marshal(wire)
}
func runNamespaceSync(ctx context.Context, session xhandler.Session, input *GetInput, output *GetOutput) error {
	if input.Sync == nil || input.Sync.KnownHash != "" && !validArtifactHash(input.Sync.KnownHash) || input.Request.Cursor != nil || input.Request.Kind != "" {
		return fail(400, "invalid namespace synchronization request")
	}
	source, err := artifactSource(ctx, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	namespace := input.Namespace
	if input.Request.URI != "" {
		id, err := ParseNamespaceURI(input.Request.URI)
		if err != nil {
			return err
		}
		catalogs, err := managementCatalogs(input.Jwt, input.Auth)
		if err != nil {
			return err
		}
		resolved := ""
		for _, catalog := range catalogs {
			if catalog.namespace.URI == input.Request.URI {
				resolved = catalog.namespace.Name
				break
			}
		}
		if resolved == "" {
			row, err := visible(ctx, session, input.Jwt, input.Auth, id)
			if err != nil {
				return err
			}
			resolved = row.Name
		}
		if namespace != "" && namespace != resolved {
			return fail(400, "namespace synchronization identity mismatch")
		}
		namespace = resolved
	}
	if namespace == "" {
		return fail(400, "namespace synchronization name is required")
	}
	if _, err := artifactcohort.RootPartition(artifactcohort.RootRef{URI: "window://" + namespace + "/probe", Kind: "window"}); err != nil {
		return fail(400, "invalid namespace synchronization name")
	}
	manifest, err := source.Discover(ctx)
	if err != nil || artifactcohort.ValidateManifest(manifest) != nil || artifactnamespace.ValidateManifestScope(ctx, manifest, namespace) != nil {
		return fail(403, "published namespace catalog is unavailable")
	}
	var value any = manifest
	if input.Sync.KnownHash == manifest.Hash {
		value = map[string]any{"hash": manifest.Hash, "unchanged": true}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	output.Response = nil
	output.ProviderIdentity = namespaceSyncProviderIdentity(namespace)
	output.Sync = &primitive.NamespaceSyncResult{Catalog: raw, FetchMethod: artifactcohort.MethodFetch}
	return nil
}

// Datly deliberately treats a nil public Payload as JSON null. Supply an
// explicit, actionable body for synchronization failures without exposing the
// underlying host/configuration diagnostics or policy catalog.
func publicNamespaceSyncError(err error) error {
	if err == nil {
		return nil
	}
	status := xresponse.ErrorStatusCode(err, 500)
	code, message := "namespace_sync_unavailable", "Published namespace synchronization is unavailable; check producer configuration."
	switch status {
	case 400:
		code = "namespace_sync_request_invalid"
		message = "Invalid namespace synchronization request; use its logical namespace name and a valid knownHash."
	case 401, 403:
		code = "namespace_sync_denied"
		message = "Configured host access to this published namespace is required."
	case 503:
		message = "Published namespace synchronization is not configured or unavailable."
	}
	return &xresponse.Error{Code: status, Payload: map[string]string{"code": code, "message": message}, Cause: err}
}

func namespaceSyncProviderIdentity(namespace string) string {
	managementSettings.RLock()
	defer managementSettings.RUnlock()
	if config := managementSettings.config; config != nil && config.Namespace == namespace {
		return config.ProviderIdentity
	}
	for _, config := range managementSettings.additional {
		if config != nil && config.Namespace == namespace {
			return config.ProviderIdentity
		}
	}
	if os.Getenv("STUDIO_MANAGEMENT_NAMESPACE") == namespace {
		return os.Getenv("STUDIO_MANAGEMENT_PROVIDER_ID")
	}
	return ""
}
