package download

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/internal/componentarchive"
	resourcefiles "github.com/viant/datly-studio/studio/report_resource_files/store_download"
	budget "github.com/viant/datly-studio/studio/report_resource_files/store_download_budget"
	versions "github.com/viant/datly-studio/studio/report_versions/reader"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// Input binds the same verified bearer and SDK identity for HTTP and MCP.
type Input struct {
	NamespaceId *string     `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId    string      `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int         `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
}

type Output struct {
	Filename  string   `parameter:"Filename,kind=output,in=body,dataType=string" json:"filename"`
	MediaType string   `parameter:"MediaType,kind=output,in=body,dataType=string" json:"mediaType"`
	Archive   []byte   `parameter:"Archive,kind=output,in=body,dataType=[]byte" json:"archive"`
	EntryDQL  string   `parameter:"EntryDQL,kind=output,in=body,dataType=string" json:"entryDql"`
	Files     []string `parameter:"Files,kind=output,in=body,dataType=[]string" json:"files"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.download,method=POST,connector=studio,handler=NewDownload" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.download\",\"description\":\"Download an authorized Datly Studio component archive\"}]" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDownload" {
		return custom.Factory(NewDownload)
	}
	return nil
}

type downloadHandler struct{}

func NewDownload() xhandler.Contract[Input, Output] { return &downloadHandler{} }

func (*downloadHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.ReportId == "" || input.VersionNo <= 0 {
		return &xresponse.Error{Code: 400, Cause: errors.New("reportId and positive versionNo are required")}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("download handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}

	// The source-bearing version reader applies ReportVersionRead, requiring a
	// current DQL grant before the unscoped internal resource reader is called.
	versionInput := &versions.Input{}
	versionInput.SetJwt(input.Jwt)
	versionInput.SetReportId(input.ReportId)
	versionInput.SetVersionNo(input.VersionNo)
	versionInput.SetLimit(1)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{
		Target: exec.ComponentTarget{
			Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[versions.VersionComponent]().PkgPath(), Name: "version"},
			Route:     spec.RouteRef{Method: "GET", Path: "/v1/studio/reports/{reportId}/versions/{versionNo}"},
		},
		Input: versionInput,
	})
	if err != nil {
		return err
	}
	read, ok := value.(*versions.Output)
	if !ok || read == nil {
		return fmt.Errorf("version reader returned %T", value)
	}
	if len(read.Versions) == 0 {
		return &xresponse.Error{Code: 404, Cause: errors.New("report version not found")}
	}
	if len(read.Versions) != 1 || read.Versions[0] == nil || read.Versions[0].ReportId == nil ||
		*read.Versions[0].ReportId != input.ReportId || read.Versions[0].VersionNo == nil ||
		*read.Versions[0].VersionNo != input.VersionNo {
		return fmt.Errorf("version reader returned an ambiguous or mismatched row")
	}
	version := read.Versions[0]
	source := ""
	if version.GeneratedDql != nil {
		source = *version.GeneratedDql
	}
	if source == "" && version.AuthoredDql != nil {
		source = *version.AuthoredDql
	}
	if source == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("component has no DQL source")}
	}
	if err := componentarchive.CheckSource(source); err != nil {
		return &xresponse.Error{Code: 400, Cause: err}
	}
	budgetInput := &budget.Input{}
	budgetInput.SetReportId(input.ReportId)
	budgetInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[budget.BudgetComponent]().PkgPath(), Name: "budget"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/report-resource-files-store/download-budget"},
	}, Input: budgetInput})
	if err != nil {
		return err
	}
	budgetOutput, ok := value.(*budget.Output)
	if !ok || budgetOutput == nil || budgetOutput.Summary == nil {
		return fmt.Errorf("resource budget reader returned %T without totals", value)
	}
	if err := componentarchive.CheckBudget(budgetOutput.Summary.FileCount, budgetOutput.Summary.ContentBytes); err != nil {
		if errors.Is(err, componentarchive.ErrInvalid) {
			return &xresponse.Error{Code: 400, Cause: err}
		}
		return err
	}

	filesInput := &resourcefiles.Input{}
	filesInput.SetReportId(input.ReportId)
	filesInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{
		Target: exec.ComponentTarget{
			Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[resourcefiles.FileComponent]().PkgPath(), Name: "file"},
			Route:     spec.RouteRef{Method: "GET", Path: "/_studio/report-resource-files-store/download"},
		},
		Input: filesInput,
	})
	if err != nil {
		return err
	}
	fileOutput, ok := value.(*resourcefiles.Output)
	if !ok || fileOutput == nil {
		return fmt.Errorf("resource reader returned %T", value)
	}
	files := make([]componentarchive.File, 0, len(fileOutput.Files))
	for _, file := range fileOutput.Files {
		if file == nil {
			return fmt.Errorf("resource reader returned a nil file")
		}
		files = append(files, componentarchive.File{ReportID: file.ReportId, VersionNo: file.VersionNo,
			ResourcePath: file.ResourcePath, Content: file.Content})
	}
	archive, err := componentarchive.Build(ctx, input.ReportId, input.VersionNo, source, files)
	if err != nil {
		if errors.Is(err, componentarchive.ErrInvalid) {
			return &xresponse.Error{Code: 400, Cause: err}
		}
		return err
	}
	output.Filename, output.MediaType, output.Archive, output.EntryDQL, output.Files =
		archive.Filename, archive.MediaType, archive.Archive, archive.EntryDQL, archive.Files
	return nil
}
