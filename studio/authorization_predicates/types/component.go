package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
}

type Output struct {
	Items []*sdk.AuthorizationPredicateType `parameter:"Items,kind=output,in=body,dataType=[]*sdk.AuthorizationPredicateType" json:"items"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.types,method=POST,connector=studio,handler=NewTypes" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.types\",\"description\":\"List predicate types available to an authorized Studio publisher\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTypes" {
		return custom.Factory(NewTypes)
	}
	return nil
}

type handler struct{}

func NewTypes() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("predicate types handler session and output are required")
	}
	_, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		return err
	}
	output.Items = make([]*sdk.AuthorizationPredicateType, 0, len(catalog.Descriptors()))
	for _, descriptor := range catalog.Descriptors() {
		output.Items = append(output.Items, &sdk.AuthorizationPredicateType{
			Alias: descriptor.Alias, PackagePath: descriptor.Package, TypeName: descriptor.TypeName,
		})
	}
	return nil
}
