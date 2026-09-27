package tables

import (
	"context"
	"fmt"
	"reflect"

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
	Jwt   *jwt.Claims           `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output    `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string                `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.TableCatalogInput `parameter:"Input,kind=body,in=input,dataType=sdk.TableCatalogInput,required=false" json:"input"`
}

type Output struct {
	Items  []sdk.DatabaseTable `parameter:"Items,kind=output,in=body,dataType=[]sdk.DatabaseTable" json:"items"`
	Limit  int                 `parameter:"Limit,kind=output,in=body,dataType=int" json:"limit"`
	Offset int                 `parameter:"Offset,kind=output,in=body,dataType=int" json:"offset"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.tables,method=POST,connector=studio,handler=NewTables" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.tables\",\"description\":\"List tables and views on an authorized active Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTables" {
		return custom.Factory(NewTables)
	}
	return nil
}

type handler struct{}

func NewTables() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector tables input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	result, err := (connectivity.SQLCatalog{}).Tables(ctx, connector, input.Input)
	if err != nil {
		return publisherguard.PublicError(502, "connector table catalog is unavailable")
	}
	if result == nil {
		return fmt.Errorf("connector table catalog returned no result")
	}
	output.Items, output.Limit, output.Offset = result.Items, result.Limit, result.Offset
	return nil
}
