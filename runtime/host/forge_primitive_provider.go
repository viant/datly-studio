package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	primitive "github.com/viant/agently-core/protocol/primitive"
	identity "github.com/viant/agently-core/protocol/resource"
	windowprotocol "github.com/viant/agently-core/protocol/window"
	coreresource "github.com/viant/agently-core/service/resource"
	"github.com/viant/forge/backend/types"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
)

// The canonical source binds presentation and exact native component contracts
// in one working candidate. It neither invents stamps nor changes logical URIs.
type stockPrimitiveSource struct {
	original         *stockWindowSource
	snapshot         *coreresource.NativeAssetSnapshot
	components       map[ComponentReference]string
	metadataCompiles atomic.Int64
	invalidated      atomic.Bool
}

func (s *stockPrimitiveSource) bytes(ctx context.Context, uri identity.ResourceURI) (json.RawMessage, error) {
	if err := s.authorizeComponents(ctx, uri); err != nil {
		return nil, err
	}
	if err := s.check(ctx, uri); err != nil {
		return nil, err
	}
	candidates, err := s.snapshot.Candidates(ctx, uri)
	if err != nil || len(candidates) != 1 {
		return nil, identity.ErrResourceStale
	}
	raw, err := s.snapshot.ReadCandidate(ctx, uri, candidates[0])
	if err != nil {
		return nil, err
	}
	if err := s.check(ctx, uri); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *stockPrimitiveSource) materialize(bundle stockWindowBundle, entry stockWindowEntry) (json.RawMessage, error) {
	uri := entry.uri
	file, err := decodeStockWindowFile(bundle.Source)
	if err != nil {
		return nil, err
	}
	w := *file.Window
	w.WindowKey, w.Namespace = entry.key, uri.Namespace
	w.Resource, w.ResourceTarget, w.AuthorizationSnapshot = nil, nil, nil
	w.DataSource = make(map[string]types.DataSource, len(file.DataSources))
	w.ResourceDependencies = make(map[string]string, len(file.DataSources))
	v := types.WindowResourceVariant{Window: &w, DataSources: make(map[string]json.RawMessage, len(file.DataSources))}
	for id, source := range file.DataSources {
		binding := bundle.Components[id].Binding
		copy := *source
		copy.Backend = &windowprotocol.Backend{Kind: "datly", Ownership: "provider", Method: "windows/datasource", Component: &binding, SchemaFingerprint: binding.SchemaFingerprint, Pinned: map[string]any{"dataSourceId": id}}
		raw, err := json.Marshal(copy)
		if err != nil {
			return nil, err
		}
		raw, err = types.CanonicalWindowDescriptor(raw)
		if err != nil {
			return nil, err
		}
		fingerprint, err := types.WindowDescriptorFingerprint(raw)
		if err != nil {
			return nil, err
		}
		presentation := source.DataSource
		w.DataSource[id] = presentation
		w.ResourceDependencies[id], v.DataSources[id] = fingerprint, raw
	}
	fingerprint, err := types.WindowVariantFingerprint(v)
	if err != nil {
		return nil, err
	}
	// These source files have no target-aware import mechanism. Advertise only
	// the actual default and explicit web/desktop rendering of the same source.
	envelope := types.WindowResourceEnvelope{SchemaVersion: 2, Format: types.WindowBundleFormat, Variants: map[string]types.WindowResourceVariant{fingerprint: v}, Targets: []types.WindowTargetBinding{{Variant: fingerprint}, {Target: types.WindowTarget{Platform: "web", FormFactor: "desktop"}, Variant: fingerprint}}}
	if err = envelope.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}
func (s *stockPrimitiveSource) Candidates(ctx context.Context, uri identity.ResourceURI) ([]identity.ResourceCandidate, error) {
	raw, err := s.bytes(ctx, uri)
	if err != nil {
		return nil, err
	}
	return []identity.ResourceCandidate{{Kind: identity.WorkingCandidate, ContentFingerprint: identity.ContentFingerprint(raw)}}, nil
}
func (s *stockPrimitiveSource) ReadCandidate(ctx context.Context, uri identity.ResourceURI, candidate identity.ResourceCandidate) (json.RawMessage, error) {
	if candidate.Kind != identity.WorkingCandidate || candidate.Revision != "" {
		return nil, identity.ErrResourceDenied
	}
	return s.bytes(ctx, uri)
}

type stockPrimitivePolicy struct {
	base   *stockWindowPolicy
	action string
}

func (p stockPrimitivePolicy) SelectRevision(ctx context.Context, ref identity.ResourceRef, candidates []identity.ResourceCandidate) (identity.ResourceDecision, error) {
	entry, ok := p.base.entries[ref.URI]
	if !ok {
		return identity.ResourceDecision{}, identity.ErrResourceDenied
	}
	if _, err := p.base.selectWindow(ctx, entry, p.action); err != nil {
		return identity.ResourceDecision{}, identity.ErrResourceDenied
	}
	return p.base.SelectRevision(ctx, ref, candidates)
}

func (s *Service) initWindowPrimitives(source *stockWindowSource, policy *stockWindowPolicy) error {
	if s.config.Forge.ProviderIdentity == "" {
		return nil
	}
	s.windowExecutionProof = s.config.Forge.ExecutionProof
	if s.windowExecutionProof == nil {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return err
		}
		proof, err := types.NewWindowTargetHMAC(key)
		if err != nil {
			return err
		}
		s.windowExecutionProof = proof
	}
	actor := func(ctx context.Context) (identity.VerifiedActor, error) {
		p, err := policy.principal(ctx)
		if err != nil {
			return identity.VerifiedActor{}, err
		}
		return identity.VerifiedActor{Subject: p.Facts.Subject, Issuer: p.Facts.Issuer, TenantID: p.Facts.Tenant, AccountID: p.AccountID, IdentityRevision: p.IdentityRevision, ValidUntil: p.Facts.ValidUntil}, nil
	}
	verify := func(ctx context.Context, expected identity.VerifiedActor) error {
		current, err := actor(ctx)
		if err != nil || !expected.Valid(time.Now()) || current.Subject != expected.Subject || current.Issuer != expected.Issuer || current.TenantID != expected.TenantID || current.AccountID != expected.AccountID || current.IdentityRevision != expected.IdentityRevision {
			return identity.ErrResourceDenied
		}
		return nil
	}
	canonical := &stockPrimitiveSource{original: source}
	if err := s.initPrimitiveMetadata(); err != nil {
		return err
	}
	if err := canonical.preload(); err != nil {
		_ = s.closePrimitiveMetadata(context.Background())
		return err
	}
	bindings := make([]coreresource.LocalResourceBinding, 0, len(source.entries))
	for _, entry := range source.entries {
		entry := entry
		bindings = append(bindings, coreresource.LocalResourceBinding{URI: entry.config.URI, Title: entry.uri.Name, FormatVersion: 2, Resolver: func(_ context.Context, _ identity.VerifiedActor, action string) (*identity.ResourceResolver, error) {
			selectedAction := "discover"
			if action == "resource.get" {
				selectedAction = "describe"
			}
			return &identity.ResourceResolver{ProviderIdentity: s.config.Forge.ProviderIdentity, Source: canonical, Policy: stockPrimitivePolicy{base: policy, action: selectedAction}}, nil
		}})
	}
	provider, err := coreresource.NewLocalProvider(coreresource.LocalConfig{ProviderIdentity: s.config.Forge.ProviderIdentity, Actor: actor, Verify: verify, Bindings: bindings, WindowIndex: canonical.index, WindowListVisibility: func(ctx context.Context, _ identity.VerifiedActor, binding coreresource.LocalResourceBinding) (bool, error) {
		entry := source.entries[binding.URI]
		_, err := policy.selectWindow(ctx, entry, "discover")
		return err == nil, nil
	}, Validators: map[string]coreresource.LocalResourceValidator{"window": coreresource.ValidateWindowBundle}, Authorize: func(ctx context.Context, _ identity.VerifiedActor, uri identity.ResourceURI, action string) error {
		if action == "namespace.list" || action == "namespace.get" {
			for _, entry := range source.entries {
				if entry.uri.Namespace == uri.Namespace {
					if _, err := policy.selectWindow(ctx, entry, "discover"); err == nil {
						return nil
					}
				}
			}
			return identity.ErrResourceDenied
		}
		entry, ok := source.entries[uri.String()]
		if !ok {
			return identity.ErrResourceDenied
		}
		if action == "resource.list" {
			_, err := policy.principal(ctx)
			return err
		}
		selectedAction := "discover"
		if action == "resource.get" {
			selectedAction = "describe"
		}
		if _, err := policy.selectWindow(ctx, entry, selectedAction); err != nil {
			return identity.ErrResourceDenied
		}
		return nil
	}})
	if err != nil {
		return err
	}
	s.windowPrimitives, s.windowPrimitiveSource, s.windowPrimitivePolicy = provider, canonical, policy
	return nil
}

type stockDatasourceInput struct {
	Resource       identity.ResolvedResource `json:"resource"`
	DataSourceID   string                    `json:"dataSourceId"`
	Inputs         map[string]any            `json:"inputs,omitempty"`
	ExecutionProof *primitive.ExecutionProof `json:"executionProof"`
}

func decodePrimitiveArgs(arguments map[string]any, target any) error {
	raw, err := json.Marshal(arguments)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("invalid primitive request")
	}
	return nil
}

func (s *Service) primitiveFetch(ctx context.Context, in stockDatasourceInput) (json.RawMessage, error) {
	if s.windowPrimitives == nil || in.Resource.ProviderIdentity != s.config.Forge.ProviderIdentity || in.DataSourceID == "" {
		return nil, identity.ErrResourceDenied
	}
	if err := s.verifyExecutionProof(ctx, in.Resource, in.ExecutionProof); err != nil {
		return nil, err
	}
	operationPin := in.Resource
	entry, ok := s.windowPrimitiveSource.original.entries[in.Resource.URI]
	if !ok {
		return nil, identity.ErrResourceDenied
	}
	resolver := &identity.ResourceResolver{ProviderIdentity: s.config.Forge.ProviderIdentity, Source: s.windowPrimitiveSource, Policy: stockPrimitivePolicy{base: s.windowPrimitivePolicy, action: "execute"}}
	approved, current, err := resolver.ReadResolved(ctx, operationPin)
	if err != nil {
		return nil, err
	}
	if current.ValidUntil.Before(operationPin.ValidUntil) {
		operationPin.ValidUntil = current.ValidUntil
	}
	variant, err := types.SelectWindowResource(approved, nil)
	if err != nil {
		return nil, err
	}
	if in.ExecutionProof.Binding != studioExecutionBindingDomain+variant.Fingerprint {
		return nil, identity.ErrResourceDenied
	}
	var descriptor windowprotocol.DataSource
	if json.Unmarshal(variant.DataSources[in.DataSourceID], &descriptor) != nil || descriptor.Backend == nil || descriptor.Backend.Component == nil {
		return nil, identity.ErrResourceDenied
	}
	ref, ok := entry.config.Components[in.DataSourceID]
	if !ok {
		return nil, identity.ErrResourceDenied
	}
	result, err := s.ExecuteComponentJSON(ctx, ref, *descriptor.Backend.Component, in.Inputs)
	if err != nil {
		return nil, err
	}
	_, finalPin, err := resolver.ReadResolved(ctx, operationPin)
	if err != nil {
		return nil, err
	}
	if finalPin.ValidUntil.Before(operationPin.ValidUntil) {
		operationPin.ValidUntil = finalPin.ValidUntil
	}
	if err := s.verifyExecutionProof(ctx, operationPin, in.ExecutionProof); err != nil {
		return nil, err
	}
	return result, nil
}

const studioExecutionBindingDomain = "studio.window.datasource.v1:"

func (s *Service) verifyExecutionProof(ctx context.Context, pin identity.ResolvedResource, proof *primitive.ExecutionProof) error {
	if s.windowExecutionProof == nil || proof == nil || !strings.HasPrefix(proof.Binding, studioExecutionBindingDomain) || proof.Token == "" {
		return identity.ErrResourceDenied
	}
	if err := s.windowExecutionProof.Verify(ctx, proof.Resource, types.WindowTarget{}, proof.Binding, proof.Token); err != nil {
		return err
	}
	if pin.ProviderIdentity != proof.Resource.ProviderIdentity || pin.URI != proof.Resource.URI || pin.ResourceCandidate != proof.Resource.ResourceCandidate || pin.AuthorityBinding != proof.Resource.AuthorityBinding || pin.ValidUntil.After(proof.Resource.ValidUntil) || !pin.ValidUntil.After(time.Now()) {
		return identity.ErrResourceDenied
	}
	return nil
}
func (s *Service) signExecutionProof(ctx context.Context, get *primitive.GetResult) error {
	if get == nil || get.ResolvedResource == nil || get.Resource == nil {
		return identity.ErrResourceDenied
	}
	variant, err := types.SelectWindowResource(get.Resource.DefinitionBytes, nil)
	if err != nil {
		return err
	}
	binding := studioExecutionBindingDomain + variant.Fingerprint
	token, err := s.windowExecutionProof.Sign(ctx, *get.ResolvedResource, types.WindowTarget{}, binding)
	if err != nil {
		return err
	}
	get.ExecutionProof = &primitive.ExecutionProof{Resource: *get.ResolvedResource, Binding: binding, Token: token}
	return nil
}

func (s *Service) registerWindowPrimitives(base *protoserver.DefaultHandler) error {
	if s.windowPrimitives == nil {
		return nil
	}
	for _, name := range []string{"namespaces/list", "namespaces/get", "windows/list", "windows/get", "windows/datasource"} {
		name := name
		var sample any
		switch name {
		case "namespaces/list":
			sample = &primitive.NamespaceListRequest{}
		case "namespaces/get":
			sample = &primitive.NamespaceGetRequest{}
		case "windows/list":
			sample = &primitive.ListRequest{}
		case "windows/get":
			sample = &primitive.GetRequest{}
		default:
			sample = &stockDatasourceInput{}
		}
		var input schema.ToolInputSchema
		if err := input.Load(sample); err != nil {
			return err
		}
		base.Registry.RegisterToolWithSchema(name, "Authorized configured Studio window operation", input, nil, func(ctx context.Context, request *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			var value any
			var err error
			switch name {
			case "namespaces/list":
				var in primitive.NamespaceListRequest
				err = decodePrimitiveArgs(request.Params.Arguments, &in)
				if err == nil {
					value, err = s.windowPrimitives.NamespaceList(ctx, in)
				}
			case "namespaces/get":
				var in primitive.NamespaceGetRequest
				err = decodePrimitiveArgs(request.Params.Arguments, &in)
				if err == nil {
					var capabilities *primitive.NamespaceCapabilities
					capabilities, err = s.windowPrimitives.NamespaceGet(ctx, in)
					if err == nil {
						for i := range capabilities.Kinds {
							if capabilities.Kinds[i].Kind == "window" {
								capabilities.Kinds[i].Operations = append(capabilities.Kinds[i].Operations, "fetch")
								capabilities.Kinds[i].Methods["fetch"] = "windows/datasource"
							}
						}
					}
					value = capabilities
				}
			case "windows/list":
				var in primitive.ListRequest
				err = decodePrimitiveArgs(request.Params.Arguments, &in)
				if err == nil {
					value, err = s.windowPrimitives.List(ctx, "window", in)
				}
			case "windows/get":
				var in primitive.GetRequest
				err = decodePrimitiveArgs(request.Params.Arguments, &in)
				if err == nil {
					var get *primitive.GetResult
					get, err = s.windowPrimitives.Get(ctx, "window", in)
					if err == nil {
						err = s.signExecutionProof(ctx, get)
					}
					value = get
				}
			default:
				var in stockDatasourceInput
				err = decodePrimitiveArgs(request.Params.Arguments, &in)
				if err == nil {
					value, err = s.primitiveFetch(ctx, in)
				}
			}
			failed := err != nil
			if failed {
				value = map[string]string{"code": "resource_unavailable", "message": "Studio window operation denied or unavailable"}
			}
			raw, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return nil, jsonrpc.NewInternalError("resource response unavailable", nil)
			}
			return &schema.CallToolResult{IsError: &failed, StructuredContent: value, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}}, nil
		})
		tool, _ := base.ToolRegistry.Get(name)
		tool.Metadata.Meta = map[string]any{primitive.AuthoringExtension: map[string]any{"version": 1, "providerIdentity": s.config.Forge.ProviderIdentity, "transport": "tools/call"}}
	}
	return nil
}
