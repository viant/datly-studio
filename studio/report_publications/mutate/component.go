package mutate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type VersionInput struct {
	Jwt       *jwt.Claims      `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportID  string           `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo int              `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Input     sdk.PublishInput `parameter:"Input,kind=body,in=input,dataType=sdk.PublishInput,required=true" json:"input"`
}

type UnpublishInput struct {
	Jwt      *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportID string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input    sdk.UnpublishInput `parameter:"Input,kind=body,in=input,dataType=sdk.UnpublishInput,required=true" json:"input"`
}

type Output struct {
	Response *sdk.Publication `parameter:"Response,kind=output,in=body,dataType=*sdk.Publication" json:"-"`
}

func (*Output) JSONWireType() reflect.Type { return reflect.TypeFor[sdk.Publication]() }

func (output *Output) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type PublishComponent struct {
	Contract xdatly.Component[VersionInput, Output] `component:"publication_publish,path=/v1/studio/sdk/publications.publish,method=POST,connector=studio,handler=NewPublish" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.publications.publish\",\"description\":\"Publish one validated Datly Studio component version\"}]" caseFormat:"lc"`
}

type RollbackComponent struct {
	Contract xdatly.Component[VersionInput, Output] `component:"publication_rollback,path=/v1/studio/sdk/publications.rollback,method=POST,connector=studio,handler=NewRollback" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.publications.rollback\",\"description\":\"Rollback one authorized Datly Studio publication\"}]" caseFormat:"lc"`
}

type UnpublishComponent struct {
	Contract xdatly.Component[UnpublishInput, Output] `component:"publication_unpublish,path=/v1/studio/sdk/publications.unpublish,method=POST,connector=studio,handler=NewUnpublish" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.publications.unpublish\",\"description\":\"Unpublish one authorized Datly Studio component\"}]" caseFormat:"lc"`
}

var (
	PublishDatly          = new(PublishComponent)
	RollbackDatly         = new(RollbackComponent)
	UnpublishDatly        = new(UnpublishComponent)
	componentReachability = []reflect.Type{reflect.TypeFor[PublishComponent](), reflect.TypeFor[RollbackComponent](), reflect.TypeFor[UnpublishComponent]()}
	publicationMu         sync.Mutex
)

func init() {}

func (PublishComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPublish" {
		return custom.Factory(NewPublish)
	}
	return nil
}

func (RollbackComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewRollback" {
		return custom.Factory(NewRollback)
	}
	return nil
}

func (UnpublishComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUnpublish" {
		return custom.Factory(NewUnpublish)
	}
	return nil
}

type versionHandler struct{ operation string }
type unpublishHandler struct{}

func NewPublish() xhandler.Contract[VersionInput, Output] {
	return &versionHandler{operation: sdk.OperationPublicationPublish}
}
func NewRollback() xhandler.Contract[VersionInput, Output] {
	return &versionHandler{operation: sdk.OperationPublicationRollback}
}
func NewUnpublish() xhandler.Contract[UnpublishInput, Output] { return &unpublishHandler{} }

func (handler *versionHandler) Exec(ctx context.Context, session xhandler.Session, input *VersionInput, output *Output) error {
	if input == nil || input.Jwt == nil || input.Jwt.Subject == "" {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportID) == "" || input.VersionNo <= 0 || input.Input.ExpectedSourceRevision <= 0 {
		return publicError(400, "reportId, versionNo, and positive expectedSourceRevision are required")
	}
	return invoke(ctx, session, input.Jwt.Subject, handler.operation, struct {
		ReportID  string           `json:"reportId"`
		VersionNo int              `json:"versionNo"`
		Input     sdk.PublishInput `json:"input"`
	}{input.ReportID, input.VersionNo, input.Input}, output)
}

func (*unpublishHandler) Exec(ctx context.Context, session xhandler.Session, input *UnpublishInput, output *Output) error {
	if input == nil || input.Jwt == nil || input.Jwt.Subject == "" {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportID) == "" || input.Input.ExpectedActiveGeneration <= 0 {
		return publicError(400, "reportId and positive expectedActiveGeneration are required")
	}
	return invoke(ctx, session, input.Jwt.Subject, sdk.OperationPublicationRemove, struct {
		ReportID string             `json:"reportId"`
		Input    sdk.UnpublishInput `json:"input"`
	}{input.ReportID, input.Input}, output)
}

func invoke(ctx context.Context, session xhandler.Session, subject, operation string, body any, output *Output) error {
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("publication session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
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
	token := os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN")
	if token == "" {
		return publicError(503, "dynamic runtime admin token is not configured")
	}
	adminURL := os.Getenv("STUDIO_DYNAMIC_HTTP_URL")
	if adminURL == "" {
		adminURL = "http://127.0.0.1:8082"
	}
	admin, err := runtimeadmin.New(adminURL, token, nil)
	if err != nil {
		return publicError(503, "dynamic runtime admin origin is invalid")
	}
	transport := &sqltransport.Transport{DB: db, Authorizer: &authorization.SDKAuthorizer{DB: db}, Activator: admin}
	defer transport.Close(context.WithoutCancel(ctx))
	// Publication is a staged lifecycle: its first database transaction must
	// commit before the external runtime reload, followed by activation or
	// compensation. It cannot join Datly's single-commit component unit. Carry
	// only cancellation/deadline and the verified principal into the existing
	// lifecycle owner; do not leak the endpoint's transaction lookup into its
	// separately committed phases.
	if err := ctx.Err(); err != nil {
		return err
	}
	var lifecycleCtx context.Context
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		lifecycleCtx, cancel = context.WithDeadline(context.Background(), deadline)
	} else {
		lifecycleCtx, cancel = context.WithCancel(context.Background())
	}
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	defer cancel()
	principal := sdk.WithPrincipal(lifecycleCtx, sdk.Principal{Subject: subject})
	publicationMu.Lock()
	defer publicationMu.Unlock()
	var response sdk.Publication
	if err = transport.Invoke(principal, operation, body, &response); err != nil {
		return mapError(err)
	}
	output.Response = &response
	return nil
}

func mapError(err error) error {
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) {
		return publicError(500, "publication failed")
	}
	code := 500
	switch sdkErr.Code {
	case sdk.ErrorInvalidArgument:
		code = 400
	case sdk.ErrorUnauthorized:
		code = 401
	case sdk.ErrorForbidden:
		code = 403
	case sdk.ErrorNotFound:
		code = 404
	case sdk.ErrorConflict:
		code = 409
	case sdk.ErrorUnavailable:
		code = 503
	}
	payload := map[string]any{"code": sdkErr.Code, "message": sdkErr.Message}
	if sdkErr.Field != "" {
		payload["field"] = sdkErr.Field
	}
	if sdkErr.ExpectedSourceRevision != 0 {
		payload["expectedSourceRevision"] = sdkErr.ExpectedSourceRevision
	}
	if sdkErr.CurrentSourceRevision != 0 {
		payload["currentSourceRevision"] = sdkErr.CurrentSourceRevision
	}
	return &xresponse.Error{Code: code, Cause: errors.New(sdkErr.Message), Payload: payload}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
