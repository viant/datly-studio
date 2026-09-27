package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/internal/resourcesnapshot"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	files "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	folders "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	skills "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	versionget "github.com/viant/datly-studio/studio/report_versions/get"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	Jwt       *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId  string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
}

type Output struct {
	Files   []*sdk.ResourceFile   `parameter:"Files,kind=output,in=body,dataType=[]*sdk.ResourceFile" json:"files"`
	Folders []*sdk.ResourceFolder `parameter:"Folders,kind=output,in=body,dataType=[]*sdk.ResourceFolder" json:"folders"`
	Skills  []*sdk.SkillRoot      `parameter:"Skills,kind=output,in=body,dataType=[]*sdk.SkillRoot" json:"skills"`
	Version *sdk.ReportVersion    `parameter:"Version,kind=output,in=body,dataType=*sdk.ReportVersion" json:"version,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"resources,path=/v1/studio/sdk/resources.get,method=POST,connector=studio,handler=NewResourceSnapshot" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.get\",\"description\":\"Read an authorized Datly Studio resource snapshot\"}]" caseFormat:"lc"`
}

var ResourceSnapshotDatly = new(Component)
var ResourceSnapshotDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewResourceSnapshot" {
		return custom.Factory(NewResourceSnapshot)
	}
	return nil
}

type snapshotHandler struct{}

func NewResourceSnapshot() xhandler.Contract[Input, Output] { return &snapshotHandler{} }

func (*snapshotHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.ReportId == "" || input.VersionNo <= 0 {
		return &xresponse.Error{Code: 400, Cause: errors.New("reportId and positive versionNo are required")}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("resource snapshot handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}

	// Resolve the exact authorized version before invoking unscoped internal
	// readers. The child reader supplies typed ReportVersionMetadataRead scope and
	// capability-aware DQL redaction.
	versionInput := &versionget.VersionGetInput{}
	versionInput.SetJwt(input.Jwt)
	versionInput.SetAuth(input.Auth)
	versionInput.SetReportId(input.ReportId)
	versionInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[versionget.VersionComponent]().PkgPath(), Name: "version"},
		Route:     spec.RouteRef{Method: "POST", Path: "/v1/studio/sdk/versions.get"},
	}, Input: versionInput})
	if err != nil {
		return err
	}
	version, ok := value.(*versionget.VersionGetOutput)
	if !ok || version == nil {
		return fmt.Errorf("version reader returned %T", value)
	}
	if err := version.ProjectItem(input.ReportId, input.VersionNo); err != nil {
		return err
	}
	encoded, err := json.Marshal(version)
	if err != nil {
		return err
	}
	output.Version = &sdk.ReportVersion{}
	if err = json.Unmarshal(encoded, output.Version); err != nil {
		return err
	}

	fileInput := &files.Input{}
	fileInput.SetReportId(input.ReportId)
	fileInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[files.FileComponent]().PkgPath(), Name: "file"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/resource-snapshot/files"},
	}, Input: fileInput})
	if err != nil {
		return err
	}
	fileOutput, ok := value.(*files.Output)
	if !ok || fileOutput == nil {
		return fmt.Errorf("resource file reader returned %T", value)
	}
	output.Files, err = resourcesnapshot.Files(input.ReportId, input.VersionNo, fileOutput.Files)
	if err != nil {
		return err
	}

	folderInput := &folders.Input{}
	folderInput.SetReportId(input.ReportId)
	folderInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[folders.FolderComponent]().PkgPath(), Name: "folder"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/resource-snapshot/folders"},
	}, Input: folderInput})
	if err != nil {
		return err
	}
	folderOutput, ok := value.(*folders.Output)
	if !ok || folderOutput == nil {
		return fmt.Errorf("resource folder reader returned %T", value)
	}
	output.Folders, err = resourcesnapshot.Folders(input.ReportId, input.VersionNo, folderOutput.Folders)
	if err != nil {
		return err
	}

	skillInput := &skills.Input{}
	skillInput.SetReportId(input.ReportId)
	skillInput.SetVersionNo(input.VersionNo)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[skills.SkillComponent]().PkgPath(), Name: "skill"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/resource-snapshot/skills"},
	}, Input: skillInput})
	if err != nil {
		return err
	}
	skillOutput, ok := value.(*skills.Output)
	if !ok || skillOutput == nil {
		return fmt.Errorf("skill reader returned %T", value)
	}
	output.Skills, err = resourcesnapshot.Skills(input.ReportId, input.VersionNo, skillOutput.Skills)
	return err
}
