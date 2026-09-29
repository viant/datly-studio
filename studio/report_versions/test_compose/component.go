package test_compose

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

type Input struct {
	NamespaceId *string                  `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string                   `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                      `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Input       sdk.CubeComposeTestInput `parameter:"Input,kind=body,in=input,dataType=sdk.CubeComposeTestInput,required=true" json:"input"`
}

type Output struct {
	Data        json.RawMessage  `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Duration    time.Duration    `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
	Diagnostics []sdk.Diagnostic `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.test_compose,method=POST,connector=studio,handler=NewTestCompose" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.test_compose\",\"description\":\"Test cube composition on an authorized exact Datly reader version\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTestCompose" {
		return custom.Factory(NewTestCompose)
	}
	return nil
}

type handler struct{}

func NewTestCompose() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("cube composition input and output are required")
	}
	executionCtx, engine, cancel, err := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportID, input.VersionNo, 30*time.Second)
	if err != nil {
		return err
	}
	defer cancel()
	result, err := engine.TestCompose(executionCtx, input.ReportID, input.VersionNo, input.Input)
	if err != nil {
		return versionrun.PublicError(err, "cube composition test")
	}
	if result == nil {
		return fmt.Errorf("cube composition test returned no result")
	}
	*output = Output{Data: append(json.RawMessage(nil), result.Data...), Duration: result.Duration, Diagnostics: result.Diagnostics}
	return nil
}
