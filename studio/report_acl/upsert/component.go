package upsert

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	acl "github.com/viant/datly-studio/studio/report_acl/reader"
	stored "github.com/viant/datly-studio/studio/report_acl/store_write"
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

type Input struct {
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId    string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	SubjectType string             `parameter:"SubjectType,kind=body,in=subjectType,dataType=string,required=true" json:"subjectType"`
	SubjectId   string             `parameter:"SubjectId,kind=body,in=subjectId,dataType=string,required=true" json:"subjectId"`
	CanView     bool               `parameter:"CanView,kind=body,in=canView,dataType=bool" json:"canView"`
	CanRun      bool               `parameter:"CanRun,kind=body,in=canRun,dataType=bool" json:"canRun"`
	CanEdit     bool               `parameter:"CanEdit,kind=body,in=canEdit,dataType=bool" json:"canEdit"`
	CanPublish  bool               `parameter:"CanPublish,kind=body,in=canPublish,dataType=bool" json:"canPublish"`
	CanUseDql   bool               `parameter:"CanUseDql,kind=body,in=canUseDql,dataType=bool" json:"canUseDql"`
	ETag        int64              `parameter:"ETag,kind=body,in=etag,dataType=int64" json:"etag,omitempty"`
}

type Output struct {
	ReportId    string `parameter:"ReportId,kind=output,in=body,dataType=string" json:"reportId"`
	SubjectType string `parameter:"SubjectType,kind=output,in=body,dataType=string" json:"subjectType"`
	SubjectId   string `parameter:"SubjectId,kind=output,in=body,dataType=string" json:"subjectId"`
	CanView     bool   `parameter:"CanView,kind=output,in=body,dataType=bool" json:"canView"`
	CanRun      bool   `parameter:"CanRun,kind=output,in=body,dataType=bool" json:"canRun"`
	CanEdit     bool   `parameter:"CanEdit,kind=output,in=body,dataType=bool" json:"canEdit"`
	CanPublish  bool   `parameter:"CanPublish,kind=output,in=body,dataType=bool" json:"canPublish"`
	CanUseDql   bool   `parameter:"CanUseDql,kind=output,in=body,dataType=bool" json:"canUseDql"`
	ETag        int64  `parameter:"ETag,kind=output,in=body,dataType=int64" json:"etag"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"acl,path=/v1/studio/sdk/acl.upsert,method=POST,connector=studio,handler=NewACLUpsert" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.acl.upsert\",\"description\":\"Create or update an ACL grant owned by the verified report owner\"}]" caseFormat:"lc"`
}

var ACLDatly = new(Component)
var ACLDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewACLUpsert" {
		return custom.Factory(NewACLUpsert)
	}
	return nil
}

type upsertHandler struct{}

func NewACLUpsert() xhandler.Contract[Input, Output] { return &upsertHandler{} }

func (*upsertHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
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
	if (input.CanRun || input.CanEdit || input.CanPublish || input.CanUseDql) && !input.CanView {
		return publicError(400, "view capability is required for run, edit, publish, or DQL access")
	}
	if input.CanUseDql && !input.CanEdit {
		return publicError(400, "edit capability is required for advanced DQL access")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("ACL upsert handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
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
	if !ok || result == nil || len(result.Items) > 1 {
		return fmt.Errorf("owner ACL reader returned %T with unexpected rows", value)
	}
	expected := int64(0)
	if len(result.Items) == 1 {
		item := result.Items[0]
		if item == nil || item.ReportId == nil || *item.ReportId != input.ReportId || item.SubjectType == nil ||
			*item.SubjectType != input.SubjectType || item.SubjectId == nil || *item.SubjectId != input.SubjectId || item.Etag == nil {
			return fmt.Errorf("owner ACL reader returned a mismatched row")
		}
		expected = int64(*item.Etag)
		if input.ETag <= 0 || input.ETag != expected || input.ETag > math.MaxInt {
			return conflict(input.ETag, expected)
		}
	}
	etag := int(input.ETag)
	if expected == 0 {
		etag = 1
	}
	canView, canRun, canEdit := bit(input.CanView), bit(input.CanRun), bit(input.CanEdit)
	canPublish, canUseDQL := bit(input.CanPublish), bit(input.CanUseDql)
	row := &stored.StoredACL{ReportId: &input.ReportId, SubjectType: &input.SubjectType, SubjectId: &input.SubjectId,
		CanView: &canView, CanRun: &canRun, CanEdit: &canEdit, CanPublish: &canPublish, CanUseDql: &canUseDQL,
		Etag: &etag, Has: &stored.StoredACLHas{ReportId: true, SubjectType: true, SubjectId: true,
			CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDql: true, Etag: true}}
	write := &stored.Input{}
	write.SetAccess([]*stored.StoredACL{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.AclComponent](), "acl", "PATCH", "/_studio/report-acl-store"), Input: write})
	if err != nil {
		var writerConflict *xhandler.Conflict
		if errors.As(err, &writerConflict) || errors.Is(err, errx.ErrDuplicateKey) {
			return conflict(input.ETag, expected)
		}
		return err
	}
	mutated, ok := value.(*stored.Output)
	if !ok || mutated == nil || len(mutated.Data) != 1 || mutated.Data[0] == nil || mutated.Data[0].Etag == nil {
		return fmt.Errorf("ACL writer returned %T without one upserted row", value)
	}
	*output = Output{ReportId: input.ReportId, SubjectType: input.SubjectType, SubjectId: input.SubjectId,
		CanView: input.CanView, CanRun: input.CanRun, CanEdit: input.CanEdit,
		CanPublish: input.CanPublish, CanUseDql: input.CanUseDql, ETag: int64(*mutated.Data[0].Etag)}
	return nil
}

func bit(value bool) int {
	if value {
		return 1
	}
	return 0
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
