package discover

import (
	"context"
	"errors"
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
	Jwt   *jwt.Claims                `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output         `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string                     `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.DatabaseDiscoveryInput `parameter:"Input,kind=body,in=input,dataType=sdk.DatabaseDiscoveryInput,required=false" json:"input"`
}

type Output struct {
	Driver   string                `parameter:"Driver,kind=output,in=body,dataType=string" json:"driver"`
	Product  string                `parameter:"Product,kind=output,in=body,dataType=string" json:"product"`
	Version  string                `parameter:"Version,kind=output,in=body,dataType=string" json:"version,omitempty"`
	Catalog  string                `parameter:"Catalog,kind=output,in=body,dataType=string" json:"catalog,omitempty"`
	Schema   string                `parameter:"Schema,kind=output,in=body,dataType=string" json:"schema,omitempty"`
	Catalogs []sdk.DatabaseCatalog `parameter:"Catalogs,kind=output,in=body,dataType=[]sdk.DatabaseCatalog" json:"catalogs"`
	Schemas  []sdk.DatabaseSchema  `parameter:"Schemas,kind=output,in=body,dataType=[]sdk.DatabaseSchema" json:"schemas"`
	Warnings []string              `parameter:"Warnings,kind=output,in=body,dataType=[]string" json:"warnings,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.discover,method=POST,connector=studio,handler=NewDiscover" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.discover\",\"description\":\"Discover database version, catalogs and schemas on an authorized active Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDiscover" {
		return custom.Factory(NewDiscover)
	}
	return nil
}

type handler struct{}

func NewDiscover() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector discovery input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	result, err := (connectivity.SQLCatalog{}).Discover(ctx, connector, input.Input)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return publisherguard.PublicError(504, "Database discovery timed out. Select a project/catalog and retry.")
		}
		return publisherguard.PublicError(502, "connector discovery is unavailable")
	}
	if result == nil {
		return fmt.Errorf("connector discovery returned no result")
	}
	*output = Output{Driver: result.Driver, Product: result.Product, Version: result.Version, Catalog: result.Catalog, Schema: result.Schema, Catalogs: result.Catalogs, Schemas: result.Schemas, Warnings: result.Warnings}
	return nil
}
