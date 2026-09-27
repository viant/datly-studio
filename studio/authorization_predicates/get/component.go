package get

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

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
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
}

type Output struct {
	Name        string    `parameter:"Name,kind=output,in=body,dataType=string" json:"name"`
	Title       string    `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description string    `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	PackagePath string    `parameter:"PackagePath,kind=output,in=body,dataType=string" json:"packagePath"`
	TypeName    string    `parameter:"TypeName,kind=output,in=body,dataType=string" json:"typeName"`
	Alias       string    `parameter:"Alias,kind=output,in=body,dataType=string" json:"alias,omitempty"`
	Columns     []string  `parameter:"Columns,kind=output,in=body,dataType=[]string" json:"columns,omitempty"`
	OwnerID     string    `parameter:"OwnerID,kind=output,in=body,dataType=string" json:"ownerId"`
	Status      string    `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	Linked      bool      `parameter:"Linked,kind=output,in=body,dataType=bool" json:"linked"`
	ETag        int64     `parameter:"ETag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt   time.Time `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt   time.Time `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}

func (output *Output) Assign(value *sdk.AuthorizationPredicate) {
	if output == nil || value == nil {
		return
	}
	*output = Output{Name: value.Name, Title: value.Title, Description: value.Description,
		PackagePath: value.PackagePath, TypeName: value.TypeName, Alias: value.Alias,
		Columns: value.Columns, OwnerID: value.OwnerID, Status: value.Status,
		Linked: value.Linked, ETag: value.ETag, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.get,method=POST,connector=studio,handler=NewGet" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.get\",\"description\":\"Read one governed authorization predicate\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewGet" {
		return custom.Factory(NewGet)
	}
	return nil
}

type handler struct{}

func NewGet() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("authorization predicate get input and output are required")
	}
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Name) == "" {
		return publisherguard.PublicError(400, "name is required")
	}
	rows, err := predicateprojection.Read(ctx, invoker, input.Name, "", "", 1, 0)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return publisherguard.PublicError(404, "authorization predicate not found")
	}
	if len(rows) != 1 || rows[0] == nil || rows[0].Name != input.Name {
		return fmt.Errorf("authorization predicate reader returned ambiguous identity")
	}
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		return err
	}
	value, err := predicateprojection.FromRow(rows[0], catalog)
	if err != nil {
		return err
	}
	output.Assign(value)
	return nil
}
