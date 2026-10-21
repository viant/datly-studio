// Package auditcontrol binds native MCP/HTTP audit controls to host infrastructure.
package auditcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/audit"
	"github.com/viant/datly-studio/internal/publisherguard"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
	"strings"
)

type Input struct {
	MaxRows     int                `parameter:"MaxRows,kind=body,in=maxRows,dataType=int,required=false" json:"maxRows,omitempty"`
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ComponentID string             `parameter:"ComponentID,kind=body,in=componentId,dataType=string,required=false" json:"componentId,omitempty"`
	ToolName    string             `parameter:"ToolName,kind=body,in=toolName,dataType=string,required=false" json:"toolName,omitempty"`
	Enabled     *bool              `parameter:"Enabled,kind=body,in=enabled,dataType=*bool,required=false" json:"enabled,omitempty"`
	Filter      audit.Filter       `parameter:"Filter,kind=body,in=filter,dataType=audit.Filter,required=false" json:"filter,omitempty"`
}
type Response struct {
	Filename  string `json:"filename,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Archive   []byte `json:"archive,omitempty"`
	Count     int    `json:"count,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	audit.Setting
	Items   []audit.Event `json:"items,omitempty"`
	NextID  int64         `json:"nextId,omitempty"`
	Dropped uint64        `json:"dropped"`
	Failed  uint64        `json:"failed"`
}
type Output struct {
	Response *Response `parameter:"Response,kind=output,in=body,dataType=*Response" json:"-"`
}

func (*Output) JSONWireType() reflect.Type     { return reflect.TypeFor[Response]() }
func (o *Output) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type Handler struct{ Operation string }

func (h *Handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil || input.Jwt.Subject == "" || input.Jwt.Subject != input.Auth.Auth.Subject {
		return publisherguard.PublicError(403, "Verified Studio identity is required")
	}
	service := audit.FromContext(ctx)
	if service == nil {
		return publisherguard.PublicError(503, "Audit service is unavailable")
	}
	var headers struct {
		Authorization string `bind:"kind=header,in=Authorization"`
	}
	if err := session.Binder().Bind(ctx, &headers); err != nil {
		return err
	}
	ctx = oauth.WithBearer(ctx, strings.TrimSpace(strings.TrimPrefix(headers.Authorization, "Bearer ")))
	scope := audit.Scope{ComponentID: input.ComponentID, ToolName: input.ToolName}
	if input.NamespaceId != nil {
		scope.NamespaceID = *input.NamespaceId
	}
	var err error
	result := &Response{}
	switch h.Operation {
	case "status":
		result.Setting, err = service.Status(ctx, scope)
	case "configure":
		if input.Enabled == nil {
			return publisherguard.PublicError(400, "enabled is required")
		}
		result.Setting, err = service.Configure(ctx, scope, *input.Enabled)
	case "export":
		input.Filter.NamespaceID = scope.NamespaceID
		var exported *audit.Export
		exported, err = service.ExportCSV(ctx, input.Filter, input.MaxRows)
		if err == nil {
			result.Filename = exported.Filename
			result.MediaType = exported.MediaType
			result.Archive = exported.Archive
			result.Count = exported.Count
			result.Truncated = exported.Truncated
			result.NextID = exported.NextID
		}
	case "list":
		input.Filter.NamespaceID = scope.NamespaceID
		var page audit.Page
		page, err = service.List(ctx, input.Filter)
		result.Items = page.Items
		result.NextID = page.NextID
		result.Dropped = page.Dropped
		result.Failed = page.Failed
	}
	if errors.Is(err, audit.ErrForbidden) {
		return publisherguard.PublicError(403, "Audit permission is required")
	}
	if errors.Is(err, audit.ErrInvalid) {
		return publisherguard.PublicError(400, "Invalid audit scope or filter")
	}
	if err != nil {
		return publisherguard.PublicError(503, "Audit operation is unavailable")
	}
	output.Response = result
	return nil
}
