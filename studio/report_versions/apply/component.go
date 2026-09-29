package apply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/readerinspection"
	"github.com/viant/datly-studio/internal/versionidentity"
	"github.com/viant/datly-studio/internal/versionprojection"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	catalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	stored "github.com/viant/datly-studio/studio/report_versions/store_edit"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	reports "github.com/viant/datly-studio/studio/reports/get"
	capabilities "github.com/viant/datly-studio/studio/reports/store_capabilities"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Command struct {
	Kind                   string          `json:"kind"`
	ExpectedSourceRevision int64           `json:"expectedSourceRevision"`
	Payload                json.RawMessage `json:"payload"`
}

type Input struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId    string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Command     Command            `parameter:"Command,kind=body,in=command,dataType=Command,required=true" json:"command"`
}

type Output struct {
	Version     *sdk.ReportVersion `parameter:"Version,kind=output,in=body,dataType=*sdk.ReportVersion" json:"version"`
	Diagnostics []sdk.Diagnostic   `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.apply,method=POST,connector=studio,handler=NewVersionApply" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.apply\",\"description\":\"Apply an authorized optimistic edit to one Datly Studio version\"}]" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewVersionApply" {
		return custom.Factory(NewVersionApply)
	}
	return nil
}

type applyHandler struct{}

func NewVersionApply() xhandler.Contract[Input, Output] { return &applyHandler{} }

func (*applyHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportId) == "" || input.VersionNo <= 0 || input.Command.ExpectedSourceRevision <= 0 {
		return publicError(400, "reportId, versionNo, and positive expectedSourceRevision are required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("version apply handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	guardInput := &guard.Input{}
	guardInput.SetJwt(input.Jwt)
	guardInput.SetAuth(input.Auth)
	guardInput.SetReportId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard"), Input: guardInput})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil {
		return fmt.Errorf("report edit guard returned %T", value)
	}
	if allowed.Item == nil || allowed.Item.Id != input.ReportId {
		return publicError(403, "Studio authorization denied")
	}
	read := &catalog.Input{}
	read.SetReportId(input.ReportId)
	read.SetVersionNo(input.VersionNo)
	read.SetLimit(2)
	read.SetOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[catalog.VersionComponent](), "version", "GET", "/_studio/report-version-store/catalog"), Input: read})
	if err != nil {
		return err
	}
	listed, ok := value.(*catalog.Output)
	if !ok || listed == nil {
		return fmt.Errorf("version source reader returned %T", value)
	}
	if len(listed.Versions) == 0 {
		return publicError(404, "report version not found")
	}
	if len(listed.Versions) != 1 || listed.Versions[0] == nil || listed.Versions[0].ReportId != input.ReportId ||
		listed.Versions[0].VersionNo != input.VersionNo {
		return fmt.Errorf("version source reader returned ambiguous or mismatched rows")
	}
	current := versionprojection.FromCatalog(listed.Versions[0])
	if current.State != "draft" {
		return publicError(409, "published version cannot be mutated")
	}
	if input.Command.ExpectedSourceRevision != current.SourceRevision {
		return publicError(409, "version source revision does not match")
	}
	if len(input.Command.Payload) == 0 {
		return publicError(400, "edit payload is required")
	}
	reportInput := &reports.ReportGetInput{}
	reportInput.SetJwt(input.Jwt)
	reportInput.SetAuth(input.Auth)
	reportInput.SetId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[reports.ReportComponent](), "report", "POST", "/v1/studio/sdk/components.get"), Input: reportInput})
	if err != nil {
		return err
	}
	report, ok := value.(*reports.ReportGetOutput)
	if !ok || report == nil || report.Item == nil || report.Item.Id != input.ReportId || report.Item.OwnerId == "" {
		return fmt.Errorf("report metadata reader returned %T without one matching row", value)
	}
	capInput := &capabilities.Input{}
	capInput.SetReportId(input.ReportId)
	capInput.SetSubject(input.Jwt.Subject)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[capabilities.CapabilityComponent](), "capability", "GET", "/_studio/report-capabilities"), Input: capInput})
	if err != nil {
		return err
	}
	caps, ok := value.(*capabilities.Output)
	if !ok || caps == nil || len(caps.Capabilities) != 1 || caps.Capabilities[0] == nil ||
		caps.Capabilities[0].ReportId != input.ReportId || caps.Capabilities[0].OwnerId != report.Item.OwnerId {
		return fmt.Errorf("report capability reader returned %T without one matching row", value)
	}
	canUseDQL := report.Item.OwnerId == input.Jwt.Subject || caps.Capabilities[0].CanUseDql
	spec := append(json.RawMessage(nil), current.ComponentSpec...)
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	authoredSQL, authoredDQL, generatedDQL := current.AuthoredSQL, current.AuthoredDQL, current.GeneratedDQL
	switch strings.ToLower(strings.TrimSpace(input.Command.Kind)) {
	case "replace_spec", "set_spec":
		var candidate any
		if err := json.Unmarshal(input.Command.Payload, &candidate); err != nil {
			return publicError(400, "component spec: "+err.Error())
		}
		spec = append(json.RawMessage(nil), input.Command.Payload...)
	case "set_dql":
		var payload struct {
			AuthoredDQL, GeneratedDQL string
		}
		if err := json.Unmarshal(input.Command.Payload, &payload); err != nil {
			return publicError(400, err.Error())
		}
		if strings.TrimSpace(payload.AuthoredDQL) == "" {
			return publicError(400, "authoredDql is required")
		}
		authoredDQL, generatedDQL = payload.AuthoredDQL, payload.GeneratedDQL
		if generatedDQL == "" {
			generatedDQL = authoredDQL
		}
	case "set_sql":
		var payload struct{ AuthoredSQL string }
		if err := json.Unmarshal(input.Command.Payload, &payload); err != nil {
			return publicError(400, err.Error())
		}
		if strings.TrimSpace(payload.AuthoredSQL) == "" {
			return publicError(400, "authoredSql is required")
		}
		authoredSQL, generatedDQL = payload.AuthoredSQL, payload.AuthoredSQL
	default:
		return publicError(400, fmt.Sprintf("unsupported version edit kind %q", input.Command.Kind))
	}
	hash := versionidentity.Hash(input.ReportId, input.VersionNo, current.AuthoringMode, authoredSQL, authoredDQL, spec)
	expected := current.SourceRevision
	row := &stored.StoredVersion{ReportId: input.ReportId, VersionNo: input.VersionNo,
		AuthoredSql: optional(authoredSQL), AuthoredDql: optional(authoredDQL),
		ComponentSpecJson: spec, SpecHash: hash, GeneratedDql: optional(generatedDQL),
		CompileStatus: "pending", CompileDiagnosticsJson: json.RawMessage(`[]`),
		SourceRevision: &expected,
		Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true, AuthoredSql: true,
			AuthoredDql: true, ComponentSpecJson: true, SpecHash: true, GeneratedDql: true,
			CompileStatus: true, CompileDiagnosticsJson: true, SourceRevision: true}}
	write := &stored.Input{}
	write.SetVersions([]*stored.StoredVersion{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.VersionComponent](), "version", "PATCH", "/_studio/report-version-store/edit"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, "version source revision does not match")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].ReportId != input.ReportId || updated.Data[0].VersionNo != input.VersionNo ||
		updated.Data[0].SourceRevision == nil {
		return fmt.Errorf("version edit writer returned %T without one updated row", value)
	}
	current.AuthoredSQL, current.AuthoredDQL, current.GeneratedDQL = authoredSQL, authoredDQL, generatedDQL
	current.ComponentSpec, current.SpecHash = spec, hash
	current.CompileStatus, current.CompileDiagnostics = "pending", json.RawMessage(`[]`)
	current.SourceRevision = *updated.Data[0].SourceRevision
	output.Version = readerinspection.RedactVersion(current, canUseDQL)
	return nil
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
