package delete

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	status "github.com/viant/datly-studio/studio/connectors/disable"
	access "github.com/viant/datly-studio/studio/connectors/store_access"
	stored "github.com/viant/datly-studio/studio/connectors/store_status"
	usage "github.com/viant/datly-studio/studio/connectors/store_usage"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input = status.Input
type Output struct{}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.delete,method=POST,connector=studio,handler=NewConnectorDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.delete\",\"description\":\"Archive an unreferenced authorized Datly Studio connector\"}]" caseFormat:"lc"`
}

var ConnectorDatly = new(Component)
var ConnectorDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewConnectorDelete" {
		return custom.Factory(NewConnectorDelete)
	}
	return nil
}

type deleteHandler struct{}

func NewConnectorDelete() xhandler.Contract[Input, Output] { return &deleteHandler{} }

func (*deleteHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if input.Name == "" {
		return publicError(400, "connector name is required")
	}
	if session == nil || session.Binder() == nil || session.Response() == nil || output == nil {
		return fmt.Errorf("connector delete handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	guard := &access.Input{}
	guard.SetName(input.Name)
	guard.SetSubject(input.Jwt.Subject)
	guard.SetPermission("edit")
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[access.ConnectorComponent](), "connector", "GET", "/_studio/connector-store/access"), Input: guard})
	if err != nil {
		return err
	}
	allowed, ok := value.(*access.Output)
	if !ok || allowed == nil {
		return fmt.Errorf("connector edit guard returned %T", value)
	}
	if len(allowed.Connectors) == 0 {
		return publicError(403, "Studio authorization denied")
	}
	if len(allowed.Connectors) != 1 || allowed.Connectors[0] == nil || allowed.Connectors[0].Name != input.Name {
		return fmt.Errorf("connector edit guard returned ambiguous or mismatched rows")
	}
	count := &usage.Input{}
	count.SetName(input.Name)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[usage.UsageComponent](), "usage", "GET", "/_studio/connector-store/usage"), Input: count})
	if err != nil {
		return err
	}
	used, ok := value.(*usage.Output)
	if !ok || used == nil || len(used.Usages) != 1 || used.Usages[0] == nil || used.Usages[0].Used < 0 {
		return fmt.Errorf("connector usage reader returned %T without one valid total", value)
	}
	if used.Usages[0].Used > 0 {
		return publicError(409, "connector is referenced by a report")
	}
	now, etag := time.Now().UTC(), input.ETag
	row := &stored.StoredConnector{Name: input.Name, Etag: &etag, UpdatedAt: &now, DeletedAt: &now,
		Has: &stored.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true, DeletedAt: true}}
	write := &stored.Input{}
	write.SetOperation("delete")
	write.SetConnectors([]*stored.StoredConnector{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ConnectorComponent](), "connector", "PATCH", "/_studio/connector-store/status"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(404, "connector not found")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].Name != input.Name || updated.Data[0].Status != "deleted" || updated.Data[0].DeletedAt == nil {
		return fmt.Errorf("connector status writer returned %T without one deleted row", value)
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
