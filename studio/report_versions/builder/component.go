package builder

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

func init() {}

type Input struct {
	NamespaceId *string                  `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Auth        *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	ReportId    string                   `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                      `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Command     sdk.ReaderBuilderCommand `parameter:"Command,kind=body,in=command,dataType=sdk.ReaderBuilderCommand,required=true" json:"command"`
}

type Output struct {
	Applied    bool                  `parameter:"Applied,kind=output,in=body,dataType=bool" json:"applied"`
	Inspection *sdk.ReaderInspection `parameter:"Inspection,kind=output,in=body,dataType=*sdk.ReaderInspection" json:"inspection"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.builder,method=POST,connector=studio,handler=NewReaderBuilder" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.builder\",\"description\":\"Apply a versioned Datly Reader Builder command\"}]" caseFormat:"lc"`
}

var BuilderComponent = new(Component)
var BuilderComponentType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReaderBuilder" {
		return custom.Factory(NewReaderBuilder)
	}
	return nil
}

type handler struct{}

func NewReaderBuilder() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Jwt.Subject != input.Auth.Auth.Subject {
		return publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportId) == "" || input.VersionNo <= 0 || input.Command.ExpectedSourceRevision <= 0 || len(input.Command.Operation) == 0 {
		return publisherguard.PublicError(400, "reportId, versionNo, revision, and builder operation are required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("reader builder session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	access := &guard.Input{}
	access.SetJwt(input.Jwt)
	access.SetAuth(input.Auth)
	access.SetReportId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[guard.ReportComponent]().PkgPath(), Name: "report"},
		Route:     spec.RouteRef{Method: "POST", Path: "/_studio/reports/edit-guard"},
	}, Input: access})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != input.ReportId {
		return publisherguard.PublicError(403, "report edit access is required")
	}
	value, found, err = session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if err != nil {
		return err
	}
	provider, ok := value.(xconnector.Provider)
	if !found || !ok {
		return fmt.Errorf("trusted Studio connector capability is unavailable")
	}
	db, err := provider.Connector(ctx, "studio")
	if err != nil || db == nil {
		return fmt.Errorf("configured Studio database is unavailable")
	}
	types, err := (host.Config{}).RuntimeTypes()
	if err != nil {
		return err
	}
	engine := &sqltransport.Transport{DB: db, ComponentInvoker: invoker, ContractInspector: preview.Dynamic{StudioDB: db, ModulePath: "github.com/viant/datly-studio", Types: types}}
	request := struct {
		ReportID  string                   `json:"reportId"`
		VersionNo int                      `json:"versionNo"`
		Command   sdk.ReaderBuilderCommand `json:"command"`
	}{input.ReportId, input.VersionNo, input.Command}
	var result sdk.ReaderBuilderResult
	ctx = sdk.WithPrincipal(ctx, sdk.Principal{Subject: input.Jwt.Subject})
	if err := engine.ApplyReaderBuilder(ctx, request, &result); err != nil {
		return publicBuilderError(err)
	}
	*output = Output{Applied: result.Applied, Inspection: result.Inspection}
	return nil
}

func publicBuilderError(err error) error {
	var value *sdk.Error
	if !errors.As(err, &value) {
		return publisherguard.PublicError(502, "reader builder failed")
	}
	status := 502
	switch value.Code {
	case sdk.ErrorInvalidArgument:
		status = 400
	case sdk.ErrorNotFound:
		status = 404
	case sdk.ErrorForbidden:
		status = 403
	case sdk.ErrorConflict:
		status = 409
	case sdk.ErrorUnavailable:
		status = 503
	}
	message := "reader builder failed"
	payload := map[string]any{"message": message}
	if status == 409 {
		message = "reader version changed; reload before saving"
		payload = map[string]any{"message": message, "code": "conflict", "field": "sourceRevision"}
	}
	return &xresponse.Error{Code: status, Cause: errors.New(message), Payload: payload}
}
