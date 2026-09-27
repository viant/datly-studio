package test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk/connectivity"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/connectors/store_status"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
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
	Name      string    `parameter:"Name,kind=output,in=body,dataType=string" json:"name"`
	Status    string    `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	ErrorCode string    `parameter:"ErrorCode,kind=output,in=body,dataType=string" json:"errorCode,omitempty"`
	Message   string    `parameter:"Message,kind=output,in=body,dataType=string" json:"message,omitempty"`
	TestedAt  time.Time `parameter:"TestedAt,kind=output,in=body,dataType=time.Time" json:"testedAt"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.test,method=POST,connector=studio,handler=NewTest" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.test\",\"description\":\"Probe an authorized Studio connector and persist bounded test evidence\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTest" {
		return custom.Factory(NewTest)
	}
	return nil
}

type handler struct{}

func NewTest() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector test input and output are required")
	}
	connector, err := connectorcatalog.ForProbe(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	result, probeErr := (connectivity.SQLProbe{}).Probe(ctx, connector)
	status, code, message := "passed", "", ""
	if probeErr != nil {
		status, code, message = "failed", "connectivity_failed", "connector connectivity check failed"
	} else if result != nil && result.Status != "" {
		status = result.Status
	}
	now := time.Now().UTC()
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("connector test handler session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	etag := connector.ETag
	row := &stored.StoredConnector{Name: connector.Name, Etag: &etag, UpdatedAt: &now,
		LastTestStatus: &status, LastTestErrorCode: optional(code), LastTestedAt: &now,
		Has: &stored.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true,
			LastTestStatus: true, LastTestErrorCode: true, LastTestedAt: true}}
	write := &stored.Input{}
	write.SetOperation("probe")
	write.SetConnectors([]*stored.StoredConnector{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[stored.ConnectorComponent]().PkgPath(), Name: "connector"},
		Route:     spec.RouteRef{Method: "PATCH", Path: "/_studio/connector-store/status"},
	}, Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publisherguard.PublicError(409, "connector changed during connectivity test")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil || updated.Data[0].Name != connector.Name {
		return fmt.Errorf("connector test writer returned %T without one matching row", value)
	}
	*output = Output{Name: connector.Name, Status: status, ErrorCode: code, Message: message, TestedAt: now}
	return nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
