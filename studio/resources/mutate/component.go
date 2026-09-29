package mutate

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	resourceget "github.com/viant/datly-studio/studio/resources/get"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type identity struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
}

type FileUpsertInput struct {
	identity
	Value *sdk.ResourceFile `parameter:"Value,kind=body,in=,dataType=*sdk.ResourceFile,required=true" anonymous:"true"`
}
type FileDeleteInput struct {
	identity
	Value *sdk.ResourceDeleteInput `parameter:"Value,kind=body,in=,dataType=*sdk.ResourceDeleteInput,required=true" anonymous:"true"`
}
type FolderUpsertInput struct {
	identity
	Value *sdk.ResourceFolder `parameter:"Value,kind=body,in=,dataType=*sdk.ResourceFolder,required=true" anonymous:"true"`
}
type FolderDeleteInput struct {
	identity
	Value *sdk.ResourceDeleteInput `parameter:"Value,kind=body,in=,dataType=*sdk.ResourceDeleteInput,required=true" anonymous:"true"`
}
type SkillUpsertInput struct {
	identity
	Value *sdk.SkillRoot `parameter:"Value,kind=body,in=,dataType=*sdk.SkillRoot,required=true" anonymous:"true"`
}
type SkillDeleteInput struct {
	identity
	Value *sdk.ResourceDeleteInput `parameter:"Value,kind=body,in=,dataType=*sdk.ResourceDeleteInput,required=true" anonymous:"true"`
}

type Output struct {
	Files   []*sdk.ResourceFile   `parameter:"Files,kind=output,in=body,dataType=[]*sdk.ResourceFile" json:"files"`
	Folders []*sdk.ResourceFolder `parameter:"Folders,kind=output,in=body,dataType=[]*sdk.ResourceFolder" json:"folders"`
	Skills  []*sdk.SkillRoot      `parameter:"Skills,kind=output,in=body,dataType=[]*sdk.SkillRoot" json:"skills"`
	Version *sdk.ReportVersion    `parameter:"Version,kind=output,in=body,dataType=*sdk.ReportVersion" json:"version,omitempty"`
}

type FileUpsertComponent struct {
	Contract xdatly.Component[FileUpsertInput, Output] `component:"resource_upsert_file,path=/v1/studio/sdk/resources.upsert_file,method=POST,connector=studio,handler=NewUpsertFile" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.upsert_file\",\"description\":\"Upsert an authorized Datly Studio resource file\"}]" caseFormat:"lc"`
}
type FileDeleteComponent struct {
	Contract xdatly.Component[FileDeleteInput, Output] `component:"resource_delete_file,path=/v1/studio/sdk/resources.delete_file,method=POST,connector=studio,handler=NewDeleteFile" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.delete_file\",\"description\":\"Delete an authorized Datly Studio resource file\"}]" caseFormat:"lc"`
}
type FolderUpsertComponent struct {
	Contract xdatly.Component[FolderUpsertInput, Output] `component:"resource_upsert_folder,path=/v1/studio/sdk/resources.upsert_folder,method=POST,connector=studio,handler=NewUpsertFolder" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.upsert_folder\",\"description\":\"Upsert an authorized Datly Studio resource folder\"}]" caseFormat:"lc"`
}
type FolderDeleteComponent struct {
	Contract xdatly.Component[FolderDeleteInput, Output] `component:"resource_delete_folder,path=/v1/studio/sdk/resources.delete_folder,method=POST,connector=studio,handler=NewDeleteFolder" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.delete_folder\",\"description\":\"Delete an authorized Datly Studio resource folder\"}]" caseFormat:"lc"`
}
type SkillUpsertComponent struct {
	Contract xdatly.Component[SkillUpsertInput, Output] `component:"resource_upsert_skill,path=/v1/studio/sdk/resources.upsert_skill,method=POST,connector=studio,handler=NewUpsertSkill" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.upsert_skill\",\"description\":\"Upsert an authorized Datly Studio skill root\"}]" caseFormat:"lc"`
}
type SkillDeleteComponent struct {
	Contract xdatly.Component[SkillDeleteInput, Output] `component:"resource_delete_skill,path=/v1/studio/sdk/resources.delete_skill,method=POST,connector=studio,handler=NewDeleteSkill" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.resources.delete_skill\",\"description\":\"Delete an authorized Datly Studio skill root\"}]" caseFormat:"lc"`
}

var (
	FileUpsertDatly                      = new(FileUpsertComponent)
	FileDeleteDatly                      = new(FileDeleteComponent)
	FolderUpsertDatly                    = new(FolderUpsertComponent)
	FolderDeleteDatly                    = new(FolderDeleteComponent)
	SkillUpsertDatly                     = new(SkillUpsertComponent)
	SkillDeleteDatly                     = new(SkillDeleteComponent)
	_datlyReachableFileUpsertComponent   = reflect.TypeFor[FileUpsertComponent]()
	_datlyReachableFileDeleteComponent   = reflect.TypeFor[FileDeleteComponent]()
	_datlyReachableFolderUpsertComponent = reflect.TypeFor[FolderUpsertComponent]()
	_datlyReachableFolderDeleteComponent = reflect.TypeFor[FolderDeleteComponent]()
	_datlyReachableSkillUpsertComponent  = reflect.TypeFor[SkillUpsertComponent]()
	_datlyReachableSkillDeleteComponent  = reflect.TypeFor[SkillDeleteComponent]()
)

func (FileUpsertComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpsertFile" {
		return custom.Factory(NewUpsertFile)
	}
	return nil
}
func (FileDeleteComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDeleteFile" {
		return custom.Factory(NewDeleteFile)
	}
	return nil
}
func (FolderUpsertComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpsertFolder" {
		return custom.Factory(NewUpsertFolder)
	}
	return nil
}
func (FolderDeleteComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDeleteFolder" {
		return custom.Factory(NewDeleteFolder)
	}
	return nil
}
func (SkillUpsertComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpsertSkill" {
		return custom.Factory(NewUpsertSkill)
	}
	return nil
}
func (SkillDeleteComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDeleteSkill" {
		return custom.Factory(NewDeleteSkill)
	}
	return nil
}

type fileUpsertHandler struct{}
type fileDeleteHandler struct{}
type folderUpsertHandler struct{}
type folderDeleteHandler struct{}
type skillUpsertHandler struct{}
type skillDeleteHandler struct{}

func NewUpsertFile() xhandler.Contract[FileUpsertInput, Output]     { return &fileUpsertHandler{} }
func NewDeleteFile() xhandler.Contract[FileDeleteInput, Output]     { return &fileDeleteHandler{} }
func NewUpsertFolder() xhandler.Contract[FolderUpsertInput, Output] { return &folderUpsertHandler{} }
func NewDeleteFolder() xhandler.Contract[FolderDeleteInput, Output] { return &folderDeleteHandler{} }
func NewUpsertSkill() xhandler.Contract[SkillUpsertInput, Output]   { return &skillUpsertHandler{} }
func NewDeleteSkill() xhandler.Contract[SkillDeleteInput, Output]   { return &skillDeleteHandler{} }

func (*fileUpsertHandler) Exec(ctx context.Context, session xhandler.Session, input *FileUpsertInput, output *Output) error {
	if input == nil || input.Value == nil {
		return publisherguard.PublicError(400, "resource file is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesUpsertFile, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}
func (*fileDeleteHandler) Exec(ctx context.Context, session xhandler.Session, input *FileDeleteInput, output *Output) error {
	if input == nil || input.Value == nil || strings.TrimSpace(input.Value.ResourceID) == "" {
		return publisherguard.PublicError(400, "resourceId is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesDeleteFile, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}
func (*folderUpsertHandler) Exec(ctx context.Context, session xhandler.Session, input *FolderUpsertInput, output *Output) error {
	if input == nil || input.Value == nil {
		return publisherguard.PublicError(400, "resource folder is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesUpsertFolder, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}
func (*folderDeleteHandler) Exec(ctx context.Context, session xhandler.Session, input *FolderDeleteInput, output *Output) error {
	if input == nil || input.Value == nil || strings.TrimSpace(input.Value.FolderID) == "" {
		return publisherguard.PublicError(400, "folderId is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesDeleteFolder, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}
func (*skillUpsertHandler) Exec(ctx context.Context, session xhandler.Session, input *SkillUpsertInput, output *Output) error {
	if input == nil || input.Value == nil {
		return publisherguard.PublicError(400, "skill root is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesUpsertSkill, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}
func (*skillDeleteHandler) Exec(ctx context.Context, session xhandler.Session, input *SkillDeleteInput, output *Output) error {
	if input == nil || input.Value == nil || strings.TrimSpace(input.Value.SkillID) == "" {
		return publisherguard.PublicError(400, "skillId is required")
	}
	return mutate(ctx, session, input.identity, sdk.OperationResourcesDeleteSkill, input.Value.ReportID, input.Value.VersionNo, input.Value, output)
}

func mutate(ctx context.Context, session xhandler.Session, principal identity, operation, reportID string, versionNo int, input any, output *Output) error {
	if output == nil || principal.Jwt == nil || principal.Auth == nil || principal.Auth.Auth == nil ||
		principal.Jwt.Subject == "" || principal.Jwt.Subject != principal.Auth.Auth.Subject {
		return publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(reportID) == "" || versionNo <= 0 {
		return publisherguard.PublicError(400, "reportId and positive versionNo are required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("resource mutation session is required")
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
	access.SetJwt(principal.Jwt)
	access.SetAuth(principal.Auth)
	access.SetReportId(reportID)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard"), Input: access})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != reportID {
		return publisherguard.PublicError(403, "report edit access is required")
	}
	value, found, err = session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
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
	engine := &sqltransport.Transport{DB: db, ComponentInvoker: invoker}
	mutationCtx := sdk.WithPrincipal(ctx, sdk.Principal{Subject: principal.Jwt.Subject})
	if principal.NamespaceId != nil {
		mutationCtx = sdk.WithNamespaceSelection(mutationCtx, *principal.NamespaceId)
	}
	if err = engine.ApplyResourceMutation(mutationCtx, operation, input); err != nil {
		return publicMutationError(err)
	}
	read := &resourceget.Input{Jwt: principal.Jwt, Auth: principal.Auth, ReportId: reportID, VersionNo: versionNo}
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[resourceget.Component](), "resources", "POST", "/v1/studio/sdk/resources.get"), Input: read})
	if err != nil {
		return err
	}
	snapshot, ok := value.(*resourceget.Output)
	if !ok || snapshot == nil || snapshot.Version == nil || snapshot.Version.ReportID != reportID || snapshot.Version.VersionNo != versionNo {
		return fmt.Errorf("resource snapshot returned %T without matching version", value)
	}
	*output = Output{Files: snapshot.Files, Folders: snapshot.Folders, Skills: snapshot.Skills, Version: snapshot.Version}
	return nil
}

func publicMutationError(err error) error {
	var value *sdk.Error
	if !errors.As(err, &value) {
		return publisherguard.PublicError(502, "resource mutation failed")
	}
	status, message := 502, "resource mutation failed"
	switch value.Code {
	case sdk.ErrorInvalidArgument:
		status, message = 400, "resource mutation input is invalid"
	case sdk.ErrorNotFound:
		status, message = 404, "resource mutation target was not found"
	case sdk.ErrorForbidden:
		status, message = 403, "report edit access is required"
	case sdk.ErrorConflict:
		status, message = 409, "resource mutation conflicts with the current version"
	case sdk.ErrorUnavailable:
		status, message = 503, "resource mutation is unavailable"
	}
	payload := map[string]any{"message": message}
	if status == 409 {
		if value.ExpectedSourceRevision > 0 {
			payload["expectedSourceRevision"] = value.ExpectedSourceRevision
		}
		if value.CurrentSourceRevision > 0 {
			payload["currentSourceRevision"] = value.CurrentSourceRevision
		}
	}
	return &xresponse.Error{Code: status, Cause: errors.New(message), Payload: payload}
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
