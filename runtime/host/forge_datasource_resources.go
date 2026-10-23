package host

import (
	"context"
	"encoding/json"
	"time"

	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
)

// Configured datasource resources remain actor-filtered even at resources/list.
// No registry entry turns static enumeration into an authorization bypass.
type stockResourceHandler struct {
	protoserver.Handler
	host *Service
}

func (h *stockResourceHandler) Implements(method string) bool {
	if len(h.host.windowPrimitiveSource.datasources) > 0 && (method == schema.MethodResourcesList || method == schema.MethodResourcesRead) {
		return true
	}
	return h.Handler.Implements(method)
}
func (h *stockResourceHandler) Initialize(ctx context.Context, request *schema.InitializeRequestParams, out *schema.InitializeResult) {
	h.Handler.Initialize(ctx, request, out)
	if len(h.host.windowPrimitiveSource.datasources) > 0 {
		out.Capabilities.Resources = &schema.ServerCapabilitiesResources{}
	}
}

// ListResourceTemplates advertises the optional complete stock-window closure.
// The template itself contains no actor-specific names or authorization facts;
// each read still performs current provider admission.
func (h *stockResourceHandler) ListResourceTemplates(ctx context.Context, request *jsonrpc.TypedRequest[*schema.ListResourceTemplatesRequest]) (*schema.ListResourceTemplatesResult, *jsonrpc.Error) {
	result, rpcErr := h.Handler.ListResourceTemplates(ctx, request)
	if rpcErr != nil {
		return nil, rpcErr
	}
	if result == nil {
		result = &schema.ListResourceTemplatesResult{}
	}
	if result.ResultType == "" && result.NextCursor == nil {
		result.ResultType = schema.ResultTypeComplete
	}
	result.CacheScope = schema.ListResourceTemplatesResultCacheScopePrivate
	result.TtlMs = 0
	mime := "application/json"
	result.ResourceTemplates = append(result.ResourceTemplates, schema.ResourceTemplate{Name: "stock-window-closure", UriTemplate: "bundle://{namespace}/{name}", MimeType: &mime})
	return result, nil
}
func (h *stockResourceHandler) Discover(ctx context.Context, out *schema.DiscoverResult) {
	if discovery, ok := h.Handler.(interface {
		Discover(context.Context, *schema.DiscoverResult)
	}); ok {
		discovery.Discover(ctx, out)
	}
	if len(h.host.windowPrimitiveSource.datasources) > 0 {
		out.Capabilities.Resources = &schema.ServerCapabilitiesResources{}
	}
}
func (h *stockResourceHandler) ListResources(ctx context.Context, request *jsonrpc.TypedRequest[*schema.ListResourcesRequest]) (*schema.ListResourcesResult, *jsonrpc.Error) {
	input := primitive.ListRequest{Limit: 100}
	if request != nil && request.Request != nil && request.Request.Params.Cursor != nil {
		input.Cursor = *request.Request.Params.Cursor
	}
	listed, err := h.host.windowPrimitives.List(ctx, "datasource", input)
	if err != nil {
		return nil, jsonrpc.NewInvalidRequest("configured datasource list denied", nil)
	}
	out := &schema.ListResourcesResult{Resources: []schema.Resource{}}
	mime := "application/json"
	for _, resource := range listed.Resources {
		out.Resources = append(out.Resources, schema.Resource{Uri: resource.URI, Name: resource.Name, MimeType: &mime, Meta: map[string]any{"providerIdentity": h.host.config.Forge.ProviderIdentity}})
	}
	if listed.NextCursor != "" {
		next := listed.NextCursor
		out.NextCursor = &next
	}
	return out, nil
}
func (h *stockResourceHandler) ReadResource(ctx context.Context, request *jsonrpc.TypedRequest[*schema.ReadResourceRequest]) (*schema.ReadResourceResult, *jsonrpc.Error) {
	if request == nil || request.Request == nil {
		return nil, jsonrpc.NewInvalidRequest("resource request required", nil)
	}
	uri, err := identity.ParseResourceURI(request.Request.Params.Uri)
	if err == nil && uri.Kind == "bundle" {
		return h.readWindowBundleResource(ctx, uri)
	}
	if err != nil || uri.Kind != "datasource" {
		return h.Handler.ReadResource(ctx, request)
	}
	get, err := h.host.windowPrimitives.Get(ctx, "datasource", primitive.GetRequest{URI: uri.String(), Revision: identity.WorkingCandidate})
	if err != nil {
		return nil, jsonrpc.NewInvalidRequest("configured datasource read denied", nil)
	}
	if err = h.host.signDatasourceExecutionProof(ctx, get); err != nil {
		return nil, jsonrpc.NewInvalidRequest("datasource proof unavailable", nil)
	}
	raw := get.Resource.DefinitionBytes
	if len(raw) == 0 {
		raw = get.Resource.Definition
	}
	if !json.Valid(raw) {
		return nil, jsonrpc.NewInternalError("datasource definition unavailable", nil)
	}
	entry, err := h.host.datasourceEntry(uri.String())
	if err != nil || !get.ResolvedResource.ValidUntil.After(time.Now()) {
		return nil, jsonrpc.NewInvalidRequest("datasource authority expired", nil)
	}
	ctx, cancel := context.WithDeadline(ctx, get.ResolvedResource.ValidUntil)
	defer cancel()
	fresh, current, err := h.host.datasourceResolver(entry, "describe").ReadResolved(ctx, *get.ResolvedResource)
	if err != nil || string(fresh) != string(raw) || !current.ValidUntil.After(time.Now()) {
		return nil, jsonrpc.NewInvalidRequest("datasource authority changed", nil)
	}
	if current.ValidUntil.Before(get.ResolvedResource.ValidUntil) {
		get.ResolvedResource.ValidUntil = current.ValidUntil
		var narrowCancel context.CancelFunc
		ctx, narrowCancel = context.WithDeadline(ctx, current.ValidUntil)
		defer narrowCancel()
		if err = h.host.signDatasourceExecutionProof(ctx, get); err != nil {
			return nil, jsonrpc.NewInvalidRequest("datasource proof unavailable", nil)
		}
	}
	if _, err = h.host.windowPrimitivePolicy.principal(ctx); err != nil || ctx.Err() != nil || !get.ResolvedResource.ValidUntil.After(time.Now()) || !current.ValidUntil.After(time.Now()) {
		return nil, jsonrpc.NewInvalidRequest("datasource authority changed", nil)
	}
	mime := "application/json"
	return &schema.ReadResourceResult{CacheScope: schema.ReadResourceResultCacheScopePrivate, ResultType: schema.ResultTypeComplete, TtlMs: 0, Contents: []schema.ReadResourceResultContentsElem{{Uri: uri.String(), MimeType: &mime, Text: string(raw), Meta: map[string]any{"providerIdentity": h.host.config.Forge.ProviderIdentity, "resource": get.ResolvedResource, "executionProof": get.ExecutionProof}}}}, nil
}
