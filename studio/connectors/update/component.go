package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	status "github.com/viant/datly-studio/studio/connectors/disable"
	access "github.com/viant/datly-studio/studio/connectors/store_access"
	catalog "github.com/viant/datly-studio/studio/connectors/store_catalog"
	stored "github.com/viant/datly-studio/studio/connectors/store_config"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Options struct {
	Driver      *string          `json:"driver,omitempty"`
	DsnTemplate *string          `json:"dsnTemplate,omitempty"`
	SecretRef   *string          `json:"secretRef,omitempty"`
	Description *string          `json:"description,omitempty"`
	Options     *json.RawMessage `json:"options,omitempty"`
	ETag        int64            `json:"etag"`
}

type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

// Reuse the redacted connector DTO contract of activate/disable. Neither
// private connection material nor the catalog row is embedded in this output.
type Output = status.Output

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.update,method=POST,connector=studio,handler=NewConnectorUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.update\",\"description\":\"Update an authorized Datly Studio connector at an expected revision\"}]" caseFormat:"lc"`
}

var ConnectorDatly = new(Component)
var ConnectorDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewConnectorUpdate" {
		return custom.Factory(NewConnectorUpdate)
	}
	return nil
}

type updateHandler struct{}

func NewConnectorUpdate() xhandler.Contract[Input, Output] { return &updateHandler{} }

func (*updateHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if input.Name == "" || input.Input.ETag <= 0 {
		return publicError(400, "name and positive etag are required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("connector update handler session and output are required")
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
	read := &catalog.Input{}
	read.SetName(input.Name)
	read.SetQuery("")
	read.SetStatus("")
	read.SetOwnerId("")
	read.SetDriver("")
	read.SetSubject(input.Jwt.Subject)
	read.SetScoped(true)
	read.SetPageLimit(2)
	read.SetPageOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[catalog.ConnectorComponent](), "connector", "GET", "/_studio/connector-store/catalog"), Input: read})
	if err != nil {
		return err
	}
	listed, ok := value.(*catalog.Output)
	if !ok || listed == nil {
		return fmt.Errorf("connector catalog returned %T", value)
	}
	if len(listed.Connectors) == 0 {
		return publicError(404, "connector not found")
	}
	if len(listed.Connectors) != 1 || listed.Connectors[0] == nil || listed.Connectors[0].Name != input.Name {
		return fmt.Errorf("connector catalog returned ambiguous or mismatched rows")
	}
	current := listed.Connectors[0]
	if current.Etag != input.Input.ETag {
		return publicError(409, "connector etag does not match")
	}
	driver := current.Driver
	dsn, secret, description := stringValue(current.DsnTemplate), stringValue(current.SecretRef), stringValue(current.Description)
	options := append(json.RawMessage(nil), current.OptionsJson...)
	if input.Input.Driver != nil {
		driver = *input.Input.Driver
	}
	if input.Input.DsnTemplate != nil {
		dsn = *input.Input.DsnTemplate
	}
	if input.Input.SecretRef != nil {
		secret = *input.Input.SecretRef
	}
	if input.Input.Description != nil {
		description = *input.Input.Description
	}
	if input.Input.Options != nil {
		options = append(options[:0], (*input.Input.Options)...)
	}
	changedConnection := driver != current.Driver || dsn != stringValue(current.DsnTemplate) ||
		secret != stringValue(current.SecretRef) || !bytes.Equal(options, current.OptionsJson)
	now, etag := time.Now().UTC(), input.Input.ETag
	row := &stored.StoredConnector{Name: input.Name, Driver: driver, DsnTemplate: optional(dsn),
		SecretRef: optional(secret), Description: optional(description), OptionsJson: options,
		Etag: &etag, UpdatedAt: &now,
		Has: &stored.StoredConnectorHas{Name: true, Driver: true, DsnTemplate: true,
			SecretRef: true, Description: true, OptionsJson: true, Etag: true, UpdatedAt: true}}
	write := &stored.Input{}
	write.SetConnectors([]*stored.StoredConnector{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ConnectorComponent](), "connector", "PATCH", "/_studio/connector-store/config"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, "connector etag does not match")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].Name != input.Name || updated.Data[0].Etag == nil {
		return fmt.Errorf("connector config writer returned %T without one updated row", value)
	}
	statusValue := current.Status
	lastTestStatus, lastTestCode, lastTestedAt := stringValue(current.LastTestStatus), stringValue(current.LastTestErrorCode), current.LastTestedAt
	if changedConnection {
		statusValue, lastTestStatus, lastTestCode, lastTestedAt = "draft", "", "", nil
	}
	*output = Output{Name: input.Name, Driver: driver, DsnConfigured: strings.TrimSpace(dsn) != "",
		SecretConfigured: strings.TrimSpace(secret) != "", OwnerId: current.OwnerId,
		Status: statusValue, Options: append(json.RawMessage(nil), options...),
		LastTestStatus: lastTestStatus, LastTestErrorCode: lastTestCode, LastTestedAt: lastTestedAt,
		ETag: *updated.Data[0].Etag, CreatedAt: current.CreatedAt, UpdatedAt: now}
	output.Description = description
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
