package list

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/warmupprojection"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	expired "github.com/viant/datly-studio/studio/report_warmup_runs/store_expired"
	storedreader "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
	storedwriter "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
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

const warmupRunTimeout = 5 * time.Minute

type Options struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type Input struct {
	Jwt       *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId  string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Input     Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=false" json:"input"`
}

type Output struct {
	Items  []*sdk.WarmupRun `parameter:"Items,kind=output,in=body,dataType=[]*sdk.WarmupRun" json:"items"`
	Limit  int              `parameter:"Limit,kind=output,in=body,dataType=int" json:"limit"`
	Offset int              `parameter:"Offset,kind=output,in=body,dataType=int" json:"offset"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"warmup_run,path=/v1/studio/sdk/versions.warmup_list,method=POST,connector=studio,handler=NewWarmupList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.warmup_list\",\"description\":\"List authorized Datly Studio warmup runs after recovering expired runs\"}]" caseFormat:"lc"`
}

var WarmupListDatly = new(Component)
var WarmupListDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewWarmupList" {
		return custom.Factory(NewWarmupList)
	}
	return nil
}

type listHandler struct{}

func NewWarmupList() xhandler.Contract[Input, Output] { return &listHandler{} }

func (*listHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		strings.TrimSpace(input.ReportId) == "" || input.VersionNo <= 0 || input.Input.Offset < 0 {
		return &xresponse.Error{Code: 400, Cause: errors.New("reportId, versionNo, and non-negative offset are required")}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("warmup list handler session and output are required")
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
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/publish-guard"), Input: accessInput})
	if err != nil {
		return err
	}
	access, ok := value.(*guard.Output)
	if !ok || access == nil {
		return fmt.Errorf("publish guard returned %T", value)
	}
	if access.Item == nil || access.Item.Id != input.ReportId {
		return &xresponse.Error{Code: 404, Cause: errors.New("report not found")}
	}
	if err = recoverExpired(ctx, invoker, time.Now().UTC()); err != nil {
		return err
	}
	limit := input.Input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	page := &storedreader.Input{}
	page.SetReportId(input.ReportId)
	page.SetRunId("")
	page.SetActiveKey("")
	page.SetVersionNo(input.VersionNo)
	page.SetPageLimit(limit)
	page.SetPageOffset(input.Input.Offset)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedreader.WarmupRunComponent](), "warmup_run", "GET", "/_studio/report-warmup-run-store/read"), Input: page})
	if err != nil {
		return err
	}
	read, ok := value.(*storedreader.Output)
	if !ok || read == nil {
		return fmt.Errorf("warmup reader returned %T", value)
	}
	if len(read.WarmupRuns) > limit {
		return fmt.Errorf("warmup reader exceeded page limit")
	}
	output.Items, err = warmupprojection.Runs(input.ReportId, "", input.VersionNo, read.WarmupRuns)
	if err != nil {
		return err
	}
	output.Limit, output.Offset = limit, input.Input.Offset
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func recoverExpired(ctx context.Context, invoker exec.ComponentInvoker, now time.Time) error {
	diagnostics, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "warmup_expired", Message: "cache warmup did not reach a terminal state before the server-owned timeout"}})
	message := json.RawMessage(diagnostics)
	actor := sdk.SystemPrincipal().Subject
	for {
		request := &expired.Input{}
		request.SetBefore(now.Add(-warmupRunTimeout))
		request.SetPageLimit(100)
		value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[expired.WarmupRunComponent](), "warmup_run", "GET", "/_studio/report-warmup-run-store/expired"), Input: request})
		if err != nil {
			return err
		}
		page, ok := value.(*expired.Output)
		if !ok || page == nil {
			return fmt.Errorf("expired warmup reader returned %T", value)
		}
		if len(page.WarmupRuns) == 0 {
			return nil
		}
		if len(page.WarmupRuns) > 100 {
			return fmt.Errorf("expired warmup reader exceeded page limit")
		}
		changed := 0
		for _, row := range page.WarmupRuns {
			if row == nil || row.RunId == "" || row.UpdatedAt == nil {
				return fmt.Errorf("expired warmup reader returned an incomplete row")
			}
			write := &storedwriter.Input{}
			write.SetRuns([]*storedwriter.StoredWarmupRun{{RunId: row.RunId, Status: "failed",
				UpdatedAt: row.UpdatedAt, UpdatedBy: &actor, DiagnosticsJson: &message,
				CompletedAt: &now,
				Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true, ActiveKey: true,
					UpdatedAt: true, UpdatedBy: true, DiagnosticsJson: true, CompletedAt: true}}})
			_, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedwriter.WarmupRunComponent](), "warmup_run", "PATCH", "/_studio/report-warmup-run-store/write"), Input: write})
			if err == nil {
				changed++
				continue
			}
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				continue
			}
			return err
		}
		if changed == 0 {
			return nil
		}
	}
}
