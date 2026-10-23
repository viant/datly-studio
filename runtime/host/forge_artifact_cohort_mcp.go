package host

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/forge/backend/artifactcohort"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
)

// artifactCohortHandler only advertises the optional capability. Its tools
// are registered on the existing authenticated Forge MCP handler below.
type artifactCohortHandler struct {
	protoserver.Handler
	notifier transport.Notifier
	source   interface {
		SubscribeResources(transport.Notifier) func()
	}
	subscriptionOnce sync.Once
}

func (h *artifactCohortHandler) activateNotifications() {
	if h == nil || h.source == nil || h.notifier == nil {
		return
	}
	h.subscriptionOnce.Do(func() { h.source.SubscribeResources(h.notifier) })
}

func (h *artifactCohortHandler) Initialize(ctx context.Context, request *schema.InitializeRequestParams, result *schema.InitializeResult) {
	h.Handler.Initialize(ctx, request, result)
	artifactCohortCapability(&result.Capabilities)
}
func (h *artifactCohortHandler) Discover(ctx context.Context, result *schema.DiscoverResult) {
	if discovery, ok := h.Handler.(interface {
		Discover(context.Context, *schema.DiscoverResult)
	}); ok {
		discovery.Discover(ctx, result)
	}
	artifactCohortCapability(&result.Capabilities)
}
func artifactCohortCapability(c *schema.ServerCapabilities) {
	if c.Extensions == nil {
		c.Extensions = map[string]map[string]interface{}{}
	}
	c.Extensions[artifactcohort.Capability] = map[string]interface{}{"schemaVersion": 1, "methods": []string{artifactcohort.MethodDiscover, artifactcohort.MethodFetch}}
}

func (s *Service) registerArtifactCohortTools(ctx context.Context, base *protoserver.DefaultHandler, activateNotifications func()) error {
	if s == nil || s.config.ArtifactCohorts == nil {
		return nil
	}
	if base == nil || s.config.AuthorizeArtifactHost == nil {
		return fmt.Errorf("artifact cohort requires authenticated MCP registry and host authorizer")
	}
	boundNamespace := ""
	if s.config.NamespaceID != "" {
		rows, err := (namespacemcp.SQLDefinitions{DB: s.studio}).LoadNamespace(ctx, s.config.NamespaceID)
		if err != nil || len(rows) != 1 || rows[0].NamespaceID != s.config.NamespaceID || !rows[0].Active || !rows[0].Enabled || rows[0].Name == "" {
			return fmt.Errorf("artifact cohort bound namespace unavailable")
		}
		boundNamespace = rows[0].Name
	}
	var discoverSchema schema.ToolInputSchema
	if err := discoverSchema.Load(&struct {
		KnownHash string `json:"knownHash,omitempty"`
	}{}); err != nil {
		return err
	}
	var fetchSchema schema.ToolInputSchema
	if err := fetchSchema.Load(&artifactcohort.FetchInput{}); err != nil {
		return err
	}
	register := func(name string, input schema.ToolInputSchema, run func(context.Context, map[string]any) (any, error)) error {
		base.Registry.RegisterToolWithSchema(name, "Read a trusted published artifact cohort", input, nil, func(ctx context.Context, request *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			if err := s.config.AuthorizeArtifactHost(ctx); err != nil {
				return artifactCohortFailure(), nil
			}
			value, err := run(ctx, request.Params.Arguments)
			if err != nil {
				return artifactCohortFailure(), nil
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return artifactCohortFailure(), nil
			}
			if activateNotifications != nil {
				activateNotifications()
			}
			failed := false
			return &schema.CallToolResult{IsError: &failed, StructuredContent: value, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}}, nil
		})
		return nil
	}
	if err := register(artifactcohort.MethodDiscover, discoverSchema, func(ctx context.Context, args map[string]any) (any, error) {
		if _, hasKnownHash := args["knownHash"]; len(args) > 1 || len(args) == 1 && !hasKnownHash {
			return nil, artifactcohort.ErrInvalid
		}
		knownHash := ""
		if supplied, exists := args["knownHash"]; exists {
			value, ok := supplied.(string)
			if !ok || !validArtifactKnownHash(value) {
				return nil, artifactcohort.ErrInvalid
			}
			knownHash = value
		}
		manifest, err := s.config.ArtifactCohorts.Discover(ctx)
		if err != nil || artifactcohort.ValidateManifest(manifest) != nil || validateArtifactManifestScope(ctx, manifest, boundNamespace) != nil {
			return nil, artifactcohort.ErrUnavailable
		}
		if knownHash != "" && manifest.Hash == knownHash {
			return map[string]any{"unchanged": true, "hash": manifest.Hash}, nil
		}
		return manifest, nil
	}); err != nil {
		return err
	}
	return register(artifactcohort.MethodFetch, fetchSchema, func(ctx context.Context, args map[string]any) (any, error) {
		var input artifactcohort.FetchInput
		if err := decodePrimitiveArgs(args, &input); err != nil || input.Tenant == "" || !artifactRootInNamespace(input.Root, boundNamespace) {
			return nil, artifactcohort.ErrInvalid
		}
		artifact, err := s.config.ArtifactCohorts.Fetch(ctx, input)
		if err != nil || artifactcohort.ValidateArtifact(artifact) != nil || artifact.Root != input.Root {
			return nil, artifactcohort.ErrUnavailable
		}
		return artifact, nil
	})
}

func validArtifactKnownHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func artifactRootInNamespace(root artifactcohort.RootRef, namespace string) bool {
	if namespace == "" {
		return true
	}
	uri, err := identity.ParseResourceURI(root.URI)
	return err == nil && uri.Namespace == namespace && uri.Kind == root.Kind
}

// A namespace listener returns an all-or-nothing scoped feed. Filtering an
// unscoped producer manifest would forge a new hash/generation and could leak
// policy documents; reject it instead. The configured source must publish one
// complete manifest for the persisted namespace name bound to NamespaceID.
func validateArtifactManifestScope(ctx context.Context, manifest artifactcohort.Manifest, namespace string) error {
	if namespace == "" {
		return nil
	}
	store, err := authz.NewStaticStore(manifest.Policies)
	if err != nil {
		return artifactcohort.ErrInvalid
	}
	used := map[authz.Resource]bool{}
	for _, cohort := range manifest.Cohorts {
		for _, root := range append(append([]artifactcohort.RootRef(nil), cohort.Roots...), cohort.RevisionOverrides...) {
			if !artifactRootInNamespace(root, namespace) {
				return artifactcohort.ErrDenied
			}
		}
		for _, rawURI := range append(append([]string(nil), cohort.FollowPublished...), cohort.DeniedURIs...) {
			uri, err := identity.ParseResourceURI(rawURI)
			if err != nil || namespace != "" && uri.Namespace != namespace {
				return artifactcohort.ErrDenied
			}
		}
		doc, err := store.Get(ctx, cohort.PolicyResource)
		if err != nil {
			return artifactcohort.ErrDenied
		}
		used[doc.Resource] = true
		_, named, err := authz.ResolveNamedDocument(ctx, store, doc)
		if err != nil {
			return artifactcohort.ErrDenied
		}
		for _, dependency := range named {
			used[dependency.Resource] = true
		}
	}
	for _, doc := range manifest.Policies {
		if !used[doc.Resource] {
			return artifactcohort.ErrDenied
		}
	}
	return nil
}
func artifactCohortFailure() *schema.CallToolResult {
	failed := true
	value := map[string]string{"code": "artifact_unavailable", "message": "published artifact unavailable"}
	return &schema.CallToolResult{IsError: &failed, StructuredContent: value, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: `{"code":"artifact_unavailable","message":"published artifact unavailable"}`}}}
}
