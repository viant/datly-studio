package access

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/store/sql/accesscatalog"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	reports "github.com/viant/datly-studio/studio/reports/get"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"github.com/viant/datly/spec"
	"os"
	"reflect"
	"strings"
	"time"

	store "github.com/viant/authz/datly/store/sql"
	accessoauth "github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/accessconfig"
	"github.com/viant/datly-studio/sdk"
	acl "github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type ResourceInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Resource    authz.Resource     `parameter:"Resource,kind=body,in=,dataType=authz.Resource,required=true" anonymous:"true"`
}

type ReplaceInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Document    authz.Document     `parameter:"Document,kind=body,in=,dataType=authz.Document,required=true" anonymous:"true"`
}

type DocumentOutput struct {
	Response *authz.Document `parameter:"Response,kind=output,in=body,dataType=*authz.Document" json:"-"`
}

func (*DocumentOutput) JSONWireType() reflect.Type { return reflect.TypeFor[authz.Document]() }
func (output *DocumentOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type ContextOutput struct {
	Response *authz.EditorContext `parameter:"Response,kind=output,in=body,dataType=*authz.EditorContext" json:"-"`
}

func (*ContextOutput) JSONWireType() reflect.Type { return reflect.TypeFor[authz.EditorContext]() }
func (output *ContextOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type GetComponent struct {
	Contract xdatly.Component[ResourceInput, DocumentOutput] `component:"access_get,path=/v1/studio/sdk/access.get,method=POST,connector=studio,handler=NewGet" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.access.get\",\"description\":\"Read one authorized resource access policy\"}]" caseFormat:"lc"`
}

type ContextComponent struct {
	Contract xdatly.Component[ResourceInput, ContextOutput] `component:"access_context,path=/v1/studio/sdk/access.context,method=POST,connector=studio,handler=NewContext" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.access.context\",\"description\":\"Inspect verified policy editor context\"}]" caseFormat:"lc"`
}

type ReplaceComponent struct {
	Contract xdatly.Component[ReplaceInput, DocumentOutput] `component:"access_replace,path=/v1/studio/sdk/access.replace,method=POST,connector=studio,handler=NewReplace" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.access.replace\",\"description\":\"Replace one authorized resource policy at its expected revision\"}]" caseFormat:"lc"`
}

var (
	GetDatly              = new(GetComponent)
	ContextDatly          = new(ContextComponent)
	ReplaceDatly          = new(ReplaceComponent)
	componentReachability = []reflect.Type{reflect.TypeFor[ListComponent](), reflect.TypeFor[GetComponent](), reflect.TypeFor[ContextComponent](), reflect.TypeFor[ReplaceComponent]()}
)

func init() {}

func (GetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewGet" {
		return custom.Factory(NewGet)
	}
	return nil
}
func (ContextComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewContext" {
		return custom.Factory(NewContext)
	}
	return nil
}
func (ReplaceComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReplace" {
		return custom.Factory(NewReplace)
	}
	return nil
}

type getHandler struct{}
type contextHandler struct{}
type replaceHandler struct{}

func NewGet() xhandler.Contract[ResourceInput, DocumentOutput]    { return &getHandler{} }
func NewContext() xhandler.Contract[ResourceInput, ContextOutput] { return &contextHandler{} }
func NewReplace() xhandler.Contract[ReplaceInput, DocumentOutput] { return &replaceHandler{} }

func (*getHandler) Exec(ctx context.Context, session xhandler.Session, input *ResourceInput, output *DocumentOutput) error {
	if input == nil || output == nil {
		return publicError(400, "resource identity is required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if err := checkResourceNamespace(requestCtx, service, input.NamespaceId, input.Resource, input.Auth, input.Jwt); err != nil {
		return err
	}
	value, err := service.Get(requestCtx, input.Resource)
	if err != nil {
		return mapError(err)
	}
	output.Response = &value
	return nil
}

func (*contextHandler) Exec(ctx context.Context, session xhandler.Session, input *ResourceInput, output *ContextOutput) error {
	if input == nil || output == nil {
		return publicError(400, "resource identity is required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if err := checkResourceNamespace(requestCtx, service, input.NamespaceId, input.Resource, input.Auth, input.Jwt); err != nil {
		return err
	}
	value, err := service.EditorContext(requestCtx, input.Resource)
	if err != nil {
		return mapError(err)
	}
	output.Response = &value
	return nil
}

func (*replaceHandler) Exec(ctx context.Context, session xhandler.Session, input *ReplaceInput, output *DocumentOutput) error {
	if input == nil || output == nil {
		return publicError(400, "policy document is required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if err := checkResourceNamespace(requestCtx, service, input.NamespaceId, input.Document.Resource, input.Auth, input.Jwt); err != nil {
		return err
	}
	value, err := service.Replace(requestCtx, input.Document)
	if err != nil {
		return mapError(err)
	}
	output.Response = &value
	return nil
}

func checkResourceNamespace(ctx context.Context, service *authz.Service, namespaceID *string, resource authz.Resource, auth *studioauth.Output, claims *jwt.Claims) error {
	if namespaceID == nil {
		return nil
	}
	policyStore, ok := service.Store.(*store.Store)
	if !ok || auth == nil || auth.Auth == nil || claims == nil || auth.Auth.Subject != claims.Subject {
		return publicError(403, "Resource access is not permitted")
	}
	source := &accesscatalog.Store{DB: policyStore.DB, Invoker: policyStore.Invoker}
	err := acl.CheckNamespaceResource(ctx, namespaceID, resource, source, func(ctx context.Context, id string) bool {
		read := &reports.ReportGetInput{}
		read.SetJwt(claims)
		read.SetAuth(auth)
		read.SetId(id)
		read.SetNamespaceId(namespaceID)
		value, err := policyStore.Invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[reports.ReportComponent]().PkgPath(), Name: "report"}, Route: spec.RouteRef{Method: "POST", Path: "/v1/studio/sdk/components.get"}}, Input: read})
		output, ok := value.(*reports.ReportGetOutput)
		return err == nil && ok && output != nil && output.Item != nil && output.Item.Id == id
	})
	if err != nil {
		return publicError(403, "Resource access is not permitted")
	}
	return nil
}

func setup(ctx context.Context, session xhandler.Session, claims *jwt.Claims) (*authz.Service, context.Context, func(), error) {
	if claims == nil || claims.Subject == "" {
		return nil, nil, nil, publicError(401, "verified Studio principal is required")
	}
	if session == nil || session.Binder() == nil {
		return nil, nil, nil, fmt.Errorf("access handler session is required")
	}
	var header struct {
		Authorization string `bind:"kind=header,in=Authorization,required"`
	}
	if err := session.Binder().Bind(ctx, &header); err != nil {
		return nil, nil, nil, publicError(401, "ACL bearer credential is required")
	}
	authorization := header.Authorization
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) || strings.TrimSpace(authorization[len(prefix):]) == "" {
		return nil, nil, nil, publicError(401, "ACL bearer credential is required")
	}
	provider, err := accessconfig.FromEnvironment()
	if err != nil || provider == nil {
		return nil, nil, nil, publicError(503, "ACL verifier is not configured correctly")
	}
	value, found, err := session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if err != nil {
		return nil, nil, nil, err
	}
	connector, ok := value.(xconnector.Provider)
	if !found || !ok {
		return nil, nil, nil, fmt.Errorf("trusted Studio connector capability is unavailable")
	}
	db, err := connector.Connector(ctx, "studio")
	if err != nil || db == nil {
		return nil, nil, nil, fmt.Errorf("configured Studio database is unavailable")
	}
	value, found, err = session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, nil, nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, nil, nil, fmt.Errorf("Datly component invoker is unavailable")
	}
	policyStore := &store.Store{DB: db, Invoker: invoker}
	service := &authz.Service{Store: policyStore, Provider: subjectProvider{Provider: provider, subject: claims.Subject}}
	requestCtx := accessoauth.WithBearer(ctx, strings.TrimSpace(authorization[len(prefix):]))
	return service, requestCtx, func() { _ = policyStore.Close(context.WithoutCancel(ctx)) }, nil
}

type subjectProvider struct {
	authz.Provider
	subject string
}

func (p subjectProvider) Resolve(ctx context.Context) (authz.Facts, error) {
	facts, err := p.Provider.Resolve(ctx)
	if err != nil || facts.Subject != p.subject {
		return authz.Facts{}, authz.ErrDenied
	}
	return facts, nil
}

func mapError(err error) error {
	if errors.Is(err, authz.ErrConflict) {
		return publicError(409, "Access policy changed. Reload before saving.")
	}
	return publicError(403, "Resource access is not permitted")
}

func publicError(code int, message string) error {
	category := sdk.ErrorInternal
	switch code {
	case 400:
		category = sdk.ErrorInvalidArgument
	case 401:
		category = sdk.ErrorUnauthorized
	case 403:
		category = sdk.ErrorForbidden
	case 404:
		category = sdk.ErrorNotFound
	case 409:
		category = sdk.ErrorConflict
	case 503:
		category = sdk.ErrorUnavailable
	}
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]any{"code": category, "message": message}}
}

// List exposes only resources whose exact effective policy can be inspected.
type ListInput struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Query       acl.CatalogInput   `parameter:"Query,kind=body,in=,dataType=acl.CatalogInput,required=true" anonymous:"true"`
}
type ListOutput struct {
	Response *acl.CatalogPage `parameter:"Response,kind=output,in=body,dataType=*acl.CatalogPage" json:"-"`
}

func (*ListOutput) JSONWireType() reflect.Type     { return reflect.TypeFor[acl.CatalogPage]() }
func (o *ListOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type ListComponent struct {
	Contract xdatly.Component[ListInput, ListOutput] `component:"access_list,path=/v1/studio/sdk/access.list,method=POST,connector=studio,handler=NewList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.access.list\",\"description\":\"List authorized permission resources\"}]" caseFormat:"lc"`
}

var ListDatly = new(ListComponent)
var ListDatlyLinkedType = reflect.TypeFor[ListComponent]()

func (ListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewList" {
		return custom.Factory(NewList)
	}
	return nil
}

type listHandler struct{}

func NewList() xhandler.Contract[ListInput, ListOutput] { return &listHandler{} }

// The authoring catalog lists components authorized by Studio. Independent ACL
// facts remain required to inspect/edit their policies or execute scoped tools.
func (*listHandler) Exec(ctx context.Context, session xhandler.Session, input *ListInput, output *ListOutput) error {
	if input == nil || output == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil || input.Jwt.Subject != input.Auth.Auth.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, input.Jwt)
	if err != nil {
		if os.Getenv("STUDIO_ACCESS_ISSUER") != "" || os.Getenv("STUDIO_ACCESS_AUDIENCE") != "" || os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE") != "" {
			return err
		}
		value, found, e := session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
		if e != nil {
			return e
		}
		connector, ok := value.(xconnector.Provider)
		if !found || !ok {
			return fmt.Errorf("Studio connector capability is unavailable")
		}
		db, e := connector.Connector(ctx, "studio")
		if e != nil {
			return e
		}
		policyStore := &store.Store{DB: db}
		service = &authz.Service{Store: policyStore, Provider: catalogPrincipal{claims: input.Jwt}}
		requestCtx = ctx
		closeStore = func() { _ = policyStore.Close(context.WithoutCancel(ctx)) }
	}
	defer closeStore()
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Studio component invoker is unavailable")
	}
	visible := map[string]bool{}
	catalogService := &acl.Catalog{Service: service}
	if policyStore, ok := service.Store.(*store.Store); ok {
		catalogService.Source = &accesscatalog.Store{DB: policyStore.DB, Invoker: policyStore.Invoker}
	}
	catalogService.AuthoringAccess = func(ctx context.Context, id string) bool {
		if id == "" {
			return false
		}
		if allowed, seen := visible[id]; seen {
			return allowed
		}
		read := &reports.ReportGetInput{}
		read.SetJwt(input.Jwt)
		read.SetAuth(input.Auth)
		read.SetId(id)
		read.SetNamespaceId(input.NamespaceId)
		value, e := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[reports.ReportComponent]().PkgPath(), Name: "report"}, Route: spec.RouteRef{Method: "POST", Path: "/v1/studio/sdk/components.get"}}, Input: read})
		item, ok := value.(*reports.ReportGetOutput)
		allowed := e == nil && ok && item != nil && item.Item != nil && item.Item.Id == id
		visible[id] = allowed
		return allowed
	}
	if input.NamespaceId != nil {
		// Keep workspace scope independent from policy administration grants.
		catalogService.ResourceScope = func(ctx context.Context, row *catalog.Entry) bool {
			return row.NamespaceID == *input.NamespaceId && catalogService.AuthoringAccess(ctx, row.ComponentID)
		}
	}
	page, err := catalogService.List(requestCtx, input.Query)
	if err != nil {
		return mapError(err)
	}
	output.Response = &page
	return nil
}

type catalogPrincipal struct{ claims *jwt.Claims }

func (p catalogPrincipal) Resolve(context.Context) (authz.Facts, error) {
	if p.claims == nil || p.claims.ExpiresAt == nil || !p.claims.ExpiresAt.Time.After(time.Now()) {
		return authz.Facts{}, authz.ErrDenied
	}
	return authz.Facts{Subject: p.claims.Subject, Issuer: p.claims.Issuer, ValidUntil: p.claims.ExpiresAt.Time}, nil
}
