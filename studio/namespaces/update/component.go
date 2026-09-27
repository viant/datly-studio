package update

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	storedreader "github.com/viant/datly-studio/studio/namespaces/store_read"
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

type Options struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	ETag        int64   `json:"etag"`
}

type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

type Output struct {
	OwnerId     string    `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	Name        string    `parameter:"Name,kind=output,in=body,dataType=string" json:"name"`
	Title       string    `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description string    `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	Status      string    `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	ETag        int64     `parameter:"ETag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt   time.Time `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt   time.Time `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"namespace,path=/v1/studio/sdk/namespaces.update,method=POST,connector=studio,handler=NewNamespaceUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespaces.update\",\"description\":\"Update an owned Datly Studio namespace at an expected revision\"}]" caseFormat:"lc"`
}

var NamespaceDatly = new(Component)
var NamespaceDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewNamespaceUpdate" {
		return custom.Factory(NewNamespaceUpdate)
	}
	return nil
}

type updateHandler struct{}

func NewNamespaceUpdate() xhandler.Contract[Input, Output] { return &updateHandler{} }

func (*updateHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("verified Studio owner is required")}
	}
	if input.Name == "" || input.Input.ETag <= 0 {
		return &xresponse.Error{Code: 400, Cause: errors.New("name and positive etag are required")}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("namespace update handler session and output are required")
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
	result, ok := value.(*storedreader.Output)
	if !ok || result == nil {
		return fmt.Errorf("owned namespace reader returned %T", value)
	}
	if len(result.Namespaces) == 0 {
		return &xresponse.Error{Code: 403, Cause: errors.New("Studio authorization denied")}
	}
	if len(result.Namespaces) != 1 || result.Namespaces[0] == nil ||
		result.Namespaces[0].OwnerId != input.Jwt.Subject || result.Namespaces[0].Name != input.Name {
		return fmt.Errorf("owned namespace reader returned ambiguous or mismatched rows")
	}
	current := result.Namespaces[0]
	if current.Etag != input.Input.ETag {
		return &xresponse.Error{Code: 409, Cause: errors.New("namespace etag does not match")}
	}
	title, status := current.Title, current.Status
	description := current.Description
	if input.Input.Title != nil {
		title = strings.TrimSpace(*input.Input.Title)
	}
	if input.Input.Description != nil {
		clean := strings.TrimSpace(*input.Input.Description)
		description = nil
		if clean != "" {
			description = &clean
		}
	}
	if input.Input.Status != nil {
		status = strings.TrimSpace(*input.Input.Status)
	}
	if title == "" || status != "active" && status != "archived" {
		return &xresponse.Error{Code: 400, Cause: errors.New("namespace title and active or archived status are required")}
	}
	now := time.Now().UTC()
	etag := input.Input.ETag
	row := &storedwriter.StoredNamespace{OwnerId: current.OwnerId, Name: current.Name,
		Title: title, Description: description, Status: status, Etag: &etag, UpdatedAt: &now,
		Has: &storedwriter.StoredNamespaceHas{OwnerId: true, Name: true, Title: true,
			Description: true, Status: true, Etag: true, UpdatedAt: true}}
	write := &storedwriter.Input{}
	write.SetNamespaces([]*storedwriter.StoredNamespace{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedwriter.NamespaceComponent](), "namespace", "PATCH", "/_studio/namespace-store/write"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &xresponse.Error{Code: 409, Cause: errors.New("namespace etag does not match")}
		}
		return err
	}
	updated, ok := value.(*storedwriter.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].OwnerId != current.OwnerId || updated.Data[0].Name != current.Name || updated.Data[0].Etag == nil {
		return fmt.Errorf("namespace writer returned %T without one updated row", value)
	}
	*output = Output{OwnerId: current.OwnerId, Name: current.Name, Title: title,
		Status: status, ETag: *updated.Data[0].Etag, CreatedAt: current.CreatedAt,
		UpdatedAt: now}
	if description != nil {
		output.Description = *description
	}
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
