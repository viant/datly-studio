package host

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/viant/datly-studio/sdk/access"
	dexec "github.com/viant/datly/exec"
	handlerprovider "github.com/viant/datly/runtime/handler/provider"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
)

// scopeParameterKind is the DQL parameter source kind reserved for
// deployment-bound authorization scope. No transport provider serves it, so a
// client cannot populate or override a scope parameter.
const scopeParameterKind = "scope"

// resolvedScopeBinding is a deployment declaration validated against the
// compiled component it names. It owns the exact compiled Go type of the input
// field so authorized IDs are converted, never coerced by the transport layer.
type resolvedScopeBinding struct {
	declaration ScopeBinding
	sliceType   reflect.Type
}

func isScopeParameter(parameter *spec.Parameter) bool {
	return parameter != nil && !parameter.EmitOutput && strings.EqualFold(strings.TrimSpace(parameter.Source.Kind), scopeParameterKind)
}

// compiledComponent is the slice of a registration that scope resolution needs:
// the authored parameters and the compiled Go input type they bind into.
type compiledComponent struct {
	key       spec.Key
	component *spec.Component
	inputType reflect.Type
}

func compiledComponents(registrations []*registry.RegisteredComponent) []compiledComponent {
	result := make([]compiledComponent, 0, len(registrations))
	for _, registered := range registrations {
		if registered == nil || registered.Component == nil {
			continue
		}
		item := compiledComponent{key: registered.Component.Key, component: registered.Component}
		if registered.Input != nil {
			item.inputType = registered.Input.Type()
		}
		result = append(result, item)
	}
	return result
}

// resolveScopeBindings pairs every deployment declaration with the compiled
// component of the active version and rejects any mismatch. A component that
// declares a scope parameter without a declaration, or a declaration without a
// matching parameter, fails the whole generation so the runtime never serves a
// component whose authorization scope is undefined.
func resolveScopeBindings(declarations []ScopeBinding, accessEnabled bool, components []compiledComponent, reportByComponent map[spec.Key]string, versionByReport map[string]int) (map[string]*resolvedScopeBinding, error) {
	result := map[string]*resolvedScopeBinding{}
	for _, compiled := range components {
		if compiled.component == nil {
			continue
		}
		reportID := reportByComponent[compiled.key]
		if reportID == "" {
			return nil, fmt.Errorf("component %s has no owning report", compiled.key.String())
		}
		version := strconv.Itoa(versionByReport[reportID])
		var matching []ScopeBinding
		for _, declaration := range declarations {
			if declaration.Component == reportID && declaration.Version == version {
				matching = append(matching, declaration)
			}
		}
		resolved, err := resolveScopeBinding(matching, accessEnabled, reportID, version, compiled.component, compiled.inputType)
		if err != nil {
			return nil, err
		}
		if resolved == nil {
			continue
		}
		if existing := result[reportID]; existing != nil && existing.declaration != resolved.declaration {
			return nil, fmt.Errorf("report %s version %s resolves conflicting scope bindings", reportID, version)
		}
		result[reportID] = resolved
	}
	return result, nil
}

// resolveScopeBinding validates one component against its declarations. It
// returns nil when the component has no scope parameter and no declaration.
func resolveScopeBinding(declarations []ScopeBinding, accessEnabled bool, reportID, version string, component *spec.Component, inputType reflect.Type) (*resolvedScopeBinding, error) {
	var scoped []*spec.Parameter
	for _, parameter := range spec.EffectiveParameters(component.Parameters) {
		if isScopeParameter(parameter) {
			scoped = append(scoped, parameter)
		}
	}
	switch {
	case len(scoped) == 0 && len(declarations) == 0:
		return nil, nil
	case len(scoped) == 0:
		return nil, fmt.Errorf("report %s version %s declares a scope binding but its component has no %s parameter", reportID, version, scopeParameterKind)
	case !accessEnabled:
		return nil, fmt.Errorf("report %s version %s declares a %s parameter, which requires generic resource access configuration", reportID, version, scopeParameterKind)
	case len(scoped) > 1:
		return nil, fmt.Errorf("report %s version %s declares %d %s parameters; one entity dimension per component is supported", reportID, version, len(scoped), scopeParameterKind)
	case len(declarations) == 0:
		return nil, fmt.Errorf("report %s version %s declares %s parameter %q without a deployment scope binding", reportID, version, scopeParameterKind, scoped[0].Name)
	case len(declarations) > 1:
		return nil, fmt.Errorf("report %s version %s has %d scope bindings", reportID, version, len(declarations))
	}
	parameter, declaration := scoped[0], declarations[0]
	if parameter.Name != declaration.Parameter {
		return nil, fmt.Errorf("report %s version %s scope binding names parameter %q but the component declares %q", reportID, version, declaration.Parameter, parameter.Name)
	}
	if strings.TrimSpace(parameter.Source.Name) != declaration.EntityType {
		return nil, fmt.Errorf("report %s version %s scope binding entity type %q does not match parameter %s/%s", reportID, version, declaration.EntityType, scopeParameterKind, parameter.Source.Name)
	}
	if parameter.Required == nil || !*parameter.Required {
		return nil, fmt.Errorf("report %s version %s scope parameter %q must be declared Required()", reportID, version, parameter.Name)
	}
	if inputType == nil || inputType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("report %s version %s has no compiled input type for scope parameter %q", reportID, version, parameter.Name)
	}
	field, ok := typecatalog.FieldByName(inputType, parameter.Name)
	if !ok {
		return nil, fmt.Errorf("report %s version %s compiled input has no field for scope parameter %q", reportID, version, parameter.Name)
	}
	if field.Type.Kind() != reflect.Slice || !scopeElementSupported(field.Type.Elem()) {
		return nil, fmt.Errorf("report %s version %s scope parameter %q must be a slice of string or integer, got %s", reportID, version, parameter.Name, field.Type)
	}
	return &resolvedScopeBinding{declaration: declaration, sliceType: field.Type}, nil
}

func scopeElementSupported(element reflect.Type) bool {
	switch element.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// values converts an entity-bounded decision into the compiled slice type. Every
// entity must belong to the bound dimension, and integer IDs must be canonical
// decimals so that "007" can never alias "7" in the compiled query.
func (b *resolvedScopeBinding) values(decision access.Decision) (any, error) {
	if b == nil {
		return nil, errors.New("scope binding is required")
	}
	if !decision.Bounded || len(decision.Entities) == 0 {
		return nil, errors.New("scope binding requires an entity-bounded decision")
	}
	element := b.sliceType.Elem()
	result := reflect.MakeSlice(b.sliceType, 0, len(decision.Entities))
	seen := map[string]bool{}
	for _, entity := range decision.Entities {
		if entity.Type != b.declaration.EntityType {
			return nil, fmt.Errorf("decision entity type %q is incompatible with bound dimension %q", entity.Type, b.declaration.EntityType)
		}
		if entity.ID == "" || seen[entity.ID] {
			return nil, errors.New("decision contains an empty or duplicate entity ID")
		}
		seen[entity.ID] = true
		value := reflect.New(element).Elem()
		switch element.Kind() {
		case reflect.String:
			value.SetString(entity.ID)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parsed, err := strconv.ParseInt(entity.ID, 10, element.Bits())
			if err != nil || strconv.FormatInt(parsed, 10) != entity.ID {
				return nil, fmt.Errorf("entity ID is not a canonical %s", element)
			}
			value.SetInt(parsed)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			parsed, err := strconv.ParseUint(entity.ID, 10, element.Bits())
			if err != nil || strconv.FormatUint(parsed, 10) != entity.ID {
				return nil, fmt.Errorf("entity ID is not a canonical %s", element)
			}
			value.SetUint(parsed)
		default:
			return nil, fmt.Errorf("unsupported scope element type %s", element)
		}
		result = reflect.Append(result, value)
	}
	return result.Interface(), nil
}

// bind converts the decision and attaches a scope provider to the invocation
// captured in ctx. The provider serves only the bound entity dimension and only
// the compiled type; any other request fails instead of silently widening.
func (b *resolvedScopeBinding) bind(ctx context.Context, decision access.Decision) error {
	value, err := b.values(decision)
	if err != nil {
		return err
	}
	valueType := reflect.TypeOf(value)
	entityType := b.declaration.EntityType
	return dexec.BindScope(ctx, handlerprovider.Named(scopeParameterKind, func(_ context.Context, target reflect.Type, name string) (any, bool, error) {
		if name != entityType {
			return nil, false, fmt.Errorf("scope dimension %q is not bound for this component", name)
		}
		if target != nil && !valueType.AssignableTo(target) {
			return nil, false, fmt.Errorf("scope dimension %q is bound as %s, component expects %s", name, valueType, target)
		}
		return value, true, nil
	}))
}
