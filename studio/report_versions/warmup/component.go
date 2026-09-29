package warmup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/internal/versionprojection"
	"github.com/viant/datly-studio/internal/warmupprojection"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	catalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	warmuplist "github.com/viant/datly-studio/studio/report_warmup_runs/list"
	storedreader "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
	storedwriter "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	guard "github.com/viant/datly-studio/studio/reports/publish_guard"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

const modulePath = "github.com/viant/datly-studio"
const runTimeout = 5 * time.Minute

type Input struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID    string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo   int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
}

type Output struct {
	Response *sdk.WarmupRun `parameter:"Response,kind=output,in=body,dataType=*sdk.WarmupRun" json:"-"`
}

func (*Output) JSONWireType() reflect.Type { return reflect.TypeFor[sdk.WarmupRun]() }

func (output *Output) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.warmup,method=POST,connector=studio,handler=NewWarmup" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.warmup\",\"description\":\"Accept one durable authorized Datly cache warmup run\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewWarmup" {
		return custom.Factory(NewWarmup)
	}
	return nil
}

type handler struct{}

func NewWarmup() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(input.ReportID) == "" || input.VersionNo <= 0 {
		return publisherguard.PublicError(400, "reportId and positive versionNo are required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("warmup handler session is required")
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
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/publish-guard"), Input: access})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != input.ReportID {
		return publisherguard.PublicError(404, "report not found")
	}
	if allowed.Item.OwnerId == "" || allowed.Item.Namespace == "" {
		return fmt.Errorf("warmup component ownership is unavailable")
	}
	namespaceID := namespaceaccess.ID(allowed.Item.OwnerId, allowed.Item.Namespace)
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
		return fmt.Errorf("warmup version reader returned %T", value)
	}
	if len(listed.Versions) == 0 {
		return publisherguard.PublicError(404, "report version not found")
	}
	if len(listed.Versions) != 1 || listed.Versions[0] == nil || listed.Versions[0].ReportId != input.ReportID || listed.Versions[0].VersionNo != input.VersionNo {
		return fmt.Errorf("warmup version reader returned ambiguous identity")
	}
	version := versionprojection.FromCatalog(listed.Versions[0])
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
	// Reuse the selected native list workflow's system-owned expiry recovery
	// before testing the unique active-key for this exact source revision.
	recovery := &warmuplist.Input{Jwt: input.Jwt, Auth: input.Auth, ReportId: input.ReportID, VersionNo: input.VersionNo,
		Input: warmuplist.Options{Limit: 1}}
	if _, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[warmuplist.Component](), "warmup_run", "POST", "/v1/studio/sdk/versions.warmup_list"), Input: recovery}); err != nil {
		return err
	}
	planKey := warmupprojection.PlanKey(version)
	activeKey := fmt.Sprintf("%s:%d:%s", input.ReportID, input.VersionNo, planKey)
	active, err := readRun(ctx, invoker, input.ReportID, "", activeKey)
	if err != nil {
		return err
	}
	if active != nil {
		output.Response = active
		return nil
	}
	runID, err := newRunID()
	if err != nil {
		return err
	}
	now, actor := time.Now().UTC(), input.Jwt.Subject
	row := &storedwriter.StoredWarmupRun{NamespaceId: namespaceID, RunId: runID, ReportId: input.ReportID,
		VersionNo: input.VersionNo, SourceRevision: version.SourceRevision, SpecHash: version.SpecHash,
		PlanKey: planKey, ActiveKey: &activeKey, Status: "accepted", RequestedBy: actor,
		TargetJson: json.RawMessage(`{}`), RequestedAt: now, CreatedAt: &now, CreatedBy: &actor,
		UpdatedAt: &now, UpdatedBy: &actor,
		Has: &storedwriter.StoredWarmupRunHas{NamespaceId: true, RunId: true, ReportId: true, VersionNo: true,
			SourceRevision: true, SpecHash: true, PlanKey: true, ActiveKey: true,
			Status: true, RequestedBy: true, TargetJson: true, RequestedAt: true,
			CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true}}
	write := &storedwriter.Input{}
	write.SetRuns([]*storedwriter.StoredWarmupRun{row})
	if _, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedwriter.WarmupRunComponent](), "warmup_run", "PATCH", "/_studio/report-warmup-run-store/write"), Input: write}); err != nil {
		if winner, readErr := readRun(ctx, invoker, input.ReportID, "", activeKey); readErr == nil && winner != nil {
			output.Response = winner
			return nil
		}
		return err
	}
	// The outer custom-handler transaction commits after Exec returns. Project
	// the accepted response from server-owned input and let the worker read its
	// durable update token only after that commit becomes visible.
	output.Response = &sdk.WarmupRun{RunID: runID, ReportID: input.ReportID, VersionNo: input.VersionNo,
		SourceRevision: version.SourceRevision, SpecHash: version.SpecHash, PlanKey: planKey,
		Status: "accepted", RequestedBy: actor, RequestedAt: now,
		CreatedAt: &now, CreatedBy: &actor, UpdatedAt: &now, UpdatedBy: &actor}
	reportID, versionNo := input.ReportID, input.VersionNo
	go func() {
		jobCtx, cancel := context.WithTimeout(sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: actor}), runTimeout)
		defer cancel()
		worker := &sqltransport.Transport{DB: db, Warmup: preview.Dynamic{StudioDB: db, ModulePath: modulePath}}
		visibleUntil := time.Now().Add(5 * time.Second)
		for time.Now().Before(visibleUntil) && jobCtx.Err() == nil {
			committed, readErr := worker.ReadAcceptedWarmupRun(jobCtx, reportID, runID)
			if readErr == nil && committed != nil && committed.UpdatedAt != nil {
				worker.ExecuteAcceptedWarmupRun(jobCtx, runID, reportID, versionNo, *committed.UpdatedAt)
				return
			}
			// A pending commit or SQLite lock is transient here; retry until the
			// bounded visibility deadline and let expiry recovery handle a crash.
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return nil
}

func readRun(ctx context.Context, invoker exec.ComponentInvoker, reportID, runID, activeKey string) (*sdk.WarmupRun, error) {
	input := &storedreader.Input{}
	input.SetReportId(reportID)
	input.SetRunId(runID)
	input.SetActiveKey(activeKey)
	input.SetVersionNo(0)
	input.SetPageLimit(2)
	input.SetPageOffset(0)
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[storedreader.WarmupRunComponent](), "warmup_run", "GET", "/_studio/report-warmup-run-store/read"), Input: input})
	if err != nil {
		return nil, err
	}
	page, ok := value.(*storedreader.Output)
	if !ok || page == nil || len(page.WarmupRuns) > 1 {
		return nil, fmt.Errorf("warmup run reader returned %T with invalid cardinality", value)
	}
	if len(page.WarmupRuns) == 0 {
		return nil, nil
	}
	runs, err := warmupprojection.Runs(reportID, runID, 0, page.WarmupRuns)
	if err != nil {
		return nil, err
	}
	return runs[0], nil
}

func newRunID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "w" + hex.EncodeToString(value), nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
