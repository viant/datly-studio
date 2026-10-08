package access

import (
	"context"
	"reflect"

	"github.com/viant/authz"
	sharedapi "github.com/viant/authz/component/api"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

// Studio supplies verified authority and workspace isolation to the shared SDK.
// The published handlers own policy reads, management checks, CAS and errors.
type authoritySession struct {
	xhandler.Session
	binder xhandler.Binder
}

func (s authoritySession) Binder() xhandler.Binder { return s.binder }

type authorityBinder struct {
	xhandler.Binder
	services *sharedapi.Services
}

func (b authorityBinder) Lookup(ctx context.Context, key xhandler.ValueKey) (any, bool, error) {
	if key == sharedapi.Capability {
		return b.services, true, nil
	}
	return b.Binder.Lookup(ctx, key)
}
func sharedSession(session xhandler.Session, service *authz.Service) xhandler.Session {
	services := &sharedapi.Services{Authorization: service, Policies: &authz.Administration{Store: service.Store, Provider: service.Provider, Management: service}}
	return authoritySession{Session: session, binder: authorityBinder{Binder: session.Binder(), services: services}}
}

type PolicyInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Resource    authz.Resource     `parameter:"Resource,kind=body,in=resource,dataType=authz.Resource,required=true" json:"resource"`
}
type PolicyWriteInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Document    authz.Document     `parameter:"Document,kind=body,in=document,dataType=authz.Document,required=true" json:"document"`
}
type AuthorizationInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Resource    authz.Resource     `parameter:"Resource,kind=body,in=resource,dataType=authz.Resource,required=true" json:"resource"`
	Action      string             `parameter:"Action,kind=body,in=action,dataType=string,required=true" json:"action"`
	Selection   *[]authz.Entity    `parameter:"Selection,kind=body,in=selection,dataType=*[]authz.Entity,required=false" json:"selection,omitempty"`
}

type PolicyGetComponent struct {
	Contract xdatly.Component[PolicyInput, sharedapi.PolicyOutput] `component:"policy_get,path=/v1/authz/sdk/policies.get,method=POST,connector=studio,handler=NewPolicyGet" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.policies.get\"}]" caseFormat:"lc"`
}
type PolicyContextComponent struct {
	Contract xdatly.Component[PolicyInput, sharedapi.PolicyContextOutput] `component:"policy_context,path=/v1/authz/sdk/policies.context,method=POST,connector=studio,handler=NewPolicyContext" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.policies.context\"}]" caseFormat:"lc"`
}
type PolicyReplaceComponent struct {
	Contract xdatly.Component[PolicyWriteInput, sharedapi.PolicyOutput] `component:"policy_replace,path=/v1/authz/sdk/policies.replace,method=POST,connector=studio,handler=NewPolicyReplace" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.policies.replace\"}]" caseFormat:"lc"`
}
type AuthorizationComponent struct {
	Contract xdatly.Component[AuthorizationInput, sharedapi.DecisionOutput] `component:"authorization_check,path=/v1/authz/sdk/authorization.check,method=POST,connector=studio,handler=NewAuthorizationCheck" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.authorization.check\"}]" caseFormat:"lc"`
}

var (
	PolicyGetDatly              = new(PolicyGetComponent)
	PolicyContextDatly          = new(PolicyContextComponent)
	PolicyReplaceDatly          = new(PolicyReplaceComponent)
	AuthorizationDatly          = new(AuthorizationComponent)
	sharedComponentReachability = []reflect.Type{reflect.TypeFor[PolicyGetComponent](), reflect.TypeFor[PolicyContextComponent](), reflect.TypeFor[PolicyReplaceComponent](), reflect.TypeFor[AuthorizationComponent]()}
)

func (PolicyGetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPolicyGet" {
		return custom.Factory(NewPolicyGet)
	}
	return nil
}
func (PolicyContextComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPolicyContext" {
		return custom.Factory(NewPolicyContext)
	}
	return nil
}
func (PolicyReplaceComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPolicyReplace" {
		return custom.Factory(NewPolicyReplace)
	}
	return nil
}
func (AuthorizationComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewAuthorizationCheck" {
		return custom.Factory(NewAuthorizationCheck)
	}
	return nil
}

type policyGetHandler struct{}
type policyContextHandler struct{}
type policyReplaceHandler struct{}
type authorizationHandler struct{}

func NewPolicyGet() xhandler.Contract[PolicyInput, sharedapi.PolicyOutput] {
	return &policyGetHandler{}
}
func NewPolicyContext() xhandler.Contract[PolicyInput, sharedapi.PolicyContextOutput] {
	return &policyContextHandler{}
}
func NewPolicyReplace() xhandler.Contract[PolicyWriteInput, sharedapi.PolicyOutput] {
	return &policyReplaceHandler{}
}
func NewAuthorizationCheck() xhandler.Contract[AuthorizationInput, sharedapi.DecisionOutput] {
	return &authorizationHandler{}
}
func (*policyGetHandler) Exec(ctx context.Context, session xhandler.Session, input *PolicyInput, output *sharedapi.PolicyOutput) error {
	if input == nil || output == nil {
		return publicError(400, "resource identity is required")
	}
	value := &DocumentOutput{}
	err := NewGet().Exec(ctx, session, &ResourceInput{NamespaceId: input.NamespaceId, Auth: input.Auth, Jwt: input.Jwt, Resource: input.Resource}, value)
	if err == nil {
		output.Document = *value.Response
	}
	return err
}
func (*policyContextHandler) Exec(ctx context.Context, session xhandler.Session, input *PolicyInput, output *sharedapi.PolicyContextOutput) error {
	if input == nil || output == nil {
		return publicError(400, "resource identity is required")
	}
	value := &ContextOutput{}
	err := NewContext().Exec(ctx, session, &ResourceInput{NamespaceId: input.NamespaceId, Auth: input.Auth, Jwt: input.Jwt, Resource: input.Resource}, value)
	if err == nil {
		output.Context = *value.Response
	}
	return err
}
func (*policyReplaceHandler) Exec(ctx context.Context, session xhandler.Session, input *PolicyWriteInput, output *sharedapi.PolicyOutput) error {
	if input == nil || output == nil {
		return publicError(400, "policy document is required")
	}
	value := &DocumentOutput{}
	err := NewReplace().Exec(ctx, session, &ReplaceInput{NamespaceId: input.NamespaceId, Auth: input.Auth, Jwt: input.Jwt, Document: input.Document}, value)
	if err == nil {
		output.Document = *value.Response
	}
	return err
}
func (*authorizationHandler) Exec(ctx context.Context, session xhandler.Session, input *AuthorizationInput, output *sharedapi.DecisionOutput) error {
	if input == nil || output == nil {
		return publicError(400, "resource identity and action are required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if err := checkResourceNamespace(requestCtx, service, input.NamespaceId, input.Resource, input.Auth, input.Jwt); err != nil {
		return err
	}
	return sharedapi.NewCheck().Exec(requestCtx, sharedSession(session, service), &sharedapi.CheckInput{JWT: input.Jwt, Resource: input.Resource, Action: input.Action, Selection: input.Selection}, output)
}
