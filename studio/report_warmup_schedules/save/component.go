package save

import (
	"context"
	"encoding/json"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	shared "github.com/viant/datly-studio/studio/report_warmup_schedules"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

type Input struct {
	NamespaceId *string                     `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims                 `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Auth        *studioauth.Output          `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	ReportID    string                      `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                         `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Input       sdk.SaveWarmupScheduleInput `parameter:"Input,kind=body,in=input,dataType=sdk.SaveWarmupScheduleInput,required=true" json:"input"`
}
type Output struct {
	Response *sdk.WarmupScheduleResult `parameter:"Response,kind=output,in=body,dataType=*sdk.WarmupScheduleResult" json:"-"`
}

func (*Output) JSONWireType() reflect.Type     { return reflect.TypeFor[sdk.WarmupScheduleResult]() }
func (o *Output) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"schedule,path=/v1/studio/sdk/versions.warmup_schedule_save,method=POST,connector=studio,handler=NewSchedule" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.warmup_schedule_save\",\"description\":\"Save an exact-version recurring warmup schedule\"}]" caseFormat:"lc" componentCallPolicy:"imperative"`
}

var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewSchedule" {
		return custom.Factory(NewSchedule)
	}
	return nil
}

type handler struct{}

func NewSchedule() xhandler.Contract[Input, Output] { return &handler{} }
func (*handler) Exec(ctx context.Context, session xhandler.Session, in *Input, out *Output) error {
	var result sdk.WarmupScheduleResult
	e := shared.Manage(ctx, session, in.Jwt, in.Auth, in.ReportID, in.VersionNo, sdk.OperationWarmupScheduleSave, map[string]any{"reportId": in.ReportID, "versionNo": in.VersionNo, "input": in.Input}, &result)
	if e != nil {
		return e
	}
	out.Response = &result
	return nil
}
