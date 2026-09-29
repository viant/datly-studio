package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/datly-studio/internal/versionrun"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

const maxExecution = 30 * time.Second

type Input struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Input       sdk.PreviewInput   `parameter:"Input,kind=body,in=input,dataType=sdk.PreviewInput,required=false" json:"input"`
}

type Output struct {
	Data        json.RawMessage       `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Diagnostics []sdk.Diagnostic      `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
	Duration    time.Duration         `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
	Evidence    sdk.ExecutionEvidence `parameter:"Evidence,kind=output,in=body,dataType=sdk.ExecutionEvidence" json:"evidence"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"preview,path=/v1/studio/sdk/preview.execute,method=POST,connector=studio,handler=NewExecute" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.preview.execute\",\"description\":\"Run bounded exact-version Datly preview with verified report run access\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewExecute" {
		return custom.Factory(NewExecute)
	}
	return nil
}

type handler struct{}

func NewExecute() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("preview execution input and output are required")
	}
	executionCtx, engine, cancel, err := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportID, input.VersionNo, maxExecution)
	if err != nil {
		return err
	}
	defer cancel()
	result, err := engine.Execute(executionCtx, input.ReportID, input.VersionNo, input.Input)
	if err != nil {
		return versionrun.PublicError(err, "reader preview")
	}
	if result == nil || result.Evidence.ReportID != input.ReportID || result.Evidence.VersionNo != input.VersionNo {
		return fmt.Errorf("preview engine returned mismatched execution evidence")
	}
	output.Data = append(output.Data[:0], result.Data...)
	output.Diagnostics, output.Duration, output.Evidence = result.Diagnostics, result.Duration, result.Evidence
	return nil
}
