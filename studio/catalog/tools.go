package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespacevalidation"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type ToolListInput struct {
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	NamespaceID string             `parameter:"NamespaceID,kind=body,in=namespaceId,dataType=string,required=false" json:"namespaceId,omitempty"`
	Namespace   string             `parameter:"Namespace,kind=body,in=namespace,dataType=string,required=false" json:"namespace,omitempty"`
	Cursor      *string            `parameter:"Cursor,kind=body,in=cursor,dataType=*string,required=false" json:"cursor,omitempty"`
	Query       string             `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query,omitempty"`
}

type ToolCallInput struct {
	Jwt            *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth           *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	NamespaceID    string             `parameter:"NamespaceID,kind=body,in=namespaceId,dataType=string,required=true" json:"namespaceId"`
	Name           string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Arguments      map[string]any     `parameter:"Arguments,kind=body,in=arguments,dataType=map[string]interface{},required=false" json:"arguments"`
	TimeoutSeconds *int               `parameter:"TimeoutSeconds,kind=body,in=timeoutSeconds,dataType=*int,required=false" json:"timeoutSeconds,omitempty"`
}

type ToolListOutput struct {
	Response *schema.ListToolsResult `parameter:"Response,kind=output,in=body,dataType=*schema.ListToolsResult" json:"-"`
}

func (*ToolListOutput) JSONWireType() reflect.Type     { return reflect.TypeFor[schema.ListToolsResult]() }
func (o *ToolListOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type ToolCallOutput struct {
	Result json.RawMessage `parameter:"Result,kind=output,in=body,dataType=json.RawMessage" json:"result"`
}

type ToolListComponent struct {
	Contract xdatly.Component[ToolListInput, ToolListOutput] `component:"catalog_tools_list,path=/v1/studio/catalog/tools/list,method=POST,connector=studio,handler=NewToolList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.catalog.tools.list\",\"description\":\"Discover current authorized namespace tools and input schemas after publication, without reconnecting the client\"}]" caseFormat:"lc"`
}
type ToolCallComponent struct {
	Contract xdatly.Component[ToolCallInput, ToolCallOutput] `component:"catalog_tools_call,path=/v1/studio/catalog/tools/call,method=POST,connector=studio,handler=NewToolCall" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.catalog.tools.call\",\"description\":\"Invoke a published namespace tool using the caller credential and current runtime; inspect result.isError\"}]" caseFormat:"lc"`
}

var ToolListDatlyLinkedType = reflect.TypeFor[ToolListComponent]()
var ToolCallDatlyLinkedType = reflect.TypeFor[ToolCallComponent]()

func (ToolListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewToolList" {
		return custom.Factory(NewToolList)
	}
	return nil
}
func (ToolCallComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewToolCall" {
		return custom.Factory(NewToolCall)
	}
	return nil
}
func NewToolList() xhandler.Contract[ToolListInput, ToolListOutput] {
	return &handler[ToolListInput, ToolListOutput]{run: runToolList}
}
func NewToolCall() xhandler.Contract[ToolCallInput, ToolCallOutput] {
	return &handler[ToolCallInput, ToolCallOutput]{run: runToolCall}
}

func toolRuntime(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, id string) (*runtimeadmin.Client, string, error) {
	if namespaceaccess.ValidateResourceOwnership(id, "") != nil {
		return nil, "", fail(400, "canonical namespaceId is required")
	}
	row, err := visible(ctx, session, claims, auth, id)
	if err != nil {
		return nil, "", err
	}
	if !row.McpEnabled {
		return nil, "", fail(409, "namespace MCP must be enabled before discovering or invoking tools")
	}
	client, err := catalogRuntime(ctx)
	if err != nil {
		return nil, "", err
	}
	bearer, err := catalogBearer(ctx, session)
	return client, bearer, err
}
func runToolList(ctx context.Context, session xhandler.Session, input *ToolListInput, output *ToolListOutput) error {
	if input == nil || output == nil {
		return fail(400, "tool list input/output required")
	}
	namespaceID, err := selectedToolNamespace(ctx, session, input.Jwt, input.Auth, input.NamespaceID, input.Namespace)
	if err != nil {
		return err
	}
	client, bearer, err := toolRuntime(ctx, session, input.Jwt, input.Auth, namespaceID)
	if err != nil {
		return err
	}
	raw, err := client.ReadCatalog(sdk.WithNamespaceSelection(ctx, namespaceID), bearer, "tools/list", "", input.Cursor)
	if err != nil {
		return namespaceToolError(err)
	}
	var result schema.ListToolsResult
	if json.Unmarshal(raw, &result) != nil {
		return fail(503, "invalid namespace tool catalog")
	}
	if query := strings.ToLower(strings.TrimSpace(input.Query)); query != "" {
		filtered := []schema.Tool{}
		for _, tool := range result.Tools {
			if strings.Contains(strings.ToLower(tool.Name), query) {
				filtered = append(filtered, tool)
			}
		}
		result.Tools = filtered
	}
	output.Response = &result
	return nil
}

func selectedToolNamespace(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, id, name string) (string, error) {
	if id != "" {
		if name != "" {
			row, err := visible(ctx, session, claims, auth, id)
			if err != nil {
				return "", err
			}
			if row.Name != name {
				return "", fail(409, "namespace identity and name disagree")
			}
		}
		return id, nil
	}
	if !namespacevalidation.ValidName(name) {
		return "", fail(400, "canonical namespaceId or namespace name is required")
	}
	rows, err := directoryPage(ctx, session, claims, auth, "", name, "", false, 2, 0)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fail(404, "namespace not found or access denied")
	}
	if len(rows) != 1 || rows[0].Name != name {
		return "", fail(409, "namespace name is ambiguous")
	}
	return rows[0].NamespaceId, nil
}
func runToolCall(ctx context.Context, session xhandler.Session, input *ToolCallInput, output *ToolCallOutput) error {
	if input == nil || output == nil || strings.TrimSpace(input.Name) == "" {
		return fail(400, "tool name and input/output required")
	}
	timeout := 60 * time.Second
	if input.TimeoutSeconds != nil {
		if *input.TimeoutSeconds < 0 || int64(*input.TimeoutSeconds) > math.MaxInt64/int64(time.Second) {
			return fail(400, "timeoutSeconds must be nonnegative; 0 disables the Studio timeout")
		}
		timeout = time.Duration(*input.TimeoutSeconds) * time.Second
	}
	client, bearer, err := toolRuntime(ctx, session, input.Jwt, input.Auth, input.NamespaceID)
	if err != nil {
		return err
	}
	raw, err := client.CallTool(sdk.WithNamespaceSelection(ctx, input.NamespaceID), bearer, input.Name, input.Arguments, timeout)
	if err != nil {
		return namespaceToolError(err)
	}
	output.Result = raw
	return nil
}
func namespaceToolError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fail(504, "namespace tool deadline expired; increase timeoutSeconds or narrow the request")
	}
	if errors.Is(err, runtimeadmin.ErrNamespaceUnavailable) {
		return fail(503, "namespace MCP runtime is unavailable")
	}
	var rpc *runtimeadmin.CatalogError
	if errors.As(err, &rpc) {
		return fail(400, "namespace MCP rejected the tool request; check the advertised name, input schema and caller permissions")
	}
	return fail(503, "namespace tool transport is unavailable")
}
