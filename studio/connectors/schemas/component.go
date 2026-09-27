package schemas

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
	Jwt   *jwt.Claims            `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output     `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string                 `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.SchemaCatalogInput `parameter:"Input,kind=body,in=input,dataType=sdk.SchemaCatalogInput,required=false" json:"input"`
}

type Output struct {
	Items []sdk.DatabaseSchema `parameter:"Items,kind=output,in=body,dataType=[]sdk.DatabaseSchema" json:"items"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.schemas,method=POST,connector=studio,handler=NewSchemas" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.schemas\",\"description\":\"List schemas on an authorized active Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewSchemas" {
		return custom.Factory(NewSchemas)
	}
	return nil
}

type handler struct{}

func NewSchemas() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector schemas input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	result, err := (connectivity.SQLCatalog{}).Schemas(ctx, connector, input.Input)
	if err != nil {
		return publisherguard.PublicError(502, "connector schema catalog is unavailable")
	}
	if result == nil {
		return fmt.Errorf("connector schema catalog returned no result")
	}
	output.Items = result.Items
	return nil
}
