package table

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/connectivity"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt   *jwt.Claims          `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output   `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string               `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.TableDetailInput `parameter:"Input,kind=body,in=input,dataType=sdk.TableDetailInput,required=true" json:"input"`
}

type Output struct {
	Table   sdk.DatabaseTable    `parameter:"Table,kind=output,in=body,dataType=sdk.DatabaseTable" json:"table"`
	Columns []sdk.DatabaseColumn `parameter:"Columns,kind=output,in=body,dataType=[]sdk.DatabaseColumn" json:"columns"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.table,method=POST,connector=studio,handler=NewTable" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.table\",\"description\":\"Inspect one table or view on an authorized active Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTable" {
		return custom.Factory(NewTable)
	}
	return nil
}

type handler struct{}

func NewTable() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector table input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Input.Table) == "" {
		return publisherguard.PublicError(400, "table is required")
	}
	result, err := (connectivity.SQLCatalog{}).Table(ctx, connector, input.Input)
	if err != nil {
		return publisherguard.PublicError(502, "connector table detail is unavailable")
	}
	if result == nil {
		return fmt.Errorf("connector table detail returned no result")
	}
	output.Table, output.Columns = result.Table, result.Columns
	return nil
}
