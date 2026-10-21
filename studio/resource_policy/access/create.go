package access

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
	sharedapi "github.com/viant/authz/component/api"
	"github.com/viant/datly-studio/runtime/accessprovider"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"os"
	"reflect"
	"strings"
)

type PolicyCreateComponent struct {
	Contract xdatly.Component[PolicyWriteInput, sharedapi.PolicyOutput] `component:"policy_create,path=/v1/authz/sdk/policies.create,method=POST,connector=studio,handler=NewPolicyCreate" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.policies.create\"}]" caseFormat:"lc"`
}

var PolicyCreateDatly = new(PolicyCreateComponent)
var createReachability = reflect.TypeFor[PolicyCreateComponent]()

func (PolicyCreateComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPolicyCreate" {
		return custom.Factory(NewPolicyCreate)
	}
	return nil
}

type policyCreateHandler struct{}

func NewPolicyCreate() xhandler.Contract[PolicyWriteInput, sharedapi.PolicyOutput] {
	return &policyCreateHandler{}
}

// Initial policy creation is opt-in through deployment-owned administrator roles.
// Existing policy replacement continues to require its manageAccess policy.
func creationAdministration(ctx context.Context, service *authz.Service) *authz.Administration {
	roles := strings.FieldsFunc(os.Getenv("STUDIO_ACCESS_POLICY_EDITOR_ROLES"), func(r rune) bool { return r == ',' || r == ' ' })
	admin := &authz.Administration{Store: service.Store, Provider: service.Provider, EditorRoles: roles, EditorCheck: accessprovider.PolicyEditor(ctx)}
	fallback := *admin
	admin.EditorCheck = func(ctx context.Context, facts authz.Facts, resource authz.Resource) (bool, error) {
		if resource.Kind == "namespace" {
			return canManageNamespacePolicy(ctx, service, facts, resource)
		}
		return fallback.CanReplace(ctx, resource)
	}
	return admin
}
func (*policyCreateHandler) Exec(ctx context.Context, session xhandler.Session, input *PolicyWriteInput, output *sharedapi.PolicyOutput) error {
	if input == nil || output == nil {
		return publicError(400, "policy document is required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if err = checkResourceNamespace(requestCtx, service, input.NamespaceId, input.Document.Resource, input.Auth, input.Jwt); err != nil {
		return err
	}
	if err = validateNamespacePolicy(input.Document); err != nil {
		return err
	}
	facts, err := service.Provider.Resolve(requestCtx)
	if err != nil || input.Document.Resource.Tenant != facts.Tenant {
		return publicError(403, "policy creation requires the verified tenant")
	}
	services := &sharedapi.Services{Authorization: service, Policies: creationAdministration(requestCtx, service)}
	bound := authoritySession{Session: session, binder: authorityBinder{Binder: session.Binder(), services: services}}
	return sharedapi.NewCreate().Exec(requestCtx, bound, &sharedapi.WriteInput{JWT: input.Jwt, Document: input.Document}, output)
}

func initialPolicy(ctx context.Context, service *authz.Service, resource authz.Resource) (*authz.Document, *authz.EditorContext, error) {
	_, err := service.Store.Get(ctx, resource)
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if resource.Tenant == "*" {
		return nil, nil, publicError(403, "initial policy requires a concrete tenant")
	}
	allowed, err := creationAdministration(ctx, service).CanReplace(ctx, resource)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, publicError(403, "initial policy creation is not permitted")
	}
	editor, err := verifiedEditorContext(ctx, service)
	if err != nil {
		return nil, nil, err
	}
	facts, err := service.Provider.Resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	document := &authz.Document{Resource: resource, Revision: 0, Policies: map[string]authz.Policy{
		"viewAccess":   {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: facts.Subject}},
		"manageAccess": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: facts.Subject}},
	}}
	if resource.Kind == "namespace" {
		document.Policies = map[string]authz.Policy{"discover": {Mode: "deny"}, "describe": {Mode: "deny"}, "execute": {Mode: "deny"}}
	}
	return document, editor, nil
}

func verifiedEditorContext(ctx context.Context, service *authz.Service) (*authz.EditorContext, error) {
	facts, err := service.Provider.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	choices := authz.Choices{Subjects: []authz.Choice{{ID: facts.Subject, Label: facts.Subject}}}
	for _, role := range facts.Roles {
		choices.Roles = append(choices.Roles, authz.Choice{ID: role, Label: role})
	}
	for _, exposure := range facts.Exposures {
		choices.Exposures = append(choices.Exposures, authz.Choice{ID: exposure, Label: exposure})
	}
	entities, err := facts.FlatEntities()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, entity := range entities {
		value := entity
		choices.Entities = append(choices.Entities, authz.Choice{Entity: &value, Label: entity.Type + ": " + entity.ID})
		if !seen[entity.Type] {
			choices.EntityTypes = append(choices.EntityTypes, entity.Type)
			seen[entity.Type] = true
		}
	}

	return &authz.EditorContext{Source: "verified-principal", CanManage: true, Choices: choices}, nil
}
