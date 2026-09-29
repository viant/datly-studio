package validate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/internal/readerinspection"
	"github.com/viant/datly-studio/internal/versionprojection"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	catalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	stored "github.com/viant/datly-studio/studio/report_versions/store_validation"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	capabilities "github.com/viant/datly-studio/studio/reports/store_capabilities"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

const modulePath = "github.com/viant/datly-studio"
const maxValidation = 30 * time.Second

type Input struct {
	NamespaceId            *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt                    *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth                   *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID               string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo              int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	ExpectedSourceRevision int64              `parameter:"ExpectedSourceRevision,kind=body,in=expectedSourceRevision,dataType=int64,required=true" json:"expectedSourceRevision"`
}

type Output struct {
	Valid       bool               `parameter:"Valid,kind=output,in=body,dataType=bool" json:"valid"`
	Version     *sdk.ReportVersion `parameter:"Version,kind=output,in=body,dataType=*sdk.ReportVersion" json:"version"`
	Diagnostics []sdk.Diagnostic   `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.validate,method=POST,connector=studio,handler=NewValidate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.validate\",\"description\":\"Validate the exact authorized Datly reader version and persist compile evidence\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewValidate" {
		return custom.Factory(NewValidate)
	}
	return nil
}

type handler struct{}

func NewValidate() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportID) == "" || input.VersionNo <= 0 || input.ExpectedSourceRevision <= 0 {
		return publisherguard.PublicError(400, "reportId, versionNo, and positive expectedSourceRevision are required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("version validation session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	access := &guard.Input{}
	access.SetJwt(input.Jwt)
	access.SetAuth(input.Auth)
	access.SetReportId(input.ReportID)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard"), Input: access})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != input.ReportID {
		return publisherguard.PublicError(403, "Studio authorization denied")
	}
	read := &catalog.Input{}
	read.SetReportId(input.ReportID)
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
		return publisherguard.PublicError(404, "report version not found")
	}
	if len(listed.Versions) != 1 || listed.Versions[0] == nil || listed.Versions[0].ReportId != input.ReportID || listed.Versions[0].VersionNo != input.VersionNo {
		return fmt.Errorf("version source reader returned ambiguous or mismatched rows")
	}
	version := versionprojection.FromCatalog(listed.Versions[0])
	if version.SourceRevision != input.ExpectedSourceRevision {
		return publisherguard.PublicError(409, "version source revision does not match")
	}
	capInput := &capabilities.Input{}
	capInput.SetReportId(input.ReportID)
	capInput.SetSubject(input.Jwt.Subject)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[capabilities.CapabilityComponent](), "capability", "GET", "/_studio/report-capabilities"), Input: capInput})
	if err != nil {
		return err
	}
	caps, ok := value.(*capabilities.Output)
	if !ok || caps == nil || len(caps.Capabilities) != 1 || caps.Capabilities[0] == nil || caps.Capabilities[0].ReportId != input.ReportID {
		return fmt.Errorf("report capability reader returned %T without matching row", value)
	}
	canUseDQL := caps.Capabilities[0].OwnerId == input.Jwt.Subject || caps.Capabilities[0].CanUseDql
	valid := strings.TrimSpace(version.AuthoredDQL) != "" || strings.TrimSpace(version.AuthoredSQL) != ""
	var diagnostics []sdk.Diagnostic
	if !valid {
		diagnostics = append(diagnostics, sdk.Diagnostic{Severity: "error", Code: "empty_source", Message: "authored SQL or DQL is required"})
	} else {
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
		validationCtx, cancel := context.WithTimeout(ctx, maxValidation)
		defer cancel()
		validationCtx = sdk.WithPrincipal(validationCtx, sdk.Principal{Subject: input.Jwt.Subject})
		if input.NamespaceId != nil {
			validationCtx = sdk.WithNamespaceSelection(validationCtx, *input.NamespaceId)
		}
		validationCtx = sdk.WithVerifiedCredential(validationCtx, sdk.VerifiedCredential{Claims: input.Jwt})
		if validateErr := (preview.Dynamic{StudioDB: db, ModulePath: modulePath}).Validate(validationCtx, input.ReportID, input.VersionNo); validateErr != nil {
			valid = false
			var compileErr *transcribe.CompileError
			if errors.As(validateErr, &compileErr) && len(compileErr.Diagnostics) > 0 {
				for _, item := range compileErr.Diagnostics {
					if item != nil {
						diagnostics = append(diagnostics, sdk.Diagnostic{Severity: string(item.Severity), Code: item.Code,
							Message: item.Message, Hint: item.Hint, Line: item.Span.Start.Line, Column: item.Span.Start.Char})
					}
				}
			} else {
				diagnostics = append(diagnostics, sdk.Diagnostic{Severity: "error", Code: "runtime_contract", Message: "reader runtime contract could not be initialized"})
			}
		}
	}
	status := "valid"
	if !valid {
		status = "invalid"
	}
	diagnosticJSON, err := json.Marshal(diagnostics)
	if err != nil {
		return err
	}
	now, expected := time.Now().UTC(), input.ExpectedSourceRevision
	row := &stored.StoredVersion{ReportId: input.ReportID, VersionNo: input.VersionNo,
		CompileStatus: status, CompileDiagnosticsJson: diagnosticJSON, ValidatedAt: &now,
		SourceRevision: &expected,
		Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true,
			CompileStatus: true, CompileDiagnosticsJson: true, ValidatedAt: true, SourceRevision: true}}
	write := &stored.Input{}
	write.SetVersions([]*stored.StoredVersion{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.VersionComponent](), "version", "PATCH", "/_studio/report-version-store/validation"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publisherguard.PublicError(409, "version source revision changed during validation")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].ReportId != input.ReportID || updated.Data[0].VersionNo != input.VersionNo {
		return fmt.Errorf("validation writer returned %T without one matching row", value)
	}
	version.CompileStatus, version.CompileDiagnostics, version.ValidatedAt = status, diagnosticJSON, &now
	output.Valid = valid
	output.Version = readerinspection.RedactVersion(version, canUseDQL)
	output.Diagnostics = readerinspection.RedactDiagnostics(diagnostics, canUseDQL)
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
