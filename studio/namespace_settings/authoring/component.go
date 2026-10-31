// Package authoring exposes namespace settings through authenticated Datly components.
package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/authz/oauth"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespacevisibility"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	read "github.com/viant/datly-studio/studio/namespace_settings/store_read"
	write "github.com/viant/datly-studio/studio/namespace_settings/store_write"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly-studio/studio/settings"
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
	ETag     int64              `json:"etag"`
	Settings settings.Namespace `json:"settings"`
}
type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input *Options           `parameter:"Input,kind=body,in=input,dataType=*Options" json:"input,omitempty"`
}
type Output struct {
	Settings *settings.Namespace `parameter:"Settings,kind=output,in=body,dataType=*settings.Namespace" json:"settings"`
	ETag     int64               `parameter:"ETag,kind=output,in=body,dataType=int64" json:"etag"`
}
type GetComponent struct {
	Contract xdatly.Component[Input, Output] `component:"settings_get,path=/v1/studio/sdk/namespaces.settings_get,method=POST,connector=studio,handler=NewGet" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespaces.settings_get\",\"description\":\"Read namespace security and resource settings\"}]" caseFormat:"lc"`
}
type UpdateComponent struct {
	Contract xdatly.Component[Input, Output] `component:"settings_update,path=/v1/studio/sdk/namespaces.settings_update,method=POST,connector=studio,handler=NewUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespaces.settings_update\",\"description\":\"Update namespace settings at its expected ETag\"}]" caseFormat:"lc"`
}

var GetLinkedType = reflect.TypeFor[GetComponent]()
var UpdateLinkedType = reflect.TypeFor[UpdateComponent]()

func (GetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewGet" {
		return custom.Factory(NewGet)
	}
	return nil
}
func (UpdateComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpdate" {
		return custom.Factory(NewUpdate)
	}
	return nil
}
func NewGet() xhandler.Contract[Input, Output]    { return &handler{} }
func NewUpdate() xhandler.Contract[Input, Output] { return &handler{write: true} }

type handler struct{ write bool }

func public(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message)}
}
func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
func (h *handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil || input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return public(403, "verified Studio identity is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return public(400, "namespace name is required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("settings session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly invoker is unavailable")
	}
	var headers struct {
		Authorization string `bind:"kind=header,in=Authorization"`
	}
	if err = session.Binder().Bind(ctx, &headers); err != nil {
		return err
	}
	ctx = oauth.WithBearer(ctx, strings.TrimPrefix(headers.Authorization, "Bearer "))
	query := &namespaces.Input{}
	query.SetName(input.Name)
	query.SetSubject(input.Jwt.Subject)
	query.SetScoped(true)
	query.SetPageLimit(2)
	query.SetPageOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: query})
	if err != nil {
		return err
	}
	rows, ok := value.(*namespaces.Output)
	if !ok || rows == nil {
		return fmt.Errorf("namespace reader returned %T", value)
	}
	if len(rows.Namespaces) != 1 || rows.Namespaces[0] == nil {
		return public(403, "Studio namespace access denied")
	}
	namespace := rows.Namespaces[0]
	if !namespacevisibility.CanConfigure(ctx, input.Jwt.Subject, namespace.OwnerId, namespace.AllowedRolesJson) {
		return public(403, "namespace settings require owner or manager permission")
	}
	output.ETag = namespace.Etag
	querySettings := &read.Input{}
	querySettings.SetOwnerId(namespace.OwnerId)
	querySettings.SetNamespace(namespace.Name)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[read.SettingsComponent](), "settings", "GET", "/_studio/namespace-settings/read"), Input: querySettings})
	if err != nil {
		return err
	}
	listed, ok := value.(*read.Output)
	if !ok || listed == nil {
		return fmt.Errorf("settings reader returned %T", value)
	}
	output.Settings = &settings.Namespace{}
	for _, row := range listed.Settings {
		decoded, err := settings.Decode(row.SettingKey, row.ValueJson)
		if err != nil {
			return err
		}
		switch v := decoded.(type) {
		case *settings.Security:
			output.Settings.Security = v
		case *settings.ResourceLimits:
			output.Settings.ResourceLimits = v
		}
	}
	if h.write {
		if input.Input == nil || input.Input.ETag <= 0 {
			return public(400, "settings and positive etag are required")
		}
		if input.Input.ETag != namespace.Etag {
			return public(409, "namespace etag does not match")
		}
		authoring, err := host.AuthoringFromContext(ctx)
		if err != nil {
			return err
		}
		if err = authoring.ValidateSecuritySettings(input.Input.Settings.Security); err != nil {
			return public(400, err.Error())
		}
		now := time.Now().UTC()
		etag := input.Input.ETag
		row := &write.StoredNamespace{OwnerId: namespace.OwnerId, Name: namespace.Name, Etag: &etag, UpdatedAt: now, Has: &write.StoredNamespaceHas{OwnerId: true, Name: true, Etag: true, UpdatedAt: true, Entries: true}}
		for key, v := range map[string]any{settings.SecurityKey: input.Input.Settings.Security, settings.ResourceLimitsKey: input.Input.Settings.ResourceLimits} {
			data, err := json.Marshal(v)
			if err != nil {
				return err
			}
			if string(data) == "null" {
				continue
			}
			if _, err = settings.Decode(key, data); err != nil {
				return public(400, err.Error())
			}
			row.Entries = append(row.Entries, &write.StoredSetting{OwnerId: namespace.OwnerId, Namespace: namespace.Name, SettingKey: key, ValueJson: data, UpdatedAt: now, UpdatedBy: input.Jwt.Subject, Has: &write.StoredSettingHas{OwnerId: true, Namespace: true, SettingKey: true, ValueJson: true, UpdatedAt: true, UpdatedBy: true}})
		}
		if len(row.Entries) == 0 {
			return public(400, "supply at least one setting")
		}
		update := &write.Input{}
		update.SetNamespaces([]*write.StoredNamespace{row})
		_, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[write.SettingsComponent](), "settings", "PATCH", "/_studio/namespace-settings/write"), Input: update})
		if err != nil {
			return err
		}
		output.ETag++
		// Child writes remain buffered until the owning invocation commits. Shape the
		// response from the authorized snapshot and explicit patch, never a stale reread.
		if input.Input.Settings.Security != nil {
			output.Settings.Security = input.Input.Settings.Security
		}
		if input.Input.Settings.ResourceLimits != nil {
			output.Settings.ResourceLimits = input.Input.Settings.ResourceLimits
		}

	}

	return nil
}
