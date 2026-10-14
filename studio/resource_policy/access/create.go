package access

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
	sharedapi "github.com/viant/authz/component/api"
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
func creationAdministration(service *authz.Service) *authz.Administration {
	roles := strings.FieldsFunc(os.Getenv("STUDIO_ACCESS_POLICY_EDITOR_ROLES"), func(r rune) bool { return r == ',' || r == ' ' })
	return &authz.Administration{Store: service.Store, Provider: service.Provider, EditorRoles: roles}
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
	facts, err := service.Provider.Resolve(requestCtx)
	if err != nil || input.Document.Resource.Tenant != facts.Tenant {
		return publicError(403, "policy creation requires the verified tenant")
	}
	services := &sharedapi.Services{Authorization: service, Policies: creationAdministration(service)}
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
	allowed, err := creationAdministration(service).CanReplace(ctx, resource)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, publicError(403, "initial policy creation is not permitted")
	}
	facts, err := service.Provider.Resolve(ctx)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
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
	document := &authz.Document{Resource: resource, Revision: 0, Policies: map[string]authz.Policy{
		"viewAccess":   {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: facts.Subject}},
		"manageAccess": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: facts.Subject}},
	}}
	return document, &authz.EditorContext{Source: "verified-principal", CanManage: true, Choices: choices}, nil
}
