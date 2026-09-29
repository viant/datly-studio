package preview

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/sdk"
	access "github.com/viant/authz"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
)

// previewContexts binds only this exact version's own server context. Facts
// come from the configured provider and must match the verified Studio caller.
func (d Dynamic) previewContexts(ctx context.Context, definition *definition, component *spec.Component) ([]*registry.RegisteredComponent, error) {
	dependencies, err := accesscontext.DependsOn(component)
	if err != nil {
		return nil, err
	}
	if len(dependencies) == 0 {
		return nil, nil
	}
	principal, verified := sdk.PrincipalFromContext(ctx)
	credential, authenticated := sdk.VerifiedCredentialFromContext(ctx)
	denied := func() error {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "scoped preview requires a verified caller and configured authorization provider"}
	}
	if d.Access == nil || !verified || !authenticated || credential.Bearer == "" || d.AccessTenant == "" {
		return nil, denied()
	}
	resource := access.Resource{Kind: "component", ID: definition.ReportID, Version: strconv.Itoa(definition.VersionNo), Tenant: d.AccessTenant}
	decide := func(callCtx context.Context) (access.Facts, access.Decision, error) {
		decision, facts, err := d.Access.AuthorizeWithFacts(callCtx, access.Request{Resource: resource, Action: "execute"})
		if err != nil || facts.Subject != principal.Subject || facts.Tenant != resource.Tenant || !facts.ValidUntil.After(time.Now()) || callCtx.Err() != nil {
			return access.Facts{}, access.Decision{}, denied()
		}
		return facts, decision, nil
	}
	if _, _, err = decide(ctx); err != nil {
		return nil, err
	}
	var result []*registry.RegisteredComponent
	for _, dependency := range dependencies {
		if dependency.ComponentID != definition.ReportID {
			return nil, denied()
		}
		registered, err := accesscontext.Register(dependency, decide)
		if err != nil {
			return nil, err
		}
		result = append(result, registered)
	}
	return result, nil
}

func rejectClientContextValues(component *spec.Component, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "preview input must be an object"}
	}
	for _, parameter := range spec.EffectiveParameters(component.Parameters) {
		if parameter == nil || (!strings.EqualFold(parameter.Source.Kind, "component") && !strings.EqualFold(parameter.Source.Kind, "param")) {
			continue
		}
		for key := range fields {
			if strings.EqualFold(key, parameter.Name) {
				return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "server-owned preview inputs cannot be supplied by the client"}
			}
		}
	}
	return nil
}
