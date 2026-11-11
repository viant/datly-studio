package table

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/providererror"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/connectivity"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	Jwt   *jwt.Claims          `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output   `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string               `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.TableDetailInput `parameter:"Input,kind=body,in=input,dataType=sdk.TableDetailInput,required=true" json:"input"`
}

type Output struct {
	Diagnostics      []sdk.Diagnostic     `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
	MetadataSource   string               `parameter:"MetadataSource,kind=output,in=body,dataType=string" json:"metadataSource,omitempty"`
	ConstraintsKnown bool                 `parameter:"ConstraintsKnown,kind=output,in=body,dataType=bool" json:"constraintsKnown"`
	Table            sdk.DatabaseTable    `parameter:"Table,kind=output,in=body,dataType=sdk.DatabaseTable" json:"table"`
	Columns          []sdk.DatabaseColumn `parameter:"Columns,kind=output,in=body,dataType=[]sdk.DatabaseColumn" json:"columns"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.table,method=POST,connector=studio,handler=NewTable" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.table\",\"description\":\"Inspect one table or view on an authorized active Studio connector\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTable" {
		return custom.Factory(NewTable)
	}
	return nil
}

type handler struct{}

func NewTable() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector table input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Input.Table) == "" {
		return publisherguard.PublicError(400, "table is required")
	}
	result, err := (connectivity.SQLCatalog{}).Table(ctx, connector, input.Input)
	if err != nil {
		return discoveryError(err)
	}
	if result == nil {
		return fmt.Errorf("connector table detail returned no result")
	}
	output.Table, output.Columns = result.Table, result.Columns
	output.MetadataSource, output.ConstraintsKnown = result.MetadataSource, result.ConstraintsKnown
	output.Diagnostics = result.Diagnostics
	return nil
}

func discoveryError(err error) error {
	if errors.Is(err, context.Canceled) {
		return publisherguard.PublicError(499, "Table discovery was canceled.", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return publisherguard.PublicError(504, "Table discovery exceeded its deadline.", err)
	}
	primary := err
	var recovery *connectivity.ColumnDiscoveryError
	if errors.As(err, &recovery) {
		primary = recovery.Primary
	}
	status := xresponse.ErrorStatusCode(primary, 502)
	if provider := providererror.FromError(primary); provider != nil {
		if provider.Provider == "Google API" {
			if code, e := strconv.Atoi(provider.Code); e == nil && code >= 400 && code <= 599 {
				status = code
			}
		}
		switch provider.Reason {
		case "accessDenied":
			status = 403
		case "notFound":
			status = 404
		case "invalidQuery":
			status = 400
		case "rateLimitExceeded", "jobRateLimitExceeded", "quotaExceeded":
			status = 429
		case "backendError", "internalError":
			status = 503
		}
	}
	message := providererror.AppendCatalogStatus("Primary table metadata discovery failed. Verify the exact source catalog and schema with the configured connector.", primary)
	if recovery != nil {
		message += " The zero-row recovery probe also failed; the original metadata failure is reported above."
	}
	result := publisherguard.PublicError(status, message, err).(*xresponse.Error)
	if recovery != nil {
		result.Payload = map[string]any{"message": message, "diagnostics": []sdk.Diagnostic{
			{Severity: "error", Stage: "catalog.columns", Code: "column_metadata_failed", Message: providererror.AppendCatalogStatus("Primary column metadata discovery failed.", primary), Hint: "Verify source metadata availability for the selected catalog and schema using the configured connector."},
			{Severity: "error", Stage: "catalog.probe", Code: "column_probe_failed", Message: providererror.AppendCatalogStatus("The zero-row recovery probe also failed.", recovery.Probe), Hint: "The original metadata failure determines this response status; inspect the probe's native failure separately."},
		}}
	}
	return result
}
