package delete

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/viant/datly-studio/internal/componentremoval"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
	"strings"
)

type Input struct {
	NamespaceId *string                  `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=true" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,required=true,errorCode=401" codec:"JwtClaim"`
	Auth        *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string                   `parameter:"ReportID,kind=body,in=id,dataType=string,required=true" json:"id"`
	Input       sdk.DeleteComponentInput `parameter:"Input,kind=body,in=input,dataType=sdk.DeleteComponentInput,required=true" json:"input"`
}
type Output struct {
	Response *sdk.DeletionResult `parameter:"Response,kind=output,in=body,dataType=*sdk.DeletionResult" json:"-"`
}

func (*Output) JSONWireType() reflect.Type     { return reflect.TypeFor[sdk.DeletionResult]() }
func (o *Output) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"delete,path=/v1/studio/sdk/components.delete,method=POST,connector=studio,handler=NewDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.components.delete\",\"description\":\"Permanently delete an unused owner-controlled component\"}]" componentCallPolicy:"imperative" caseFormat:"lc"`
}

var DeleteDatly = new(Component)
var DeleteDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDelete" {
		return custom.Factory(NewDelete)
	}
	return nil
}

type handler struct{}

func NewDelete() xhandler.Contract[Input, Output] { return &handler{} }
func (*handler) Exec(ctx context.Context, session xhandler.Session, in *Input, out *Output) error {
	if in == nil || out == nil || in.Jwt == nil || in.Auth == nil || in.Auth.Auth == nil || in.Jwt.Subject == "" || in.Jwt.Subject != in.Auth.Auth.Subject {
		return publisherguard.PublicError(403, "Verified Studio principal is required")
	}
	if in.NamespaceId == nil || len(*in.NamespaceId) != 64 || strings.TrimSpace(in.ReportID) == "" || in.Input.ETag < 1 {
		return publisherguard.PublicError(400, "An exact target, namespace, and positive concurrency token are required")
	}
	if raw, err := hex.DecodeString(*in.NamespaceId); err != nil || len(raw) != 32 || *in.NamespaceId != strings.ToLower(*in.NamespaceId) {
		return publisherguard.PublicError(400, "A valid namespace is required")
	}
	result, err := componentremoval.Execute(ctx, session, componentremoval.Request{NamespaceID: *in.NamespaceId, Actor: in.Jwt.Subject, ComponentID: in.ReportID, ETag: in.Input.ETag})
	if err != nil {
		return err
	}
	out.Response = result
	return nil
}
