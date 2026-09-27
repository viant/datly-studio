package disable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/connectors/accesspredicate"
	connectors "github.com/viant/datly-studio/studio/connectors/get"
	access "github.com/viant/datly-studio/studio/connectors/store_access"
	stored "github.com/viant/datly-studio/studio/connectors/store_status"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// Keep the predicate linked for the private access reader selected with this
// workflow. The caller never supplies its subject or permission fields.
var ConnectorAccessDatlyType = reflect.TypeFor[accesspredicate.ConnectorAccess]()

type Input struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	ETag int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}

type Output struct {
	Name              string          `parameter:"Name,kind=output,in=body,dataType=string" json:"name"`
	Driver            string          `parameter:"Driver,kind=output,in=body,dataType=string" json:"driver"`
	DsnConfigured     bool            `parameter:"DsnConfigured,kind=output,in=body,dataType=bool" json:"dsnConfigured"`
	SecretConfigured  bool            `parameter:"SecretConfigured,kind=output,in=body,dataType=bool" json:"secretConfigured"`
	Description       string          `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	OwnerId           string          `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	Status            string          `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	Options           json.RawMessage `parameter:"Options,kind=output,in=body,dataType=json.RawMessage" json:"options,omitempty"`
	LastTestStatus    string          `parameter:"LastTestStatus,kind=output,in=body,dataType=string" json:"lastTestStatus,omitempty"`
	LastTestErrorCode string          `parameter:"LastTestErrorCode,kind=output,in=body,dataType=string" json:"lastTestErrorCode,omitempty"`
	LastTestedAt      *time.Time      `parameter:"LastTestedAt,kind=output,in=body,dataType=*time.Time" json:"lastTestedAt,omitempty"`
	ETag              int64           `parameter:"ETag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt         time.Time       `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt         time.Time       `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.disable,method=POST,connector=studio,handler=NewConnectorDisable" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.disable\",\"description\":\"Disable an authorized Datly Studio connector at an expected revision\"}]" caseFormat:"lc"`
}

var ConnectorDatly = new(Component)
var ConnectorDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewConnectorDisable" {
		return custom.Factory(NewConnectorDisable)
	}
	return nil
}

type statusHandler struct{ operation string }

func NewConnectorDisable() xhandler.Contract[Input, Output] {
	return &statusHandler{operation: "disable"}
}
func NewConnectorActivate() xhandler.Contract[Input, Output] {
	return &statusHandler{operation: "activate"}
}

func (handler *statusHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if input.Name == "" || input.ETag <= 0 {
		return publicError(400, "name and positive etag are required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("connector status handler session and output are required")
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
	accessResult, ok := value.(*access.Output)
	if !ok || accessResult == nil {
		return fmt.Errorf("connector edit guard returned %T", value)
	}
	if len(accessResult.Connectors) == 0 {
		return publicError(403, "Studio authorization denied")
	}
	if len(accessResult.Connectors) != 1 || accessResult.Connectors[0] == nil || accessResult.Connectors[0].Name != input.Name {
		return fmt.Errorf("connector edit guard returned ambiguous or mismatched rows")
	}
	read := &connectors.ConnectorGetInput{}
	read.SetJwt(input.Jwt)
	read.SetAuth(input.Auth)
	read.SetName(input.Name)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[connectors.ConnectorComponent](), "connector", "POST", "/v1/studio/sdk/connectors.get"), Input: read})
	if err != nil {
		return err
	}
	result, ok := value.(*connectors.ConnectorGetOutput)
	if !ok || result == nil {
		return fmt.Errorf("connector reader returned %T", value)
	}
	if result.Item == nil || result.Item.Name != input.Name || result.Item.OwnerId == "" {
		return fmt.Errorf("connector reader returned missing or mismatched row")
	}
	current := result.Item
	if current.Etag != input.ETag {
		return publicError(409, "connector etag does not match")
	}
	if handler.operation == "activate" && (current.LastTestStatus == nil || *current.LastTestStatus != "passed") {
		return publicError(400, "connector must pass a connectivity test before activation")
	}
	if handler.operation != "activate" && handler.operation != "disable" {
		return fmt.Errorf("unsupported connector status operation %q", handler.operation)
	}
	expectedStatus := "disabled"
	if handler.operation == "activate" {
		expectedStatus = "active"
	}
	now, etag := time.Now().UTC(), input.ETag
	row := &stored.StoredConnector{Name: input.Name, Etag: &etag, UpdatedAt: &now,
		Has: &stored.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true}}
	write := &stored.Input{}
	write.SetOperation(handler.operation)
	write.SetConnectors([]*stored.StoredConnector{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ConnectorComponent](), "connector", "PATCH", "/_studio/connector-store/status"), Input: write})
	if err != nil {
		if errors.Is(err, stored.ErrProbeRequired) {
			return publicError(400, "connector must pass a connectivity test before activation")
		}
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, "connector etag does not match")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].Name != input.Name || updated.Data[0].Status != expectedStatus || updated.Data[0].Etag == nil {
		return fmt.Errorf("connector status writer returned %T without one %s row", value, handler.operation)
	}
	*output = Output{Name: current.Name, Driver: current.Driver, DsnConfigured: current.DsnConfigured,
		SecretConfigured: current.SecretConfigured, OwnerId: current.OwnerId, Status: updated.Data[0].Status,
		Options: append(json.RawMessage(nil), current.OptionsJson...), ETag: *updated.Data[0].Etag,
		CreatedAt: current.CreatedAt, UpdatedAt: now, LastTestedAt: current.LastTestedAt}
	if current.Description != nil {
		output.Description = *current.Description
	}
	if current.LastTestStatus != nil {
		output.LastTestStatus = *current.LastTestStatus
	}
	if current.LastTestErrorCode != nil {
		output.LastTestErrorCode = *current.LastTestErrorCode
	}
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
