package status

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/authorization"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
}

type Output struct {
	Response *sdk.RuntimeStatus `parameter:"Response,kind=output,in=body,dataType=*sdk.RuntimeStatus" json:"-"`
}

func (*Output) JSONWireType() reflect.Type { return reflect.TypeFor[sdk.RuntimeStatus]() }

func (output *Output) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"generation,path=/v1/studio/sdk/runtime.status,method=POST,connector=studio,handler=NewStatus" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.runtime.status\",\"description\":\"Inspect authorized Datly Studio runtime status\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var componentReachability = reflect.TypeFor[Component]()

func init() {}

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewStatus" {
		return custom.Factory(NewStatus)
	}
	return nil
}

type handler struct{}

func NewStatus() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("verified Studio principal is required"), Payload: map[string]string{"message": "verified Studio principal is required"}}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("runtime status handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if err != nil {
		return err
	}
	provider, ok := value.(xconnector.Provider)
	if !found || !ok {
		return fmt.Errorf("trusted Studio connector capability is unavailable")
	}
	db, err := provider.Connector(ctx, "studio")
	if err != nil || db == nil {
		return fmt.Errorf("configured Studio database is unavailable")
	}
	transport := &sqltransport.Transport{DB: db, Authorizer: &authorization.SDKAuthorizer{DB: db}}
	defer transport.Close(context.WithoutCancel(ctx))
	adminToken := os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN")
	if adminToken == "" {
		transport.RuntimeProbe = sqltransport.RuntimeHostProbeFunc(func(context.Context) (*sdk.RuntimeHost, error) {
			return nil, fmt.Errorf("dynamic runtime admin token is not configured")
		})
	} else {
		adminURL := os.Getenv("STUDIO_DYNAMIC_HTTP_URL")
		if adminURL == "" {
			adminURL = "http://127.0.0.1:8082"
		}
		admin, configErr := runtimeadmin.New(adminURL, adminToken, nil)
		if configErr != nil {
			return configErr
		}
		transport.RuntimeProbe = admin
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: input.Jwt.Subject})
	var response sdk.RuntimeStatus
	if err = transport.Invoke(principal, sdk.OperationRuntimeStatus, nil, &response); err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) {
			code := 500
			switch sdkErr.Code {
			case sdk.ErrorForbidden:
				code = 403
			case sdk.ErrorNotFound:
				code = 404
			case sdk.ErrorUnavailable:
				code = 503
			}
			return &xresponse.Error{Code: code, Cause: errors.New(sdkErr.Message), Payload: map[string]string{"message": sdkErr.Message}}
		}
		return err
	}
	output.Response = &response
	return nil
}
