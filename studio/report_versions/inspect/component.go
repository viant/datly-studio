package inspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/bindly/locator"
	"github.com/viant/datly-studio/internal/readercomponent"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/readerinspection"
	"github.com/viant/datly-studio/internal/versionprojection"
	"github.com/viant/datly-studio/internal/versionrun"
	"github.com/viant/datly-studio/runtime/preview"
	studiors "github.com/viant/datly-studio/runtime/resources"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	connectors "github.com/viant/datly-studio/studio/connectors/store_catalog"
	"github.com/viant/datly-studio/studio/host"
	versions "github.com/viant/datly-studio/studio/report_versions/get"
	catalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	reports "github.com/viant/datly-studio/studio/reports/get"
	publishguard "github.com/viant/datly-studio/studio/reports/publish_guard"
	capabilities "github.com/viant/datly-studio/studio/reports/store_capabilities"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	cancel                 context.CancelFunc
	NamespaceId            *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt                    *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth                   *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId               string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo              int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	DiscoverColumns        *bool              `parameter:"DiscoverColumns,kind=body,in=discoverColumns,dataType=*bool,required=false" json:"discoverColumns,omitempty"`
	IncludePreviewInput    *bool              `parameter:"IncludePreviewInput,kind=body,in=includePreviewInput,dataType=*bool,required=false" json:"includePreviewInput,omitempty"`
	IncludeEffectiveWarmup *bool              `parameter:"IncludeEffectiveWarmup,kind=body,in=includeEffectiveWarmup,dataType=*bool,required=false" json:"includeEffectiveWarmup,omitempty"`
}

type Output struct {
	EffectiveWarmup *sdk.EffectiveWarmupInspection `parameter:"EffectiveWarmup,kind=output,in=body,dataType=*sdk.EffectiveWarmupInspection" json:"effectiveWarmup,omitempty"`
	PreviewInput    *sdk.PreviewInputContract      `parameter:"PreviewInput,kind=output,in=body,dataType=*sdk.PreviewInputContract" json:"previewInput,omitempty"`
	Version         *sdk.ReportVersion             `parameter:"Version,kind=output,in=body,dataType=*sdk.ReportVersion" json:"version"`
	DQL             string                         `parameter:"DQL,kind=output,in=body,dataType=string" json:"dql,omitempty"`
	Structure       json.RawMessage                `parameter:"Structure,kind=output,in=body,dataType=json.RawMessage" json:"structure,omitempty"`
	Diagnostics     []sdk.Diagnostic               `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
	Capabilities    sdk.ReportCapabilities         `parameter:"Capabilities,kind=output,in=body,dataType=sdk.ReportCapabilities" json:"capabilities"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.inspect,method=POST,connector=studio,handler=NewVersionInspect" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.inspect\",\"description\":\"Inspect an authorized exact Datly Studio reader version\"}]" componentCallPolicy:"imperative" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewVersionInspect" {
		return custom.Factory(NewVersionInspect)
	}
	return nil
}

type inspectHandler struct{}

func NewVersionInspect() xhandler.Contract[Input, Output] { return &inspectHandler{} }

func (*inspectHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportId) == "" || input.VersionNo <= 0 {
		return publicError(400, "reportId and positive versionNo are required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("version inspect handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	access := &versions.VersionGetInput{}
	access.SetJwt(input.Jwt)
	access.SetAuth(input.Auth)
	access.SetReportId(input.ReportId)
	access.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[versions.VersionComponent](), "version", "POST", "/v1/studio/sdk/versions.get"), Input: access})
	if err != nil {
		return err
	}
	visible, ok := value.(*versions.VersionGetOutput)
	if !ok || visible == nil {
		return fmt.Errorf("version metadata reader returned %T", value)
	}
	if visible.Item == nil || visible.Item.ReportId != input.ReportId || visible.Item.VersionNo != input.VersionNo {
		return publicError(404, "report version not found")
	}
	versionInput := &catalog.Input{}
	versionInput.SetReportId(input.ReportId)
	versionInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[catalog.VersionComponent](), "version", "GET", "/_studio/report-version-store/catalog"), Input: versionInput, Providers: []locator.Provider{readercomponent.Paging("version", 2, 0)}})
	if err != nil {
		return err
	}
	read, ok := value.(*catalog.Output)
	if !ok || read == nil || len(read.Versions) != 1 || read.Versions[0] == nil ||
		read.Versions[0].ReportId != input.ReportId || read.Versions[0].VersionNo != input.VersionNo {
		return fmt.Errorf("version source reader returned %T without one matching row", value)
	}
	version := versionprojection.FromCatalog(read.Versions[0])
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
	permissions := sdk.ReportCapabilities{CanView: caps.Capabilities[0].CanView,
		CanRun: caps.Capabilities[0].CanRun, CanEdit: caps.Capabilities[0].CanEdit,
		CanPublish: caps.Capabilities[0].CanPublish, CanUseDQL: caps.Capabilities[0].CanUseDql}
	if report.Item.OwnerId == input.Jwt.Subject {
		permissions = sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true}
	}
	permissions.CanManageACL = permissions.CanPublish
	available, err := activeConnectorNames(ctx, invoker, input.Jwt.Subject)
	if err != nil {
		return err
	}
	authoring, err := host.AuthoringFromContext(ctx)
	if err != nil {
		return err
	}
	types, err := authoring.RuntimeTypes()
	if err != nil {
		return err
	}
	value, found, lookupErr := session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if lookupErr != nil {
		return lookupErr
	}
	provider, ok := value.(xconnector.Provider)
	if !found || !ok {
		return fmt.Errorf("trusted Studio connector capability is unavailable")
	}
	db, dbErr := provider.Connector(ctx, "studio")
	if dbErr != nil || db == nil {
		return fmt.Errorf("configured Studio database is unavailable")
	}
	resourceVersion := studiors.Version{ReportID: input.ReportId, VersionNo: input.VersionNo}
	loaded, err := studiors.Load(ctx, db, []studiors.Version{resourceVersion})
	if err != nil {
		return err
	}
	service := readerbuilder.New(readerbuilder.Config{Resources: loaded.ByVersion[resourceVersion], Types: types, Scope: report.Item.ComponentScope,
		Name: report.Item.ComponentName, AvailableConnectors: available, AvailableCaches: readerinspection.AvailableCacheNames(authoring.ReaderCaches())})
	source := version.GeneratedDQL
	if strings.TrimSpace(source) == "" {
		source = version.AuthoredDQL
	}
	if strings.TrimSpace(source) == "" {
		source = version.AuthoredSQL
	}
	inspected := service.Apply(ctx, readerbuilder.Request{DQL: source,
		Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if (input.DiscoverColumns == nil || *input.DiscoverColumns) && permissions.CanUseDQL && inspected.Structure != nil && inspected.Structure.Component != nil {

		resolved, inspectErr := (preview.Dynamic{Predicates: authoring.PredicateCatalog(), StudioDB: db, ModulePath: authoring.ModulePath(), Types: types, Const: authoring.Constants(), Observability: authoring.Observability(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies()}).InspectContract(ctx, input.ReportId, input.VersionNo)
		if inspectErr != nil {
			inspected.Structure.Status = "partial"
			inspected.Diagnostics = append(inspected.Diagnostics, &transcribe.Diagnostic{Severity: transcribe.SeverityWarning, Code: "column_inspection_unavailable", Message: "Column metadata is unavailable; check the draft source and connector configuration."})
		} else if resolved != nil {
			readerinspection.HydrateColumns(inspected.Structure.Component.RootView, resolved.RootView)
		}
	}
	projected := readerinspection.Project(version, &inspected, permissions)
	wantInput := (input.IncludePreviewInput == nil || *input.IncludePreviewInput) && permissions.CanRun
	wantWarmup := false
	metadataCtx := ctx
	engine := preview.Dynamic{Predicates: authoring.PredicateCatalog(), StudioDB: db, ModulePath: authoring.ModulePath(), Types: types, Const: authoring.Constants(), Observability: authoring.Observability(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies()}
	inputAuthorized := false
	if wantInput {
		previewCtx, previewEngine, cancel, authorizeErr := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportId, input.VersionNo, 0)
		if authorizeErr == nil {
			input.cancel = cancel
			metadataCtx, engine, inputAuthorized = previewCtx, previewEngine, true
		}
	}
	warmupAuthorized := false
	if input.IncludeEffectiveWarmup == nil || *input.IncludeEffectiveWarmup {
		warmupAccess := &publishguard.Input{}
		warmupAccess.SetJwt(input.Jwt)
		warmupAccess.SetAuth(input.Auth)
		warmupAccess.SetReportId(input.ReportId)
		allowed, accessErr := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[publishguard.ReportComponent](), "report", "POST", "/_studio/reports/publish-guard"), Input: warmupAccess})
		if granted, ok := allowed.(*publishguard.Output); accessErr == nil && ok && granted != nil && granted.Item != nil && granted.Item.Id == input.ReportId {
			wantWarmup = true
			var header struct {
				Authorization string `bind:"kind=header,in=Authorization"`
			}
			if bindErr := session.Binder().Bind(ctx, &header); bindErr == nil {
				// Warmup inspection has always retained the verified bearer, even
				// when preview authorization does not need an ACL provider.
				metadataCtx = sdk.WithPrincipal(metadataCtx, sdk.Principal{Subject: input.Jwt.Subject})
				metadataCtx = sdk.WithNamespaceSelection(metadataCtx, namespaceaccess.ID(report.Item.OwnerId, report.Item.Namespace))
				metadataCtx = sdk.WithVerifiedCredential(metadataCtx, sdk.VerifiedCredential{Bearer: strings.TrimSpace(strings.TrimPrefix(header.Authorization, "Bearer ")), Claims: input.Jwt})
				if !inputAuthorized {
					var cancel context.CancelFunc
					metadataCtx, cancel = context.WithTimeout(metadataCtx, 30*time.Second)
					defer cancel()
				}
				engine.SourceValidator = authoring.SourceValidator()
				warmupAuthorized = true
			}
		}
	}
	previewInput, effectiveWarmup, _, _ := engine.InspectExecutionMetadata(metadataCtx, input.ReportId, input.VersionNo, inputAuthorized, warmupAuthorized)
	if wantInput {
		readerinspection.SetPreviewInput(projected, previewInput)
	}
	if wantWarmup {
		readerinspection.SetEffectiveWarmup(projected, effectiveWarmup)
	}
	*output = Output{Version: projected.Version, DQL: projected.DQL, EffectiveWarmup: projected.EffectiveWarmup,
		Structure: projected.Structure, Diagnostics: projected.Diagnostics, Capabilities: projected.Capabilities, PreviewInput: projected.PreviewInput}
	return nil
}

func activeConnectorNames(ctx context.Context, invoker exec.ComponentInvoker, subject string) ([]string, error) {
	result := []string{}
	for offset := 0; ; offset += 500 {
		input := &connectors.Input{}
		input.SetName("")
		input.SetQuery("")
		input.SetStatus("active")
		input.SetOwnerId("")
		input.SetDriver("")
		input.SetSubject(subject)
		input.SetScoped(true)
		input.SetPageLimit(500)
		input.SetPageOffset(offset)
		value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[connectors.ConnectorComponent](), "connector", "GET", "/_studio/connector-store/catalog"), Input: input})
		if err != nil {
			return nil, err
		}
		page, ok := value.(*connectors.Output)
		if !ok || page == nil || len(page.Connectors) > 500 {
			return nil, fmt.Errorf("active connector catalog returned %T with invalid page", value)
		}
		for _, item := range page.Connectors {
			if item == nil {
				return nil, fmt.Errorf("active connector catalog returned a nil row")
			}
			result = append(result, item.Name)
		}
		if len(page.Connectors) < 500 {
			break
		}
	}
	sort.Strings(result)
	return result, nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}

// FinalizeOutcome keeps metadata's deadline alive until its native artifact
// writes have completed, and releases it on success, failure, or panic.
func (*inspectHandler) FinalizeOutcome(ctx context.Context, invocation rhandler.Invocation, _ any, _ xhandler.Outcome) error {
	if input, ok := invocation.Input.(*Input); ok && input != nil && input.cancel != nil {
		input.cancel()
		input.cancel = nil
	}
	return nil
}
