package delete

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	acl "github.com/viant/datly-studio/studio/report_acl/reader"
	stored "github.com/viant/datly-studio/studio/report_acl/store_write"
	guard "github.com/viant/datly-studio/studio/reports/publish_guard"
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
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId    string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	SubjectType string             `parameter:"SubjectType,kind=body,in=subjectType,dataType=string,required=true" json:"subjectType"`
	SubjectId   string             `parameter:"SubjectId,kind=body,in=subjectId,dataType=string,required=true" json:"subjectId"`
	ETag        int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}

type Output struct{}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"acl,path=/v1/studio/sdk/acl.delete,method=POST,connector=studio,handler=NewACLDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.acl.delete\",\"description\":\"Delete an ACL grant owned by the verified report owner\"}]" caseFormat:"lc"`
}

var ACLDatly = new(Component)
var ACLDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewACLDelete" {
		return custom.Factory(NewACLDelete)
	}
	return nil
}

type deleteHandler struct{}

func NewACLDelete() xhandler.Contract[Input, Output] { return &deleteHandler{} }

func (*deleteHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "ACL management requires authenticated Studio access")
	}
	if strings.TrimSpace(input.ReportId) == "" {
		return publicError(400, "reportId is required")
	}
	if input.SubjectType != "user" || input.SubjectId == "" || len(input.SubjectId) > 128 {
		return publicError(400, "subjectType must be user and subjectId must be a verified JWT sub")
	}
	if session == nil || session.Binder() == nil || session.Response() == nil || output == nil {
		return fmt.Errorf("ACL delete handler session and output are required")
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
	access.SetReportId(input.ReportId)
	scoped, scopeErr := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/publish-guard"), Input: access})
	if scopeErr != nil {
		return scopeErr
	}
	allowed, ok := scoped.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != input.ReportId {
		return publicError(403, "Component is outside the selected namespace")
	}
	if allowed.Item.OwnerId == "" || allowed.Item.Namespace == "" {
		return fmt.Errorf("ACL component namespace is unavailable")
	}
	namespaceID := namespaceaccess.ID(allowed.Item.OwnerId, allowed.Item.Namespace)

	read := &acl.Input{}
	read.SetJwt(input.Jwt)
	read.SetReportId(input.ReportId)
	read.SetSubjectType(input.SubjectType)
	read.SetSubjectId(input.SubjectId)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[acl.AclComponent](), "acl", "POST", "/v1/studio/sdk/acl.list"), Input: read})
	if err != nil {
		return err
	}
	result, ok := value.(*acl.Output)
	if !ok || result == nil {
		return fmt.Errorf("owner ACL reader returned %T", value)
	}
	if len(result.Items) == 0 {
		return publicError(404, "ACL entry was not found")
	}
	if len(result.Items) != 1 || result.Items[0] == nil || result.Items[0].ReportId == nil ||
		*result.Items[0].ReportId != input.ReportId || result.Items[0].SubjectType == nil ||
		*result.Items[0].SubjectType != input.SubjectType || result.Items[0].SubjectId == nil ||
		*result.Items[0].SubjectId != input.SubjectId || result.Items[0].Etag == nil {
		return fmt.Errorf("owner ACL reader returned ambiguous or mismatched rows")
	}
	current := int64(*result.Items[0].Etag)
	if input.ETag <= 0 || input.ETag != current || input.ETag > math.MaxInt {
		return conflict(input.ETag, current)
	}
	etag := int(input.ETag)
	row := &stored.StoredACL{ReportId: &input.ReportId, SubjectType: &input.SubjectType,
		SubjectId: &input.SubjectId, Etag: &etag, ShouldDelete: true,
		Has: &stored.StoredACLHas{ReportId: true, SubjectType: true, SubjectId: true, Etag: true, ShouldDelete: true}}
	row.SetNamespaceId(namespaceID)
	write := &stored.Input{}
	write.SetAccess([]*stored.StoredACL{row})
	if _, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.AclComponent](), "acl", "PATCH", "/_studio/report-acl-store"), Input: write}); err != nil {
		var writerConflict *xhandler.Conflict
		if errors.As(err, &writerConflict) {
			return conflict(input.ETag, current)
		}
		return err
	}
	session.Response().SetStatusCode(http.StatusNoContent)
	return nil
}

func conflict(expected, current int64) error {
	message := "ACL etag does not match"
	return &xresponse.Error{Code: 409, Cause: errors.New(message), Payload: map[string]any{
		"message": message, "expectedEtag": expected, "currentEtag": current,
	}}
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
