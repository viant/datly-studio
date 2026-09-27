package list

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/internal/predicateprojection"
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
	Jwt    *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth   *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Query  string             `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query,omitempty"`
	Status string             `parameter:"Status,kind=body,in=status,dataType=string,required=false" json:"status,omitempty"`
	Limit  int                `parameter:"Limit,kind=body,in=limit,dataType=int,required=false" json:"limit,omitempty"`
	Offset int                `parameter:"Offset,kind=body,in=offset,dataType=int,required=false" json:"offset,omitempty"`
}

type Output struct {
	Items  []*sdk.AuthorizationPredicate `parameter:"Items,kind=output,in=body,dataType=[]*sdk.AuthorizationPredicate" json:"items"`
	Limit  int                           `parameter:"Limit,kind=output,in=body,dataType=int" json:"limit"`
	Offset int                           `parameter:"Offset,kind=output,in=body,dataType=int" json:"offset"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.list,method=POST,connector=studio,handler=NewList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.list\",\"description\":\"List governed authorization predicates with bounded paging\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewList" {
		return custom.Factory(NewList)
	}
	return nil
}

type handler struct{}

func NewList() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("authorization predicate list input and output are required")
	}
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	limit, offset := input.Limit, input.Offset
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := predicateprojection.Read(ctx, invoker, "", input.Query, input.Status, limit, offset)
	if err != nil {
		return err
	}
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		return err
	}
	output.Items = make([]*sdk.AuthorizationPredicate, 0, len(rows))
	for _, row := range rows {
		item, projectionErr := predicateprojection.FromRow(row, catalog)
		if projectionErr != nil {
			return projectionErr
		}
		output.Items = append(output.Items, item)
	}
	output.Limit, output.Offset = limit, offset
	return nil
}
