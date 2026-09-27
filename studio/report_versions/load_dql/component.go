package load_dql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/dqlimport"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	versioncreate "github.com/viant/datly-studio/studio/report_versions/create"
	head "github.com/viant/datly-studio/studio/report_versions/store_head"
	stored "github.com/viant/datly-studio/studio/report_versions/store_import"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	reports "github.com/viant/datly-studio/studio/reports/get"
	capabilities "github.com/viant/datly-studio/studio/reports/store_capabilities"
	pointer "github.com/viant/datly-studio/studio/reports/store_draft_pointer"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/sqlx/io/errx"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Options struct {
	DQL   string `json:"dql"`
	Notes string `json:"notes,omitempty"`
}

type Input struct {
	Jwt      *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input    Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

type Output struct {
	Version  *versioncreate.Output `parameter:"Version,kind=output,in=body,dataType=*versioncreate.Output" json:"version"`
	EntryDQL string                `parameter:"EntryDQL,kind=output,in=body,dataType=string" json:"entryDql"`
	Entries  []string              `parameter:"Entries,kind=output,in=body,dataType=[]string" json:"entries"`
	Files    []string              `parameter:"Files,kind=output,in=body,dataType=[]string" json:"files"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.load_dql,method=POST,connector=studio,handler=NewLoadDQL" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.load_dql\",\"description\":\"Import an authorized Datly Studio DQL document as a draft version\"}]" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewLoadDQL" {
		return custom.Factory(NewLoadDQL)
	}
	return nil
}

type loadHandler struct{}

func NewLoadDQL() xhandler.Contract[Input, Output] { return &loadHandler{} }

func (*loadHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	return ImportBundle(ctx, session, input, output, func() (*sdk.DQLBundle, string, error) {
		bundle, err := sdk.ReadDQL(strings.NewReader(input.Input.DQL))
		if err != nil {
			return nil, "", err
		}
		return bundle, bundle.Entries[0], nil
	})
}

// ImportBundle keeps authorization before potentially expensive parsing, then
// commits the version, resources and draft pointer in one managed SQL unit.
// The archive endpoint supplies its own parser and root selection here.
func ImportBundle(ctx context.Context, session xhandler.Session, input *Input, output *Output,
	parse func() (*sdk.DQLBundle, string, error)) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportId) == "" {
		return publicError(400, "reportId is required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("DQL import handler session and output are required")
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
	bundle, entry, err := parse()
	if err != nil {
		return publicError(400, err.Error())
	}
	if bundle == nil || bundle.Files == nil || bundle.Files[entry] == nil {
		return fmt.Errorf("DQL parser returned no selected entry")
	}
	files := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		files = append(files, name)
	}
	sort.Strings(files)
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
	start, present, err := session.Binder().Lookup(ctx, xhandler.TransactionStarterKey)
	if err != nil {
		return err
	}
	starter, ok := start.(xhandler.TransactionStarter)
	if !present || !ok {
		return fmt.Errorf("managed transaction starter is unavailable")
	}
	if err := starter.Start(ctx); err != nil {
		return err
	}
	now := time.Now().UTC()
	source := string(bundle.Files[entry])
	specHash := dqlimport.SpecHash(input.ReportId, next, source, bundle, files)
	namespace := dqlimport.Namespace(input.ReportId, sdk.OwnerPackageSegment(report.Item.OwnerId))
	version := &stored.ImportedVersion{ReportId: input.ReportId, VersionNo: next, State: "draft", AuthoringMode: "dql",
		AuthoredDql: &source, GeneratedDql: &source, ComponentSpecJson: json.RawMessage(`{}`),
		SpecFormatVersion: "studio.v1", SpecHash: specHash, TypeManifestJson: json.RawMessage(`{}`),
		CompileStatus: "pending", DatlyVersion: "v1", CompilerVersion: "studio.v1",
		SourceRevision: 1, CreatedBy: input.Jwt.Subject, CreatedAt: now,
		Has: &stored.ImportedVersionHas{ReportId: true, VersionNo: true, State: true,
			AuthoringMode: true, AuthoredDql: true, GeneratedDql: true, ComponentSpecJson: true,
			SpecFormatVersion: true, SpecHash: true, TypeManifestJson: true, CompileStatus: true,
			DatlyVersion: true, CompilerVersion: true, SourceRevision: true, Notes: true,
			CreatedBy: true, CreatedAt: true}}
	if strings.TrimSpace(input.Input.Notes) != "" {
		version.Notes = &input.Input.Notes
	}
	for _, name := range files {
		content := bundle.Files[name]
		version.File = append(version.File, &stored.ImportedResourceFile{Namespace: namespace,
			ResourcePath: name, Content: content, CreatedAt: now,
			Has: &stored.ImportedResourceFileHas{Namespace: true, ResourcePath: true, Content: true, CreatedAt: true}})
	}
	version.Has.File = len(version.File) > 0
	write := &stored.Input{}
	write.SetVersions([]*stored.ImportedVersion{version})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.VersionComponent](), "version", "POST", "/_studio/report-version-store/import"), Input: write})
	if err != nil {
		if errors.Is(err, errx.ErrDuplicateKey) {
			return publicError(409, "report version changed during import")
		}
		return err
	}
	inserted, ok := value.(*stored.Output)
	if !ok || inserted == nil || len(inserted.Data) != 1 || inserted.Data[0] == nil ||
		inserted.Data[0].ReportId != input.ReportId || inserted.Data[0].VersionNo != next {
		return fmt.Errorf("version import writer returned %T without one matching row", value)
	}
	etag, draft := report.Item.Etag, next
	pointerRow := &pointer.DraftPointer{Id: input.ReportId, CurrentDraftVersion: &draft,
		Etag: &etag, UpdatedAt: &now,
		Has: &pointer.DraftPointerHas{Id: true, CurrentDraftVersion: true, Etag: true, UpdatedAt: true}}
	pointerInput := &pointer.Input{}
	pointerInput.SetReports([]*pointer.DraftPointer{pointerRow})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[pointer.ReportComponent](), "report", "PATCH", "/_studio/report-store/draft-pointer"), Input: pointerInput})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, fmt.Sprintf("report %q was modified concurrently", input.ReportId))
		}
		return err
	}
	advanced, ok := value.(*pointer.Output)
	if !ok || advanced == nil || len(advanced.Data) != 1 || advanced.Data[0] == nil ||
		advanced.Data[0].Id != input.ReportId || advanced.Data[0].CurrentDraftVersion == nil ||
		*advanced.Data[0].CurrentDraftVersion != next {
		return fmt.Errorf("draft pointer writer returned %T without one advanced row", value)
	}
	versionDTO := &versioncreate.Output{ReportId: input.ReportId, VersionNo: next, State: "draft", AuthoringMode: "dql",
		ComponentSpec: json.RawMessage(`{}`), TypeManifest: json.RawMessage(`{}`),
		SpecFormatVersion: "studio.v1", SpecHash: specHash, CompileStatus: "pending",
		DatlyVersion: "v1", CompilerVersion: "studio.v1", SourceRevision: 1,
		CreatedBy: input.Jwt.Subject, CreatedAt: now}
	if version.Notes != nil {
		versionDTO.Notes = *version.Notes
	}
	if canUseDQL {
		versionDTO.AuthoredDQL, versionDTO.GeneratedDQL = source, source
	}
	*output = Output{Version: versionDTO, EntryDQL: entry, Entries: append([]string(nil), bundle.Entries...), Files: files}
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
