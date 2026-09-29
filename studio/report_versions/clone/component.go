// Package clone implements an authorized, policy-preserving draft copy through
// linked Datly components inside the caller's managed Studio transaction.
package clone

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	policy "github.com/viant/authz/datly/store/sql"
	"github.com/viant/datly-studio/internal/versionclone"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/store/sql/accesscatalog"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	create "github.com/viant/datly-studio/studio/report_versions/create"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	resources "github.com/viant/datly-studio/studio/resources/get"
	mutate "github.com/viant/datly-studio/studio/resources/mutate"
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

type Input struct {
	NamespaceId            *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=true" json:"namespaceId,omitempty"`
	Jwt                    *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,required=true,errorCode=401" codec:"JwtClaim"`
	Auth                   *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID               string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo              int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	ExpectedSourceRevision int64              `parameter:"ExpectedSourceRevision,kind=body,in=expectedSourceRevision,dataType=int64,required=true" json:"expectedSourceRevision"`
}
type Output struct {
	Response *sdk.ReportVersion `parameter:"Response,kind=output,in=body,dataType=*sdk.ReportVersion" json:"-"`
}

func (*Output) JSONWireType() reflect.Type     { return reflect.TypeFor[sdk.ReportVersion]() }
func (o *Output) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"clone,path=/v1/studio/sdk/versions.clone,method=POST,connector=studio,handler=NewClone" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.clone\",\"description\":\"Clone an authorized exact version with resources and unchanged access policies\"}]" caseFormat:"lc"`
}

var CloneDatly = new(Component)
var CloneDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewClone" {
		return custom.Factory(NewClone)
	}
	return nil
}

type handler struct{}

func NewClone() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, in *Input, out *Output) error {
	if in == nil || out == nil || in.Jwt == nil || in.Auth == nil || in.Auth.Auth == nil || in.Jwt.Subject == "" || in.Jwt.Subject != in.Auth.Auth.Subject {
		return publicError(403, "Verified Studio principal is required")
	}
	if in.NamespaceId == nil || len(*in.NamespaceId) != 64 || strings.TrimSpace(in.ReportID) == "" || in.VersionNo < 1 || in.ExpectedSourceRevision < 1 {
		return publicError(400, "Namespace, source version and revision are required")
	}
	if decoded, err := hex.DecodeString(*in.NamespaceId); err != nil || len(decoded) != 32 || *in.NamespaceId != strings.ToLower(*in.NamespaceId) {
		return publicError(400, "A valid namespace is required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("clone session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("clone component invoker is unavailable")
	}
	edit := &guard.Input{}
	edit.SetJwt(in.Jwt)
	edit.SetAuth(in.Auth)
	edit.SetReportId(in.ReportID)
	value, err = invoke(ctx, invoker, reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard", edit)
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != in.ReportID {
		return publicError(403, "Component authoring permission is required")
	}
	value, found, err = session.Binder().Lookup(ctx, xhandler.TransactionStarterKey)
	if err != nil {
		return err
	}
	starter, ok := value.(xhandler.TransactionStarter)
	if !found || !ok {
		return fmt.Errorf("managed clone transaction is unavailable")
	}
	if err = starter.Start(ctx); err != nil {
		return err
	}
	sourceInput := &resources.Input{NamespaceId: in.NamespaceId, Jwt: in.Jwt, Auth: in.Auth, ReportId: in.ReportID, VersionNo: in.VersionNo}
	value, err = invoke(ctx, invoker, reflect.TypeFor[resources.Component](), "resources", "POST", "/v1/studio/sdk/resources.get", sourceInput)
	if err != nil {
		return err
	}
	source, ok := value.(*resources.Output)
	if !ok || source == nil || source.Version == nil || source.Version.ReportID != in.ReportID || source.Version.VersionNo != in.VersionNo {
		return fmt.Errorf("clone source snapshot is invalid")
	}
	if source.Version.SourceRevision != in.ExpectedSourceRevision {
		return publicError(409, "Source version changed. Reload before cloning.")
	}
	dql := source.Version.AuthoredDQL
	if dql == "" {
		dql = source.Version.GeneratedDQL
	}
	if dql == "" {
		return publicError(403, "Source DQL authoring access is required")
	}
	value, found, err = session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if err != nil {
		return err
	}
	connectors, ok := value.(xconnector.Provider)
	if !found || !ok {
		return fmt.Errorf("Studio connector capability is unavailable")
	}
	db, err := connectors.Connector(ctx, "studio")
	if err != nil {
		return err
	}
	store := &policy.Store{DB: db, Invoker: invoker}
	policies, err := versionclone.Snapshot(ctx, &accesscatalog.Store{DB: db, Invoker: invoker}, store, in.ReportID, *in.NamespaceId, in.VersionNo)
	if err != nil {
		return publicError(409, "Source access policies are unavailable or invalid")
	}
	createInput := &create.Input{NamespaceId: in.NamespaceId, Jwt: in.Jwt, Auth: in.Auth, ReportId: in.ReportID, Input: create.Options{AuthoringMode: source.Version.AuthoringMode, AuthoredDQL: dql, AuthoredSQL: source.Version.AuthoredSQL, ComponentSpec: source.Version.ComponentSpec, Notes: fmt.Sprintf("Draft from v%d revision %d; access policies preserved", in.VersionNo, in.ExpectedSourceRevision)}}
	value, err = invoke(ctx, invoker, reflect.TypeFor[create.Component](), "version", "POST", "/v1/studio/sdk/versions.create", createInput)
	if err != nil {
		return err
	}
	created, ok := value.(*create.Output)
	if !ok || created == nil || created.ReportId != in.ReportID || created.VersionNo <= in.VersionNo {
		return fmt.Errorf("clone target version is invalid")
	}
	encoded, err := json.Marshal(created)
	if err != nil {
		return err
	}
	latest := new(sdk.ReportVersion)
	if err = json.Unmarshal(encoded, latest); err != nil {
		return err
	}
	copyResource := func(holder reflect.Type, name, path string, input any) error {
		value, err := invoke(ctx, invoker, holder, name, "POST", path, input)
		if err != nil {
			return err
		}
		result, ok := value.(*mutate.Output)
		if !ok || result == nil || result.Version == nil || result.Version.ReportID != in.ReportID || result.Version.VersionNo != created.VersionNo || result.Version.SourceRevision != latest.SourceRevision+1 {
			return fmt.Errorf("clone resource result is invalid")
		}
		latest = result.Version
		return nil
	}
	for _, entry := range source.Files {
		if entry == nil {
			return fmt.Errorf("clone file is invalid")
		}
		file := *entry
		file.VersionNo = created.VersionNo
		file.ExpectedSourceRevision = latest.SourceRevision
		input := &mutate.FileUpsertInput{Value: &file}
		input.Jwt = in.Jwt
		input.Auth = in.Auth
		input.NamespaceId = in.NamespaceId
		if err := copyResource(reflect.TypeFor[mutate.FileUpsertComponent](), "resource_upsert_file", "/v1/studio/sdk/resources.upsert_file", input); err != nil {
			return err
		}
	}
	for _, entry := range source.Folders {
		if entry == nil {
			return fmt.Errorf("clone folder is invalid")
		}
		folder := *entry
		folder.VersionNo = created.VersionNo
		folder.ExpectedSourceRevision = latest.SourceRevision
		input := &mutate.FolderUpsertInput{Value: &folder}
		input.Jwt = in.Jwt
		input.Auth = in.Auth
		input.NamespaceId = in.NamespaceId
		if err := copyResource(reflect.TypeFor[mutate.FolderUpsertComponent](), "resource_upsert_folder", "/v1/studio/sdk/resources.upsert_folder", input); err != nil {
			return err
		}
	}
	for _, entry := range source.Skills {
		if entry == nil {
			return fmt.Errorf("clone skill is invalid")
		}
		skill := *entry
		skill.VersionNo = created.VersionNo
		skill.ExpectedSourceRevision = latest.SourceRevision
		input := &mutate.SkillUpsertInput{Value: &skill}
		input.Jwt = in.Jwt
		input.Auth = in.Auth
		input.NamespaceId = in.NamespaceId
		if err := copyResource(reflect.TypeFor[mutate.SkillUpsertComponent](), "resource_upsert_skill", "/v1/studio/sdk/resources.upsert_skill", input); err != nil {
			return err
		}
	}
	if err = versionclone.Preserve(ctx, store, policies, created.VersionNo, in.Jwt.Subject); err != nil {
		return publicError(409, "Draft access policy copy failed")
	}
	out.Response = latest
	return nil
}
func invoke(ctx context.Context, invoker exec.ComponentInvoker, holder reflect.Type, name, method, path string, input any) (any, error) {
	return invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}, Input: input})
}
func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
