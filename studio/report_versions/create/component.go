package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/versionidentity"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	head "github.com/viant/datly-studio/studio/report_versions/store_head"
	stored "github.com/viant/datly-studio/studio/report_versions/store_insert"
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

type Options struct {
	AuthoringMode string          `json:"authoringMode,omitempty"`
	AuthoredSQL   string          `json:"authoredSql,omitempty"`
	AuthoredDQL   string          `json:"authoredDql,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	CreatedBy     string          `json:"createdBy,omitempty"`
	ComponentSpec json.RawMessage `json:"componentSpec,omitempty"`
}

type Input struct {
	Jwt      *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input    Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

type Output struct {
	ReportId            string          `parameter:"ReportId,kind=output,in=body,dataType=string" json:"reportId"`
	VersionNo           int             `parameter:"VersionNo,kind=output,in=body,dataType=int" json:"versionNo"`
	State               string          `parameter:"State,kind=output,in=body,dataType=string" json:"state"`
	AuthoringMode       string          `parameter:"AuthoringMode,kind=output,in=body,dataType=string" json:"authoringMode"`
	AuthoredSQL         string          `parameter:"AuthoredSQL,kind=output,in=body,dataType=string" json:"authoredSql,omitempty"`
	AuthoredDQL         string          `parameter:"AuthoredDQL,kind=output,in=body,dataType=string" json:"authoredDql,omitempty"`
	ComponentSpec       json.RawMessage `parameter:"ComponentSpec,kind=output,in=body,dataType=json.RawMessage" json:"componentSpec,omitempty"`
	DQLExportLimits     json.RawMessage `parameter:"DQLExportLimits,kind=output,in=body,dataType=json.RawMessage" json:"dqlExportLimits,omitempty"`
	TypeManifest        json.RawMessage `parameter:"TypeManifest,kind=output,in=body,dataType=json.RawMessage" json:"typeManifest,omitempty"`
	ResourceManifest    json.RawMessage `parameter:"ResourceManifest,kind=output,in=body,dataType=json.RawMessage" json:"resourceManifest,omitempty"`
	ComponentDescriptor json.RawMessage `parameter:"ComponentDescriptor,kind=output,in=body,dataType=json.RawMessage" json:"componentDescriptor,omitempty"`
	SpecFormatVersion   string          `parameter:"SpecFormatVersion,kind=output,in=body,dataType=string" json:"specFormatVersion"`
	SpecHash            string          `parameter:"SpecHash,kind=output,in=body,dataType=string" json:"specHash"`
	GeneratedDQL        string          `parameter:"GeneratedDQL,kind=output,in=body,dataType=string" json:"generatedDql,omitempty"`
	CompileStatus       string          `parameter:"CompileStatus,kind=output,in=body,dataType=string" json:"compileStatus"`
	CompileDiagnostics  json.RawMessage `parameter:"CompileDiagnostics,kind=output,in=body,dataType=json.RawMessage" json:"compileDiagnostics,omitempty"`
	DatlyVersion        string          `parameter:"DatlyVersion,kind=output,in=body,dataType=string" json:"datlyVersion"`
	CompilerVersion     string          `parameter:"CompilerVersion,kind=output,in=body,dataType=string" json:"compilerVersion"`
	SourceRevision      int64           `parameter:"SourceRevision,kind=output,in=body,dataType=int64" json:"sourceRevision"`
	Notes               string          `parameter:"Notes,kind=output,in=body,dataType=string" json:"notes,omitempty"`
	CreatedBy           string          `parameter:"CreatedBy,kind=output,in=body,dataType=string" json:"createdBy"`
	CreatedAt           time.Time       `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	ValidatedAt         *time.Time      `parameter:"ValidatedAt,kind=output,in=body,dataType=*time.Time" json:"validatedAt,omitempty"`
	PublishedAt         *time.Time      `parameter:"PublishedAt,kind=output,in=body,dataType=*time.Time" json:"publishedAt,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.create,method=POST,connector=studio,handler=NewVersionCreate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.create\",\"description\":\"Create an authorized Datly Studio draft reader version\"}]" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewVersionCreate" {
		return custom.Factory(NewVersionCreate)
	}
	return nil
}

type createHandler struct{}

func NewVersionCreate() xhandler.Contract[Input, Output] { return &createHandler{} }

func (*createHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportId) == "" {
		return publicError(400, "reportId is required")
	}
	if actor := strings.TrimSpace(input.Input.CreatedBy); actor != "" && actor != input.Jwt.Subject {
		return publicError(403, "cannot create a version for another principal")
	}
	mode := strings.ToLower(strings.TrimSpace(input.Input.AuthoringMode))
	if mode == "" {
		mode = "structured"
	}
	if mode != "sql" && mode != "dql" && mode != "structured" {
		return publicError(400, "authoringMode must be sql, dql, or structured")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("version create handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	accessInput := &guard.Input{}
	accessInput.SetJwt(input.Jwt)
	accessInput.SetAuth(input.Auth)
	accessInput.SetReportId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard"), Input: accessInput})
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
	reportInput := &reports.ReportGetInput{}
	reportInput.SetJwt(input.Jwt)
	reportInput.SetAuth(input.Auth)
	reportInput.SetId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[reports.ReportComponent](), "report", "POST", "/v1/studio/sdk/components.get"), Input: reportInput})
	if err != nil {
		return err
	}
	report, ok := value.(*reports.ReportGetOutput)
	if !ok || report == nil || report.Item == nil || report.Item.Id != input.ReportId {
		return publicError(404, "report not found")
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
	headInput := &head.Input{}
	headInput.SetReportId(input.ReportId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[head.HeadComponent](), "head", "GET", "/_studio/report-version-store/head"), Input: headInput})
	if err != nil {
		return err
	}
	latest, ok := value.(*head.Output)
	if !ok || latest == nil || len(latest.Heads) != 1 || latest.Heads[0] == nil || latest.Heads[0].MaxVersionNo < 0 {
		return fmt.Errorf("version head returned %T without one valid total", value)
	}
	next := latest.Heads[0].MaxVersionNo + 1
	spec := append(json.RawMessage(nil), input.Input.ComponentSpec...)
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	generated := input.Input.AuthoredDQL
	if generated == "" {
		generated = input.Input.AuthoredSQL
	}
	specHash := versionidentity.Hash(input.ReportId, next, mode, input.Input.AuthoredSQL, input.Input.AuthoredDQL, spec)
	now := time.Now().UTC()
	row := &stored.StoredVersion{ReportId: input.ReportId, VersionNo: next, State: "draft", AuthoringMode: mode,
		AuthoredSql: optional(input.Input.AuthoredSQL), AuthoredDql: optional(input.Input.AuthoredDQL),
		ComponentSpecJson: spec, SpecFormatVersion: "studio.v1", SpecHash: specHash,
		GeneratedDql: optional(generated), TypeManifestJson: json.RawMessage(`{}`),
		CompileStatus: "pending", DatlyVersion: "v1", CompilerVersion: "studio.v1",
		SourceRevision: 1, Notes: optional(input.Input.Notes), CreatedBy: input.Jwt.Subject, CreatedAt: now,
		Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true, State: true,
			AuthoringMode: true, AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true,
			SpecFormatVersion: true, SpecHash: true, GeneratedDql: true, TypeManifestJson: true,
			CompileStatus: true, DatlyVersion: true, CompilerVersion: true, SourceRevision: true,
			Notes: true, CreatedBy: true, CreatedAt: true}}
	write := &stored.Input{}
	write.SetVersions([]*stored.StoredVersion{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.VersionComponent](), "version", "POST", "/_studio/report-version-store/insert"), Input: write})
	if err != nil {
		return err
	}
	inserted, ok := value.(*stored.Output)
	if !ok || inserted == nil || len(inserted.Data) != 1 || inserted.Data[0] == nil ||
		inserted.Data[0].ReportId != input.ReportId || inserted.Data[0].VersionNo != next {
		return fmt.Errorf("version insert returned %T without one matching row", value)
	}
	*output = Output{ReportId: input.ReportId, VersionNo: next, State: "draft", AuthoringMode: mode,
		AuthoredSQL: input.Input.AuthoredSQL, ComponentSpec: spec, TypeManifest: json.RawMessage(`{}`),
		SpecFormatVersion: "studio.v1", SpecHash: specHash, CompileStatus: "pending",
		DatlyVersion: "v1", CompilerVersion: "studio.v1", SourceRevision: 1,
		Notes: input.Input.Notes, CreatedBy: input.Jwt.Subject, CreatedAt: now}
	if canUseDQL {
		output.AuthoredDQL, output.GeneratedDQL = input.Input.AuthoredDQL, generated
	}
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
