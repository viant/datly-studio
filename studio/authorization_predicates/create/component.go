package create

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
	get "github.com/viant/datly-studio/studio/authorization_predicates/get"
	stored "github.com/viant/datly-studio/studio/authorization_predicates/store_insert"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name        string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Title       string             `parameter:"Title,kind=body,in=title,dataType=string,required=true" json:"title"`
	Description string             `parameter:"Description,kind=body,in=description,dataType=string,required=false" json:"description,omitempty"`
	PackagePath string             `parameter:"PackagePath,kind=body,in=packagePath,dataType=string,required=true" json:"packagePath"`
	TypeName    string             `parameter:"TypeName,kind=body,in=typeName,dataType=string,required=true" json:"typeName"`
	Alias       string             `parameter:"Alias,kind=body,in=alias,dataType=string,required=false" json:"alias,omitempty"`
	Columns     []string           `parameter:"Columns,kind=body,in=columns,dataType=[]string,required=false" json:"columns,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, get.Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.create,method=POST,connector=studio,handler=NewCreate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.create\",\"description\":\"Create a governed authorization predicate link\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewCreate" {
		return custom.Factory(NewCreate)
	}
	return nil
}

type handler struct{}

func NewCreate() xhandler.Contract[Input, get.Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *get.Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("authorization predicate create input and output are required")
	}
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	name, title := strings.TrimSpace(input.Name), strings.TrimSpace(input.Title)
	packagePath, typeName := strings.TrimSpace(input.PackagePath), strings.TrimSpace(input.TypeName)
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		return err
	}
	if !predicateprojection.ValidName(name) || title == "" || !catalog.Contains(packagePath, typeName) {
		return publisherguard.PublicError(400, "authorization predicate requires a canonical name and configured package/type link")
	}
	scopeJSON, err := predicateprojection.ScopeJSON(input.Alias, input.Columns)
	if err != nil {
		return publisherguard.PublicError(400, "invalid authorization predicate SQL scope")
	}
	existing, err := predicateprojection.Read(ctx, invoker, name, "", "", 1, 0)
	if err != nil {
		return err
	}
	if len(existing) != 0 {
		return publisherguard.PublicError(409, "authorization predicate already exists")
	}
	now := time.Now().UTC()
	row := &stored.StoredAuthorizationPredicate{Name: name, Title: title,
		Description: optional(strings.TrimSpace(input.Description)),
		PackagePath: packagePath, TypeName: typeName, SqlScopeJson: optional(scopeJSON),
		OwnerId: input.Jwt.Subject, Status: "active", Etag: 1, CreatedAt: now, UpdatedAt: now,
		Has: &stored.StoredAuthorizationPredicateHas{Name: true, Title: true, Description: true,
			PackagePath: true, TypeName: true, SqlScopeJson: true, OwnerId: true, Status: true,
			Etag: true, CreatedAt: true, UpdatedAt: true}}
	write := &stored.Input{}
	write.SetAuthorizationPredicates([]*stored.StoredAuthorizationPredicate{row})
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[stored.AuthorizationPredicateComponent]().PkgPath(), Name: "authorization_predicate"},
		Route:     spec.RouteRef{Method: "POST", Path: "/_studio/authorization-predicate-store/insert"},
	}, Input: write})
	if err != nil {
		if rows, readErr := predicateprojection.Read(ctx, invoker, name, "", "", 1, 0); readErr == nil && len(rows) != 0 {
			return publisherguard.PublicError(409, "authorization predicate already exists")
		}
		return err
	}
	inserted, ok := value.(*stored.Output)
	if !ok || inserted == nil || len(inserted.Data) != 1 || inserted.Data[0] == nil || inserted.Data[0].Name != name {
		return fmt.Errorf("authorization predicate insert returned %T without one matching row", value)
	}
	metadata := sdk.AuthorizationPredicateSQLMetadata{}
	if scopeJSON != "" {
		metadata.Alias = strings.TrimSpace(input.Alias)
		for _, column := range input.Columns {
			if column = strings.TrimSpace(column); column != "" {
				metadata.Columns = append(metadata.Columns, column)
			}
		}
	}
	output.Assign(&sdk.AuthorizationPredicate{Name: name, Title: title, Description: strings.TrimSpace(input.Description),
		PackagePath: packagePath, TypeName: typeName, Alias: metadata.Alias, Columns: metadata.Columns,
		OwnerID: input.Jwt.Subject, Status: "active", Linked: true, ETag: 1, CreatedAt: now, UpdatedAt: now})
	return nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
