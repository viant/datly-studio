package types

import (
	"context"
	"fmt"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	readsettings "github.com/viant/datly-studio/studio/namespace_settings/store_read"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"reflect"
	"strings"

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
	NamespaceId *string            `parameter:"NamespaceId,kind=body,in=namespaceId,dataType=*string" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
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
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	var policy *settings.Namespace
	if input.NamespaceId != nil {
		if strings.TrimSpace(*input.NamespaceId) == "" {
			return publisherguard.PublicError(400, "namespaceId must be nonempty")
		}
		var headers struct {
			Authorization string `bind:"kind=header,in=Authorization"`
		}
		if err = session.Binder().Bind(ctx, &headers); err != nil {
			return err
		}
		ctx = oauth.WithBearer(ctx, strings.TrimPrefix(headers.Authorization, "Bearer "))
		query := &namespaces.Input{}
		query.NamespaceId = *input.NamespaceId
		query.Has = &namespaces.InputHas{NamespaceId: true}
		query.SetSubject(input.Jwt.Subject)
		query.SetScoped(true)
		query.SetPageLimit(2)
		query.SetPageOffset(0)
		value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: settingsTarget(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "/_studio/namespace-store/read"), Input: query})
		if err != nil {
			return err
		}
		listed, ok := value.(*namespaces.Output)
		if !ok || listed == nil || len(listed.Namespaces) != 1 || listed.Namespaces[0].NamespaceId != *input.NamespaceId {
			return publisherguard.PublicError(403, "namespace access denied")
		}
		namespace := listed.Namespaces[0]
		read := &readsettings.Input{}
		read.SetOwnerId(namespace.OwnerId)
		read.SetNamespace(namespace.Name)
		value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: settingsTarget(reflect.TypeFor[readsettings.SettingsComponent](), "settings", "/_studio/namespace-settings/read"), Input: read})
		if err != nil {
			return err
		}
		stored, ok := value.(*readsettings.Output)
		if !ok || stored == nil {
			return fmt.Errorf("namespace settings reader returned %T", value)
		}
		policy = &settings.Namespace{}
		for _, row := range stored.Settings {
			if row.SettingKey == settings.SecurityKey {
				decoded, err := settings.Decode(row.SettingKey, row.ValueJson)
				if err != nil {
					return err
				}
				policy.Security = decoded.(*settings.Security)
			}
		}
	}
	authoring, err := host.AuthoringFromContext(ctx)
	if err != nil {
		return err
	}
	catalog := authoring.PredicateCatalog()
	output.Items = make([]*sdk.AuthorizationPredicateType, 0, len(catalog.Descriptors()))
	for _, descriptor := range catalog.Descriptors() {
		if !predicatepolicy.AllowsPackage(policy, descriptor.Package) {
			continue
		}
		output.Items = append(output.Items, &sdk.AuthorizationPredicateType{
			Alias: descriptor.Alias, PackagePath: descriptor.Package, TypeName: descriptor.TypeName, Description: descriptor.Description,
		})
	}
	return nil
}

func settingsTarget(holder reflect.Type, name, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: "GET", Path: path}}
}
