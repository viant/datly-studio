package test_sql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/internal/runtimevalidation"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

const defaultExecution = 60 * time.Second

type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.SQLTestInput   `parameter:"Input,kind=body,in=input,dataType=sdk.SQLTestInput,required=true" json:"input"`
}

type Output struct {
	Offset       int             `parameter:"Offset,kind=output,in=body,dataType=int" json:"offset"`
	Limit        int             `parameter:"Limit,kind=output,in=body,dataType=int" json:"limit"`
	ReturnedRows int             `parameter:"ReturnedRows,kind=output,in=body,dataType=int" json:"returnedRows"`
	HasMore      bool            `parameter:"HasMore,kind=output,in=body,dataType=bool" json:"hasMore"`
	Data         json.RawMessage `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Duration     time.Duration   `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.test_sql,method=POST,connector=studio,handler=NewTestSQL" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.test_sql\",\"description\":\"Execute bounded transient SQL through Datly on an authorized Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTestSQL" {
		return custom.Factory(NewTestSQL)
	}
	return nil
}

type handler struct{}

func NewTestSQL() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector SQL test input and output are required")
	}
	if input.Input.Offset < 0 {
		return publisherguard.PublicError(400, "SQL result offset must be nonnegative")
	}
	timeout, err := executionTimeout(input.Input.TimeoutSeconds)
	if err != nil {
		return err
	}
	connector, err := connectorcatalog.ForProbe(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	if connector.Status != "active" {
		return publisherguard.PublicError(400, "connector must be active before testing SQL")
	}
	if strings.TrimSpace(input.Input.SQL) == "" {
		return publisherguard.PublicError(400, "SQL is required")
	}
	authoring, err := host.AuthoringFromContext(ctx)
	if err != nil {
		return err
	}
	types, err := authoring.RuntimeTypes()
	if err != nil {
		return err
	}
	executionCtx, cancel := context.WithCancel(ctx)
	if timeout > 0 {
		cancel()
		executionCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	result, err := (preview.Dynamic{ModulePath: authoring.ModulePath(), Types: types, Const: authoring.Constants(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies()}).TestSQL(executionCtx, connector, input.Input)
	if err != nil {
		return executionError(executionCtx, err, timeout)
	}
	if result == nil {
		return fmt.Errorf("transient SQL test returned no result")
	}
	output.Data, output.Duration = result.Data, result.Duration
	output.Offset, output.Limit, output.ReturnedRows, output.HasMore = result.Offset, result.Limit, result.ReturnedRows, result.HasMore
	return nil
}

func executionTimeout(seconds *int) (time.Duration, error) {
	if seconds == nil {
		return defaultExecution, nil
	}
	if *seconds < 0 || int64(*seconds) > math.MaxInt64/int64(time.Second) {
		return 0, publisherguard.PublicError(400, "SQL timeout must be a nonnegative, representable number of seconds; 0 disables the Studio timeout")
	}
	return time.Duration(*seconds) * time.Second, nil
}

func executionError(ctx context.Context, err error, timeout time.Duration) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		if timeout == 0 {
			return publisherguard.PublicError(504, "The request deadline expired. The Studio SQL timeout was disabled.")
		}
		return publisherguard.PublicError(504, fmt.Sprintf("SQL query timed out (configured timeout: %d seconds). Increase Timeout (seconds) or narrow the query.", int(timeout/time.Second)))
	}
	if provider := providerExecutionError(err); provider != nil {
		message := provider.summary()
		return &xresponse.Error{Code: 400, Cause: errors.New(message), Payload: map[string]any{"message": message, "providerError": provider}}
	}
	diagnostics := runtimevalidation.Diagnostics(runtimevalidation.Wrap("contract", err))
	if len(diagnostics) == 1 && diagnostics[0].Stage == "contract.discovery" && strings.HasPrefix(diagnostics[0].Code, "runtime_discovery_") {
		// Only the static allowlisted runtime projection crosses the operator
		// boundary. Never return the original error or SQL as a payload/cause.
		message := "SQL test could not be compiled or executed"
		return &xresponse.Error{Code: 400, Cause: errors.New(message), Payload: map[string]any{"message": message, "diagnostics": diagnostics}}
	}
	return publisherguard.PublicError(400, "SQL test could not be compiled or executed")
}
