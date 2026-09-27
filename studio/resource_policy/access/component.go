package access

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk"
	acl "github.com/viant/datly-studio/sdk/access"
	accessoauth "github.com/viant/datly-studio/sdk/access/oauth"
	store "github.com/viant/datly-studio/store/sql/access"
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
	Jwt      *jwt.Claims  `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Resource acl.Resource `parameter:"Resource,kind=body,in=,dataType=acl.Resource,required=true" anonymous:"true"`
}

type ReplaceInput struct {
	Jwt      *jwt.Claims  `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Document acl.Document `parameter:"Document,kind=body,in=,dataType=acl.Document,required=true" anonymous:"true"`
}

type DocumentOutput struct {
	Response *acl.Document `parameter:"Response,kind=output,in=body,dataType=*acl.Document" json:"-"`
}

func (*DocumentOutput) JSONWireType() reflect.Type { return reflect.TypeFor[acl.Document]() }
func (output *DocumentOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type ContextOutput struct {
	Response *acl.EditorContext `parameter:"Response,kind=output,in=body,dataType=*acl.EditorContext" json:"-"`
}

func (*ContextOutput) JSONWireType() reflect.Type { return reflect.TypeFor[acl.EditorContext]() }
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
	componentReachability = []reflect.Type{reflect.TypeFor[GetComponent](), reflect.TypeFor[ContextComponent](), reflect.TypeFor[ReplaceComponent]()}
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
	value, err := service.Replace(requestCtx, input.Document)
	if err != nil {
		return mapError(err)
	}
	output.Response = &value
	return nil
}

func setup(ctx context.Context, session xhandler.Session, claims *jwt.Claims) (*acl.Service, context.Context, func(), error) {
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
	issuer, audience, keyPath := os.Getenv("STUDIO_ACCESS_ISSUER"), os.Getenv("STUDIO_ACCESS_AUDIENCE"), os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE")
	if issuer == "" || audience == "" || keyPath == "" {
		return nil, nil, nil, publicError(503, "ACL verifier is not configured")
	}
	pem, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, nil, publicError(503, "ACL public key is unavailable")
	}
	key, err := jwtlib.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, nil, nil, publicError(503, "ACL public key is invalid")
	}
	keyfunc := func(*jwtlib.Token) (any, error) { return key, nil }
	var provider acl.Provider
	if userInfoURL := strings.TrimSpace(os.Getenv("STUDIO_ACCESS_USER_INFO_URL")); userInfoURL != "" {
		provider, err = accessoauth.NewUserInfo(accessoauth.UserInfoConfig{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc, URL: userInfoURL})
	} else {
		provider, err = accessoauth.New(accessoauth.Config{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc})
	}
	if err != nil {
		return nil, nil, nil, publicError(503, "ACL verifier is invalid")
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
	service := &acl.Service{Store: policyStore, Provider: subjectProvider{Provider: provider, subject: claims.Subject}}
	requestCtx := accessoauth.WithBearer(ctx, strings.TrimSpace(authorization[len(prefix):]))
	return service, requestCtx, func() { _ = policyStore.Close(context.WithoutCancel(ctx)) }, nil
}

type subjectProvider struct {
	acl.Provider
	subject string
}

func (p subjectProvider) Resolve(ctx context.Context) (acl.Facts, error) {
	facts, err := p.Provider.Resolve(ctx)
	if err != nil || facts.Subject != p.subject {
		return acl.Facts{}, acl.ErrDenied
	}
	return facts, nil
}

func mapError(err error) error {
	if errors.Is(err, acl.ErrConflict) {
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
