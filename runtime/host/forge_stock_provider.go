package host

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/gating"
	"github.com/viant/forge/backend/mcp/portable"
	forgeservice "github.com/viant/forge/backend/mcp/service"
	"github.com/viant/forge/backend/reporting/identity"
	"github.com/viant/forge/backend/types"
	yaml "go.yaml.in/yaml/v3"
)

type stockWindowFile struct {
	ContractVersion    int                             `json:"contractVersion,omitempty" yaml:"contractVersion,omitempty"`
	DefinitionRevision string                          `json:"definitionRevision,omitempty" yaml:"definitionRevision,omitempty"`
	Resource           *identity.ResolvedResource      `json:"resource,omitempty" yaml:"resource,omitempty"`
	Window             *types.Window                   `json:"window" yaml:"window"`
	Report             any                             `json:"report,omitempty" yaml:"report,omitempty"`
	DataSources        map[string]*portable.DataSource `json:"dataSources" yaml:"dataSources"`
}

type stockWindowBundle struct {
	URI        string                       `json:"uri"`
	Source     []byte                       `json:"source"`
	Components map[string]stockComponentPin `json:"components"`
}

type stockComponentPin struct {
	Reference ComponentReference        `json:"reference"`
	Binding   portable.ComponentBinding `json:"binding"`
}

type stockWindowEntry struct {
	config ForgeWindow
	uri    identity.ResourceURI
	key    string
}

type stockWindowSource struct {
	service *Service
	root    string
	entries map[string]stockWindowEntry
	keys    map[string]string
}

type stockWindowPolicy struct {
	service    *Service
	selector   *authz.Selector
	principals gating.PrincipalResolver
	entries    map[string]stockWindowEntry
	tenant     string
}

func (s *Service) initStockForgeProvider(ctx context.Context) error {
	if s.config.Forge == nil {
		return nil
	}
	if s.config.ForgeProvider != nil {
		return fmt.Errorf("stock Forge provider conflicts with an injected provider")
	}
	if s.resourceAccess == nil || s.resourceAccess.Provider == nil {
		return fmt.Errorf("stock Forge provider requires verified resource identity")
	}
	principals, ok := s.resourceAccess.Provider.(gating.PrincipalResolver)
	if !ok || principals == nil {
		return fmt.Errorf("stock Forge provider requires a verified account principal resolver")
	}
	policies, err := authz.NewStaticStore(s.config.Forge.Policies)
	if err != nil {
		return fmt.Errorf("stock Forge policies: %w", err)
	}
	selections, err := authz.NewStaticSelectionStore(s.config.Forge.Selections)
	if err != nil {
		return fmt.Errorf("stock Forge selections: %w", err)
	}
	selector := &authz.Selector{Mappings: selections, Service: &authz.Service{Store: policies, Provider: s.resourceAccess.Provider}}
	root := s.config.RootDir
	if root == "" {
		root = "."
	}
	source := &stockWindowSource{service: s, root: root, entries: map[string]stockWindowEntry{}, keys: map[string]string{}}
	for _, configured := range s.config.Forge.Windows {
		uri, _ := identity.ParseResourceURI(configured.URI)
		key := uri.Namespace + "_" + uri.Name
		if _, exists := source.entries[configured.URI]; exists || source.keys[key] != "" {
			return fmt.Errorf("stock Forge window identity is ambiguous")
		}
		configured.Components = maps.Clone(configured.Components)
		entry := stockWindowEntry{config: configured, uri: uri, key: key}
		source.entries[configured.URI] = entry
		source.keys[key] = configured.URI
	}
	policy := &stockWindowPolicy{service: s, selector: selector, principals: principals, entries: source.entries, tenant: s.config.Access.Tenant}
	funcs := forgeservice.PortableHostFuncs{
		CatalogFunc: func(callCtx context.Context, in *portable.CatalogInput) (*portable.Catalog, error) {
			catalog := &portable.Catalog{ContractVersion: portable.Version, Windows: make([]portable.WindowSummary, 0, len(source.entries))}
			for _, entry := range source.entries {
				admitted := true
				for _, action := range []string{"discover", "describe"} {
					if _, err := policy.selectWindow(callCtx, entry, action); err != nil {
						if errors.Is(err, authz.ErrDenied) || errors.Is(err, authz.ErrIdentityDenied) {
							admitted = false
							break
						}
						return nil, err
					}
				}
				if !admitted {
					continue
				}
				title, err := source.title(callCtx, entry)
				if err != nil {
					return nil, err
				}
				catalog.Windows = append(catalog.Windows, portable.WindowSummary{ResourceURI: entry.config.URI, Namespace: entry.uri.Namespace,
					Name: entry.uri.Name, Key: entry.key, Title: title})
			}
			catalog.CatalogRevision = stockCatalogRevision(source.entries)
			return catalog, nil
		},
		ResourceRefFunc: func(_ context.Context, key string) (identity.ResourceRef, error) {
			uri, ok := source.keys[key]
			if !ok {
				return identity.ResourceRef{}, identity.ErrResourceDenied
			}
			return identity.ResourceRef{URI: uri}, nil
		},
		DefinitionResolvedFunc: source.definition,
		FetchResolvedFunc:      source.fetch,
	}
	authority := forgeservice.PortableAuthorityFuncs{
		AuthenticateFunc: func(ctx context.Context) (string, error) {
			principal, err := policy.principal(ctx)
			if err != nil {
				return "", err
			}
			return stockAuthorityBinding(principal), nil
		},
		AuthorizeFunc: func(ctx context.Context, binding, action, key, sourceID string) error {
			principal, err := policy.principal(ctx)
			if err != nil || stockAuthorityBinding(principal) != binding {
				return authz.ErrIdentityDenied
			}
			switch action {
			case "resource.discover":
				if key != "" || sourceID != "" {
					return authz.ErrDenied
				}
				return nil // Each window is filtered by its exact discover policy below.
			case "resource.describe", "resource.execute":
				entry, ok := source.entries[key]
				if !ok || (action == "resource.describe" && sourceID != "") || (action == "resource.execute" && !portableSourceID(sourceID)) {
					return authz.ErrDenied
				}
				policyAction := "describe"
				if action == "resource.execute" {
					policyAction = "execute"
				}
				_, err = policy.selectWindow(ctx, entry, policyAction)
				return err
			default:
				return authz.ErrDenied
			}
		},
		AuthorizeFetchFunc: func(ctx context.Context, binding string, input *portable.FetchInput) error {
			principal, err := policy.principal(ctx)
			if err != nil || stockAuthorityBinding(principal) != binding || input == nil || input.Resource == nil || input.Resource.ResourceCandidate.Kind != identity.WorkingCandidate || input.Resource.ResourceCandidate.Revision != "" {
				return authz.ErrIdentityDenied
			}
			entry, ok := source.entries[input.Resource.URI]
			if !ok || input.WindowKey != entry.key || !portableSourceID(input.DataSourceID) {
				return authz.ErrDenied
			}
			if _, exists := entry.config.Components[input.DataSourceID]; !exists {
				return authz.ErrDenied
			}
			_, err = policy.selectWindow(ctx, entry, "execute")
			return err
		},
	}
	s.config.ForgeProvider = &forgeservice.PortableProvider{
		Host: funcs, Authority: authority,
		ResourceResolver: func(context.Context) (*identity.ResourceResolver, error) {
			return &identity.ResourceResolver{Source: source, Policy: policy}, nil
		},
	}
	return nil
}

func (p *stockWindowPolicy) principal(ctx context.Context) (gating.Principal, error) {
	if p == nil || p.principals == nil || ctx == nil || ctx.Err() != nil {
		return gating.Principal{}, authz.ErrIdentityDenied
	}
	if p.service == nil || p.service.authorizeNamespace(ctx) != nil {
		return gating.Principal{}, authz.ErrIdentityDenied
	}
	principal, err := p.principals.ResolvePrincipal(ctx)
	if err != nil || principal.AccountID == "" || principal.IdentityRevision == "" || principal.Facts.Subject == "" || principal.Facts.Issuer == "" || principal.Facts.Tenant == "" || !principal.Facts.ValidUntil.After(time.Now()) {
		return gating.Principal{}, authz.ErrIdentityDenied
	}
	return principal, nil
}

func (p *stockWindowPolicy) SelectRevision(ctx context.Context, ref identity.ResourceRef, candidates []identity.ResourceCandidate) (identity.ResourceDecision, error) {
	if p == nil {
		return identity.ResourceDecision{}, authz.ErrDenied
	}
	uri, err := identity.ParseResourceURI(ref.URI)
	if err != nil || uri.Kind != "window" || ref.Revision != "" && ref.Revision != identity.WorkingCandidate {
		return identity.ResourceDecision{}, authz.ErrDenied
	}
	entry, ok := p.entries[ref.URI]
	if !ok {
		return identity.ResourceDecision{}, authz.ErrDenied
	}
	selected, err := p.selectWindow(ctx, entry, "discover")
	if err != nil || selected.Resource.Version != identity.WorkingCandidate {
		return identity.ResourceDecision{}, authz.ErrDenied
	}
	for _, candidate := range candidates {
		if candidate.Kind != identity.WorkingCandidate || candidate.Revision != "" {
			continue
		}
		principal, err := p.principal(ctx)
		if err != nil {
			return identity.ResourceDecision{}, err
		}
		return identity.ResourceDecision{Candidate: candidate, AuthorityBinding: stockAuthorityBinding(principal), ValidUntil: principal.Facts.ValidUntil}, nil
	}
	return identity.ResourceDecision{}, authz.ErrDenied
}

func (p *stockWindowPolicy) selectWindow(ctx context.Context, entry stockWindowEntry, action string) (authz.SelectedResource, error) {
	if p == nil || p.selector == nil || entry.config.URI == "" {
		return authz.SelectedResource{}, authz.ErrUnavailable
	}
	if p.service == nil || p.service.authorizeNamespace(ctx) != nil {
		return authz.SelectedResource{}, authz.ErrIdentityDenied
	}
	resource := authz.ResourceFamily{Kind: "window", ID: entry.uri.Namespace + "/" + entry.uri.Name, Tenant: p.tenant}
	selected, err := p.selector.Authorize(ctx, authz.SelectionRequest{Resource: resource, Action: action})
	if err != nil || selected.Decision.Bounded {
		return authz.SelectedResource{}, authz.ErrDenied
	}
	return selected, nil
}

func stockAuthorityBinding(principal gating.Principal) string {
	snapshot, err := json.Marshal(struct {
		Subject, Issuer, Tenant, AccountID, IdentityRevision string
	}{principal.Facts.Subject, principal.Facts.Issuer, principal.Facts.Tenant, principal.AccountID, principal.IdentityRevision})
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(snapshot)
	return hex.EncodeToString(hash[:])
}

func stockCatalogRevision(entries map[string]stockWindowEntry) string {
	uris := make([]string, 0, len(entries))
	for uri := range entries {
		uris = append(uris, uri)
	}
	// encoding/json sorts string map keys but a slice needs canonical order.
	sort.Strings(uris)
	raw, _ := json.Marshal(uris)
	return identity.ContentFingerprint(raw)
}

func (s *stockWindowSource) Candidates(ctx context.Context, uri identity.ResourceURI) ([]identity.ResourceCandidate, error) {
	bundle, _, err := s.bundle(ctx, uri)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	return []identity.ResourceCandidate{{Kind: identity.WorkingCandidate, ContentFingerprint: identity.ContentFingerprint(raw)}}, nil
}

func (s *stockWindowSource) ReadCandidate(ctx context.Context, uri identity.ResourceURI, candidate identity.ResourceCandidate) (json.RawMessage, error) {
	if candidate.Kind != identity.WorkingCandidate || candidate.Revision != "" {
		return nil, identity.ErrResourceDenied
	}
	bundle, _, err := s.bundle(ctx, uri)
	if err != nil {
		return nil, err
	}
	return json.Marshal(bundle)
}

func (s *stockWindowSource) bundle(ctx context.Context, uri identity.ResourceURI) (stockWindowBundle, stockWindowEntry, error) {
	entry, ok := s.entries[uri.String()]
	if !ok || entry.uri != uri {
		return stockWindowBundle{}, stockWindowEntry{}, identity.ErrResourceDenied
	}
	path, err := s.definitionPath(entry.config.DefinitionPath)
	if err != nil {
		return stockWindowBundle{}, stockWindowEntry{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || len(raw) > 8<<20 {
		return stockWindowBundle{}, stockWindowEntry{}, fmt.Errorf("stock Forge window source is unavailable")
	}
	definition, err := decodeStockWindowFile(raw)
	if err != nil || definition.Report != nil || definition.Resource != nil || definition.ContractVersion != 0 || definition.DefinitionRevision != "" || definition.Window == nil {
		return stockWindowBundle{}, stockWindowEntry{}, fmt.Errorf("stock Forge source must be an unversioned window definition")
	}
	if err := validateStockWindow(entry, definition); err != nil {
		return stockWindowBundle{}, stockWindowEntry{}, err
	}
	pins := make(map[string]stockComponentPin, len(entry.config.Components))
	for datasourceID, ref := range entry.config.Components {
		binding, err := s.service.ResolveComponentBinding(ctx, ref)
		if err != nil {
			if errors.Is(err, authz.ErrDenied) || errors.Is(err, authz.ErrIdentityDenied) {
				return stockWindowBundle{}, stockWindowEntry{}, identity.ErrResourceDenied
			}
			return stockWindowBundle{}, stockWindowEntry{}, err
		}
		if binding.ID != ref.ID || binding.Revision != ref.Revision || binding.Kind != ref.Kind || binding.SchemaFingerprint == "" || binding.ContentFingerprint == "" {
			return stockWindowBundle{}, stockWindowEntry{}, fmt.Errorf("stock Forge component binding is incomplete")
		}
		pins[datasourceID] = stockComponentPin{Reference: ref, Binding: binding}
	}
	return stockWindowBundle{URI: uri.String(), Source: append([]byte(nil), raw...), Components: pins}, entry, nil
}

func (s *stockWindowSource) definitionPath(relative string) (string, error) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, relative))
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(root, path)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) || filepath.IsAbs(inside) {
		return "", fmt.Errorf("stock Forge definition path escapes RootDir")
	}
	return path, nil
}

func decodeStockWindowFile(raw []byte) (*stockWindowFile, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if err := validateStockWindowFileRoot(&document); err != nil {
		return nil, err
	}
	var definition stockWindowFile
	if err := document.Decode(&definition); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("stock Forge definition must contain one document")
	}
	return &definition, nil
}

func validateStockWindowFileRoot(document *yaml.Node) error {
	if document == nil || document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("stock Forge definition must be one mapping")
	}
	allowed := map[string]bool{"contractVersion": true, "definitionRevision": true, "resource": true, "window": true, "report": true, "dataSources": true}
	root := document.Content[0]
	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		if key.Kind != yaml.ScalarNode || !allowed[key.Value] || seen[key.Value] {
			return fmt.Errorf("stock Forge definition contains an unsupported or duplicate root field")
		}
		seen[key.Value] = true
	}
	return nil
}

func validateStockWindow(entry stockWindowEntry, definition *stockWindowFile) error {
	window := definition.Window
	if window.Resource != nil || window.ResourceDependencies != nil || window.WindowKey != "" && window.WindowKey != entry.key || window.Namespace != "" && window.Namespace != entry.uri.Namespace || window.View.Content == nil {
		return fmt.Errorf("stock Forge window identity or content is invalid")
	}
	if len(definition.DataSources) == 0 || len(definition.DataSources) != len(entry.config.Components) {
		return fmt.Errorf("stock Forge datasource components must match definition sources (%d descriptors, %d mappings)", len(definition.DataSources), len(entry.config.Components))
	}
	for id, source := range definition.DataSources {
		if !portableSourceID(id) || source == nil || source.ID != id || source.Backend != nil || source.Service != nil {
			return fmt.Errorf("stock Forge source %q contains untrusted backend metadata", id)
		}
		if _, ok := entry.config.Components[id]; !ok {
			return fmt.Errorf("stock Forge source %q has no component mapping", id)
		}
		if _, collision := window.DataSource[id]; collision {
			return fmt.Errorf("stock Forge source %q collides with a window-local datasource", id)
		}
	}
	if err := types.ValidateResourceModels(window); err != nil {
		return fmt.Errorf("stock Forge window contract: %w", err)
	}
	return nil
}

func (s *stockWindowSource) title(ctx context.Context, entry stockWindowEntry) (string, error) {
	path, err := s.definitionPath(entry.config.DefinitionPath)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || len(raw) > 8<<20 {
		return "", fmt.Errorf("stock Forge window source is unavailable")
	}
	definition, err := decodeStockWindowFile(raw)
	if err != nil {
		return "", fmt.Errorf("decode stock Forge window source: %w", err)
	}
	if definition.Window == nil {
		return "", fmt.Errorf("stock Forge window source has no window")
	}
	if err := validateStockWindow(entry, definition); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if title := strings.TrimSpace(definition.Window.View.Content.Title); title != "" {
		return title, nil
	}
	if chip := strings.TrimSpace(definition.Window.ChipName); chip != "" {
		return chip, nil
	}
	return entry.uri.Name, nil
}

func (s *stockWindowSource) definition(ctx context.Context, pin identity.ResolvedResource, raw json.RawMessage) (*portable.Definition, error) {
	var bundle stockWindowBundle
	if err := json.Unmarshal(raw, &bundle); err != nil || bundle.URI != pin.URI {
		return nil, identity.ErrResourceDenied
	}
	entry, ok := s.entries[pin.URI]
	if !ok {
		return nil, identity.ErrResourceDenied
	}
	file, err := decodeStockWindowFile(bundle.Source)
	if err != nil || file.Report != nil || validateStockWindow(entry, file) != nil {
		return nil, identity.ErrResourceDenied
	}
	uri := entry.uri
	window := *file.Window
	window.WindowKey = entry.key
	window.Namespace = uri.Namespace
	window.Resource = &pin
	dataSources := make(map[string]*portable.DataSource, len(file.DataSources))
	for id, descriptor := range file.DataSources {
		resolved, exists := bundle.Components[id]
		if !exists || resolved.Reference != entry.config.Components[id] || resolved.Binding.ID != resolved.Reference.ID || resolved.Binding.Revision != resolved.Reference.Revision || resolved.Binding.SchemaFingerprint == "" || resolved.Binding.ContentFingerprint == "" {
			return nil, identity.ErrResourceDenied
		}
		backend := &portable.Backend{Ownership: "provider", Kind: "datly", Method: portable.FetchTool,
			Pinned:            map[string]any{"windowKey": entry.key, "dataSourceId": id, "definitionRevision": pin.ContentFingerprint},
			SchemaFingerprint: resolved.Binding.SchemaFingerprint, Component: &resolved.Binding}
		copy := *descriptor
		copy.Backend = backend
		dataSources[id] = &copy
	}
	result := &portable.Definition{Resource: &pin, ContractVersion: portable.Version, DefinitionRevision: pin.ContentFingerprint, Window: &window, DataSources: dataSources}
	if err := portable.ValidateResourceBindings(result, nil); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *stockWindowSource) fetch(ctx context.Context, input *portable.FetchInput, source *portable.DataSource) (portable.FetchOutput, error) {
	if input == nil || input.Resource == nil || source == nil || source.ID != input.DataSourceID || source.Backend == nil || source.Backend.Kind != "datly" || source.Backend.Ownership != "provider" || source.Backend.Method != portable.FetchTool || source.Backend.Component == nil {
		return nil, identity.ErrResourceDenied
	}
	entry, ok := s.entries[input.Resource.URI]
	if !ok || entry.key != input.WindowKey {
		return nil, identity.ErrResourceDenied
	}
	configured, ok := entry.config.Components[input.DataSourceID]
	if !ok || validateComponentReference(configured) != nil {
		return nil, identity.ErrResourceDenied
	}
	return s.service.ExecuteComponentJSON(ctx, configured, *source.Backend.Component, input.Inputs)
}
