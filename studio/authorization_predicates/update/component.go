package update

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/predicateprojection"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	get "github.com/viant/datly-studio/studio/authorization_predicates/get"
	stored "github.com/viant/datly-studio/studio/authorization_predicates/store_write"
	"github.com/viant/datly-studio/studio/host"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt   *jwt.Claims                           `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output                    `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string                                `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.UpdateAuthorizationPredicateInput `parameter:"Input,kind=body,in=input,dataType=sdk.UpdateAuthorizationPredicateInput,required=true" json:"input"`
}

type Component struct {
	Contract xdatly.Component[Input, get.Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.update,method=POST,connector=studio,handler=NewUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.update\",\"description\":\"Update an authorized predicate link with an exact ETag\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpdate" {
		return custom.Factory(NewUpdate)
	}
	return nil
}

type handler struct{}

func NewUpdate() xhandler.Contract[Input, get.Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *get.Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("authorization predicate update input and output are required")
	}
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Name) == "" || input.Input.ETag <= 0 || input.Input.ETag > math.MaxInt {
		return publisherguard.PublicError(400, "name and positive etag are required")
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
	current, err := predicateprojection.FromRow(rows[0], catalog)
	if err != nil {
		return err
	}
	if current.ETag != input.Input.ETag {
		return publisherguard.PublicError(409, "authorization predicate etag does not match")
	}
	if input.Input.Title != nil {
		current.Title = strings.TrimSpace(*input.Input.Title)
	}
	if input.Input.Description != nil {
		current.Description = strings.TrimSpace(*input.Input.Description)
	}
	if input.Input.PackagePath != nil {
		current.PackagePath = strings.TrimSpace(*input.Input.PackagePath)
	}
	if input.Input.TypeName != nil {
		current.TypeName = strings.TrimSpace(*input.Input.TypeName)
	}
	if input.Input.Status != nil {
		current.Status = strings.TrimSpace(*input.Input.Status)
	}
	if input.Input.Alias != nil {
		current.Alias = strings.TrimSpace(*input.Input.Alias)
	}
	if input.Input.Columns != nil {
		current.Columns = *input.Input.Columns
	}
	if current.Title == "" || current.Status != "active" && current.Status != "disabled" ||
		current.Status == "active" && !catalog.Contains(current.PackagePath, current.TypeName) {
		return publisherguard.PublicError(400, "active authorization predicate requires a title and configured package/type link")
	}
	scopeJSON, err := predicateprojection.ScopeJSON(current.Alias, current.Columns)
	if err != nil {
		return publisherguard.PublicError(400, "invalid authorization predicate SQL scope")
	}
	etag, now := int(input.Input.ETag), time.Now().UTC()
	row := &stored.StoredAuthorizationPredicate{Name: current.Name, Title: current.Title,
		Description: optional(current.Description), PackagePath: current.PackagePath,
		TypeName: current.TypeName, SqlScopeJson: optional(scopeJSON), Status: current.Status,
		Etag: &etag, UpdatedAt: &now,
		Has: &stored.StoredAuthorizationPredicateHas{Name: true, Title: true, Description: true,
			PackagePath: true, TypeName: true, SqlScopeJson: true, Status: true, Etag: true, UpdatedAt: true}}
	if err = predicateprojection.Write(ctx, invoker, row); err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publisherguard.PublicError(409, "authorization predicate etag does not match")
		}
		return err
	}
	current.ETag++
	current.UpdatedAt = now
	current.Linked = catalog.Contains(current.PackagePath, current.TypeName)
	output.Assign(current)
	return nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
