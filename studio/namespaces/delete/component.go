package delete

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	storedreader "github.com/viant/datly-studio/studio/namespaces/store_read"
	usage "github.com/viant/datly-studio/studio/namespaces/store_usage"
	storedwriter "github.com/viant/datly-studio/studio/namespaces/store_write"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	ETag int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}

type Output struct{}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"namespace,path=/v1/studio/sdk/namespaces.delete,method=POST,connector=studio,handler=NewNamespaceDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespaces.delete\",\"description\":\"Archive an unreferenced owned Datly Studio namespace\"}]" caseFormat:"lc"`
}

var NamespaceDatly = new(Component)
var NamespaceDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewNamespaceDelete" {
		return custom.Factory(NewNamespaceDelete)
	}
	return nil
}

type deleteHandler struct{}

func NewNamespaceDelete() xhandler.Contract[Input, Output] { return &deleteHandler{} }

func (*deleteHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio owner is required")
	}
	if input.Name == "" {
		return publicError(404, "namespace not found")
	}
	if session == nil || session.Binder() == nil || session.Response() == nil || output == nil {
		return fmt.Errorf("namespace delete handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	read := &storedreader.Input{}
	read.SetOwnerId(input.Jwt.Subject)
	read.SetName(input.Name)
	read.SetQuery("")
	read.SetStatus("")
	read.SetSubject("")
	read.SetScoped(false)
	read.SetPageLimit(2)
	read.SetPageOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedreader.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: read})
	if err != nil {
		return err
	}
	rows, ok := value.(*storedreader.Output)
	if !ok || rows == nil {
		return fmt.Errorf("owned namespace reader returned %T", value)
	}
	if len(rows.Namespaces) == 0 {
		return publicError(403, "Studio authorization denied")
	}
	if len(rows.Namespaces) != 1 || rows.Namespaces[0] == nil ||
		rows.Namespaces[0].OwnerId != input.Jwt.Subject || rows.Namespaces[0].Name != input.Name {
		return fmt.Errorf("owned namespace reader returned ambiguous or mismatched rows")
	}
	current := rows.Namespaces[0]
	if current.Etag != input.ETag {
		return publicError(409, "namespace etag does not match")
	}
	count := &usage.Input{}
	count.SetOwnerId(current.OwnerId)
	count.SetName(current.Name)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[usage.UsageComponent](), "usage", "GET", "/_studio/namespace-store/usage"), Input: count})
	if err != nil {
		return err
	}
	used, ok := value.(*usage.Output)
	if !ok || used == nil || len(used.Usages) != 1 || used.Usages[0] == nil {
		return fmt.Errorf("namespace usage reader returned %T without one total", value)
	}
	if used.Usages[0].Used > 0 {
		return publicError(409, "namespace is referenced by a component")
	}
	now, etag := time.Now().UTC(), input.ETag
	row := &storedwriter.StoredNamespace{OwnerId: current.OwnerId, Name: current.Name,
		Status: "archived", Etag: &etag, DeletedAt: &now, UpdatedAt: &now,
		Has: &storedwriter.StoredNamespaceHas{OwnerId: true, Name: true, Status: true,
			Etag: true, DeletedAt: true, UpdatedAt: true}}
	write := &storedwriter.Input{}
	write.SetNamespaces([]*storedwriter.StoredNamespace{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedwriter.NamespaceComponent](), "namespace", "PATCH", "/_studio/namespace-store/write"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, "namespace etag does not match")
		}
		return err
	}
	updated, ok := value.(*storedwriter.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].OwnerId != current.OwnerId || updated.Data[0].Name != current.Name ||
		updated.Data[0].Status != "archived" || updated.Data[0].DeletedAt == nil {
		return fmt.Errorf("namespace writer returned %T without one archived row", value)
	}
	session.Response().SetStatusCode(http.StatusNoContent)
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
