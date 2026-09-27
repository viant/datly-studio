package test_sql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

const modulePath = "github.com/viant/datly-studio"
const maxExecution = 15 * time.Second

type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.SQLTestInput   `parameter:"Input,kind=body,in=input,dataType=sdk.SQLTestInput,required=true" json:"input"`
}

type Output struct {
	Data     json.RawMessage `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Duration time.Duration   `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
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
	executionCtx, cancel := context.WithTimeout(ctx, maxExecution)
	defer cancel()
	result, err := (preview.Dynamic{ModulePath: modulePath}).TestSQL(executionCtx, connector, input.Input)
	if err != nil {
		return publisherguard.PublicError(400, "SQL test could not be compiled or executed")
	}
	if result == nil {
		return fmt.Errorf("transient SQL test returned no result")
	}
	output.Data, output.Duration = result.Data, result.Duration
	return nil
}
