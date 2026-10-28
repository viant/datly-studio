// Package catalog exposes permission-scoped namespace discovery and virtual MCP definitions.
package catalog

import (
	"context"
	"encoding/json"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	primitive "github.com/viant/forge/backend/provider/contract"
	"github.com/viant/forge/backend/types"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

type Namespace = namespaceext.Namespace
type Reference = namespaceext.Resource

type ListOutput struct {
	Response *namespaceext.ListResult `parameter:"Response,kind=output,in=body,dataType=*namespaceext.ListResult" json:"-"`
}

func (*ListOutput) JSONWireType() reflect.Type { return reflect.TypeFor[namespaceListWire]() }
func (o *ListOutput) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	return marshalNamespaceList(o.Response)
}

type GetOutput struct {
	ProviderIdentity string                         `json:"-"`
	Sync             *primitive.NamespaceSyncResult `json:"-"`
	Response         *namespaceext.GetResult        `parameter:"Response,kind=output,in=body,dataType=*namespaceext.GetResult" json:"-"`
}

func (*GetOutput) JSONWireType() reflect.Type { return reflect.TypeFor[namespaceGetWire]() }
func (o *GetOutput) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	if o.Sync != nil {
		return json.Marshal(namespaceGetWire{GetResult: o.Response, Sync: o.Sync, ProviderIdentity: o.ProviderIdentity})
	}
	return json.Marshal(o.Response)
}

type ResourceOutput struct {
	Response json.RawMessage `parameter:"Response,kind=output,in=body,dataType=json.RawMessage" json:"-"`
}

func (*ResourceOutput) JSONWireType() reflect.Type { return reflect.TypeFor[map[string]any]() }
func (o *ResourceOutput) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.Response)
}

type handler[I, O any] struct {
	run func(context.Context, xhandler.Session, *I, *O) error
}

func (h *handler[I, O]) Exec(ctx context.Context, session xhandler.Session, input *I, output *O) error {
	return h.run(ctx, session, input, output)
}

type ListInput struct {
	Jwt     *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request namespaceext.ListRequest `parameter:"Request,kind=body,in=,dataType=namespaceext.ListRequest,required=false" anonymous:"true"`
}
type ListComponent struct {
	Contract xdatly.Component[ListInput, ListOutput] `component:"catalog_list,path=/v1/studio/catalog/namespaces/list,method=POST,connector=studio,handler=NewList" mcp:"[{\"kind\":\"tool\",\"name\":\"namespaces/list\",\"description\":\"List visible namespaces with bounded pagination\"}]" caseFormat:"lc"`
}

var ListDatly = new(ListComponent)
var ListDatlyLinkedType = reflect.TypeFor[ListComponent]()

func (ListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewList" {
		return custom.Factory(NewList)
	}
	return nil
}
func NewList() xhandler.Contract[ListInput, ListOutput] {
	return &handler[ListInput, ListOutput]{run: runList}
}

// NamespaceGetRequest keeps ordinary inventory selectors while allowing the
// explicit synchronization operation to select its logical namespace alias.
// An ordinary get still requires URI; runGet enforces that distinction.
type NamespaceGetRequest struct {
	URI    string  `json:"uri,omitempty"`
	Kind   string  `json:"kind,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

type GetInput struct {
	Namespace string                          `parameter:"Namespace,kind=body,in=namespace" json:"namespace,omitempty"`
	Sync      *primitive.NamespaceSyncRequest `parameter:"Sync,kind=body,in=sync" json:"sync,omitempty"`
	Jwt       *jwt.Claims                     `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output              `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request   NamespaceGetRequest             `parameter:"Request,kind=body,in=,dataType=NamespaceGetRequest,required=false" anonymous:"true"`
}
type GetComponent struct {
	Contract xdatly.Component[GetInput, GetOutput] `component:"catalog_get,path=/v1/studio/catalog/namespaces/get,method=POST,connector=studio,handler=NewGet" mcp:"[{\"kind\":\"tool\",\"name\":\"namespaces/get\",\"description\":\"List accessible resources in a namespace with pagination\"}]" caseFormat:"lc"`
}

var GetDatly = new(GetComponent)
var GetDatlyLinkedType = reflect.TypeFor[GetComponent]()

func (GetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewGet" {
		return custom.Factory(NewGet)
	}
	return nil
}
func NewGet() xhandler.Contract[GetInput, GetOutput] {
	return &handler[GetInput, GetOutput]{run: runGet}
}

type ResourceInput struct {
	Target  *types.WindowTarget             `parameter:"Target,kind=body,in=target" json:"target,omitempty"`
	Jwt     *jwt.Claims                     `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output              `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request namespaceext.ResourceGetRequest `parameter:"Request,kind=body,in=,dataType=namespaceext.ResourceGetRequest,required=true" anonymous:"true"`
}
type ResourceComponent struct {
	Contract xdatly.Component[ResourceInput, ResourceOutput] `component:"catalog_resource,path=/v1/studio/catalog/resource/get,method=POST,connector=studio,handler=NewResource" mcp:"[{\"kind\":\"tool\",\"name\":\"resource/get\",\"description\":\"Resolve an accessible virtual resource to its JSON definition\"}]" caseFormat:"lc"`
}

var ResourceDatly = new(ResourceComponent)
var ResourceDatlyLinkedType = reflect.TypeFor[ResourceComponent]()

func (ResourceComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewResource" {
		return custom.Factory(NewResource)
	}
	return nil
}
func NewResource() xhandler.Contract[ResourceInput, ResourceOutput] {
	return &handler[ResourceInput, ResourceOutput]{run: runResource}
}
