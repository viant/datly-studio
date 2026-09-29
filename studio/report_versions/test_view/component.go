package test_view

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/publisherguard"
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
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	View        string             `parameter:"View,kind=body,in=view,dataType=string,required=true" json:"view"`
	Input       sdk.ViewTestInput  `parameter:"Input,kind=body,in=input,dataType=sdk.ViewTestInput,required=false" json:"input"`
}

type Output struct {
	View        string                `parameter:"View,kind=output,in=body,dataType=string" json:"view"`
	Data        json.RawMessage       `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Duration    time.Duration         `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
	Diagnostics []sdk.Diagnostic      `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
	Evidence    sdk.ExecutionEvidence `parameter:"Evidence,kind=output,in=body,dataType=sdk.ExecutionEvidence" json:"evidence"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.test_view,method=POST,connector=studio,handler=NewTestView" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.test_view\",\"description\":\"Run one view of an authorized exact Datly reader version\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTestView" {
		return custom.Factory(NewTestView)
	}
	return nil
}

type handler struct{}

func NewTestView() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("view test input and output are required")
	}
	if strings.TrimSpace(input.View) == "" {
		return publisherguard.PublicError(400, "view is required")
	}
	executionCtx, engine, cancel, err := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportID, input.VersionNo, 30*time.Second)
	if err != nil {
		return err
	}
	defer cancel()
	result, err := engine.TestView(executionCtx, input.ReportID, input.VersionNo, input.View, input.Input)
	if err != nil {
		return versionrun.PublicError(err, "view test")
	}
	if result == nil || result.Evidence.ReportID != input.ReportID || result.Evidence.VersionNo != input.VersionNo {
		return fmt.Errorf("view test returned mismatched execution evidence")
	}
	*output = Output{View: result.View, Data: append(json.RawMessage(nil), result.Data...),
		Duration: result.Duration, Diagnostics: result.Diagnostics, Evidence: result.Evidence}
	return nil
}
