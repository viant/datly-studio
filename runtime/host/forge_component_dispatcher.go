package host

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/viant/forge/backend/mcp/portable"
	"github.com/viant/forge/backend/reporting/identity"
)

// ComponentDispatchMapping binds a portable producer service and logical
// method to one exact native Datly component route. Mappings are deployment
// configuration; requests can select only among entries matching the
// already-approved component pin.
type ComponentDispatchMapping struct {
	Service   string
	Method    string
	Component ComponentReference
}

// ForgeComponentDispatcher adapts the native Datly component resolver and
// executor to Forge's portable exact-revision dispatch contract.
type ForgeComponentDispatcher struct {
	runtime   *Service
	mappings  map[componentDispatchKey][]ComponentReference
	producers map[string]struct{}
}

type componentDispatchKey struct {
	service string
	method  string
}

// NewForgeComponentDispatcher snapshots and validates all deployment-owned
// mappings. The same service/method may have multiple exact revisions, but a
// duplicate component revision is rejected as ambiguous.
func NewForgeComponentDispatcher(runtime *Service, mappings []ComponentDispatchMapping) (*ForgeComponentDispatcher, error) {
	if runtime == nil || len(mappings) == 0 {
		return nil, fmt.Errorf("component dispatcher requires a runtime and mappings")
	}
	result := &ForgeComponentDispatcher{
		runtime:   runtime,
		mappings:  make(map[componentDispatchKey][]ComponentReference, len(mappings)),
		producers: make(map[string]struct{}),
	}
	seen := map[componentDispatchKey]map[string]struct{}{}
	for _, mapping := range mappings {
		if mapping.Service == "" || strings.TrimSpace(mapping.Service) != mapping.Service || mapping.Method == "" || strings.TrimSpace(mapping.Method) != mapping.Method {
			return nil, fmt.Errorf("component dispatcher service and method must be canonical")
		}
		if err := validateComponentReference(mapping.Component); err != nil {
			return nil, fmt.Errorf("component dispatcher mapping %q/%q: %w", mapping.Service, mapping.Method, err)
		}
		key := componentDispatchKey{service: mapping.Service, method: mapping.Method}
		if seen[key] == nil {
			seen[key] = map[string]struct{}{}
		}
		identity := mapping.Component.Kind + "\x00" + mapping.Component.ID + "\x00" + mapping.Component.Revision
		if _, ok := seen[key][identity]; ok {
			return nil, fmt.Errorf("component dispatcher mapping %q/%q has a duplicate exact component", mapping.Service, mapping.Method)
		}
		seen[key][identity] = struct{}{}
		result.mappings[key] = append(result.mappings[key], mapping.Component)
		result.producers[mapping.Service] = struct{}{}
	}
	return result, nil
}

var _ portable.ComponentDispatcher = (*ForgeComponentDispatcher)(nil)

// IsComponentProducer reports only deployment-configured services. It lets a
// consumer reject unpinned generic datasource routing for this producer.
func (d *ForgeComponentDispatcher) IsComponentProducer(service string) bool {
	if d == nil || service == "" || strings.TrimSpace(service) != service {
		return false
	}
	_, ok := d.producers[service]
	return ok
}

func (d *ForgeComponentDispatcher) ObserveComponent(ctx context.Context, service, method string, expected portable.ComponentBinding) (portable.ComponentBinding, error) {
	ref, err := d.exactReference(service, method, expected)
	if err != nil {
		return portable.ComponentBinding{}, err
	}
	observed, err := d.runtime.ResolveComponentBinding(ctx, ref)
	if err != nil {
		return portable.ComponentBinding{}, err
	}
	if err := validateCompleteComponentBinding(expected); err != nil {
		return portable.ComponentBinding{}, err
	}
	if err := portable.ValidateComponentDispatch(&expected, observed); err != nil {
		return portable.ComponentBinding{}, err
	}
	return observed, nil
}

func (d *ForgeComponentDispatcher) ExecuteComponent(ctx context.Context, service, method string, expected portable.ComponentBinding, args map[string]interface{}) (json.RawMessage, error) {
	ref, err := d.exactReference(service, method, expected)
	if err != nil {
		return nil, err
	}
	if err := validateCompleteComponentBinding(expected); err != nil {
		return nil, err
	}
	return d.runtime.ExecuteComponentJSON(ctx, ref, expected, args)
}

func (d *ForgeComponentDispatcher) exactReference(service, method string, expected portable.ComponentBinding) (ComponentReference, error) {
	if d == nil || d.runtime == nil || service == "" || method == "" || expected.ID == "" || expected.Revision == "" || expected.Kind == "" {
		return ComponentReference{}, fmt.Errorf("component dispatch pin is incomplete")
	}
	if strings.TrimSpace(service) != service || strings.TrimSpace(method) != method {
		return ComponentReference{}, fmt.Errorf("component dispatch service and method must be canonical")
	}
	for _, ref := range d.mappings[componentDispatchKey{service: service, method: method}] {
		if ref.Kind == expected.Kind && ref.ID == expected.ID && ref.Revision == expected.Revision {
			return ref, nil
		}
	}
	return ComponentReference{}, fmt.Errorf("component dispatch mapping is unavailable for the exact pin")
}

func validateCompleteComponentBinding(binding portable.ComponentBinding) error {
	if binding.Kind != "dynamic" && binding.Kind != "linked" || binding.ID == "" || binding.Revision == "" || binding.Revision == "active" || binding.Revision == "latest" || binding.Revision == "working" {
		return fmt.Errorf("component dispatch requires an exact component pin")
	}
	if !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: binding.ContentFingerprint}).Valid() || !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: binding.SchemaFingerprint}).Valid() {
		return fmt.Errorf("component dispatch requires complete content and schema fingerprints")
	}
	return nil
}
