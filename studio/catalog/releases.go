package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/viant/authz/oauth"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

var releaseSource struct {
	sync.RWMutex
	manager sdkrelease.Manager
	resolve func(context.Context, string) (sdkrelease.Manager, error)
}

// ConfigureReleaseManagement is a trusted startup hook. A nil manager keeps
// all operations unavailable. The manager derives account/tenant/workspace
// and editor authority from the verified ingress, never from request DTOs.
func ConfigureReleaseManagement(manager sdkrelease.Manager) {
	releaseSource.Lock()
	releaseSource.manager = manager
	releaseSource.resolve = nil
	releaseSource.Unlock()
}

// ConfigureReleaseManagementResolver dispatches an already visible, persisted
// namespace ID to its in-process manager. The native handler verifies the
// X-Studio-Namespace ID against the requested canonical name and caller first.
func ConfigureReleaseManagementResolver(resolve func(context.Context, string) (sdkrelease.Manager, error)) {
	releaseSource.Lock()
	releaseSource.manager = nil
	releaseSource.resolve = resolve
	releaseSource.Unlock()
}
func ReleaseManagementAvailable() bool {
	releaseSource.RLock()
	defer releaseSource.RUnlock()
	return releaseSource.manager != nil || releaseSource.resolve != nil
}

// ReleaseManagementForNamespace is a status-only capability probe after the
// ordinary SDK runtime-status operation has verified the selected namespace.
// It returns no manager or grant and never authorizes a release mutation.
func ReleaseManagementForNamespace(ctx context.Context, namespaceID string) bool {
	if ctx == nil || ctx.Err() != nil || namespaceID == "" {
		return false
	}
	releaseSource.RLock()
	manager, resolve := releaseSource.manager, releaseSource.resolve
	releaseSource.RUnlock()
	if resolve == nil {
		return manager != nil
	}
	selected, err := resolve(ctx, namespaceID)
	return err == nil && selected != nil
}

func releaseManager(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, namespaceID, name string) (context.Context, sdkrelease.Manager, error) {
	if claims == nil || claims.Subject == "" || auth == nil || auth.Auth == nil || auth.Auth.Subject != claims.Subject {
		return ctx, nil, fail(403, "verified Studio identity is required")
	}
	releaseSource.RLock()
	manager, resolve := releaseSource.manager, releaseSource.resolve
	releaseSource.RUnlock()
	if resolve != nil {
		if session == nil || session.Binder() == nil {
			return ctx, nil, fail(403, "verified Studio bearer is required")
		}
		var headers struct {
			Authorization string `bind:"kind=header,in=Authorization"`
		}
		if err := session.Binder().Bind(ctx, &headers); err != nil || !strings.HasPrefix(headers.Authorization, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(headers.Authorization, "Bearer ")) == "" {
			return ctx, nil, fail(403, "verified Studio bearer is required")
		}
		ctx = oauth.WithBearer(ctx, strings.TrimPrefix(headers.Authorization, "Bearer "))
		selected, err := selectedToolNamespace(ctx, session, claims, auth, namespaceID, name)
		if err != nil {
			return ctx, nil, fmt.Errorf("resolve release namespace: %w", err)
		}
		manager, err = resolve(ctx, selected)
		if err != nil {
			return ctx, nil, fmt.Errorf("resolve release manager: %w", err)
		}
	}
	if manager == nil {
		return ctx, nil, fail(503, "release management is unavailable")
	}
	return ctx, manager, nil
}

type ReleaseIdentity struct {
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	NamespaceID string             `parameter:"NamespaceID,kind=header,in=X-Studio-Namespace,dataType=string,required=false" json:"-"`
}
type ReleaseListInput struct {
	ReleaseIdentity
	Request contract.ListInput `parameter:"Request,kind=body,in=,dataType=artifactrelease.ListInput,required=true" anonymous:"true"`
}
type ReleaseCandidatesInput struct {
	ReleaseIdentity
	Request contract.CandidatesInput `parameter:"Request,kind=body,in=,dataType=artifactrelease.CandidatesInput,required=true" anonymous:"true"`
}
type ReleasePublishInput struct {
	ReleaseIdentity
	Request contract.PublishInput `parameter:"Request,kind=body,in=,dataType=artifactrelease.PublishInput,required=true" anonymous:"true"`
}
type ReleasePromoteInput struct {
	ReleaseIdentity
	Request contract.PromoteInput `parameter:"Request,kind=body,in=,dataType=artifactrelease.PromoteInput,required=true" anonymous:"true"`
}
type ReleaseHistoryInput struct {
	ReleaseIdentity
	Request contract.HistoryInput `parameter:"Request,kind=body,in=,dataType=artifactrelease.HistoryInput,required=true" anonymous:"true"`
}

type ReleaseListOutput struct {
	Response contract.ListOutput `parameter:"Response,kind=output,in=body,dataType=artifactrelease.ListOutput" json:"-"`
}
type ReleaseCandidatesOutput struct {
	Response contract.CandidatesOutput `parameter:"Response,kind=output,in=body,dataType=artifactrelease.CandidatesOutput" json:"-"`
}
type ReleaseResultOutput struct {
	Response contract.Result `parameter:"Response,kind=output,in=body,dataType=artifactrelease.Result" json:"-"`
}
type ReleaseHistoryOutput struct {
	Response contract.HistoryOutput `parameter:"Response,kind=output,in=body,dataType=artifactrelease.HistoryOutput" json:"-"`
}

func (*ReleaseListOutput) JSONWireType() reflect.Type     { return reflect.TypeFor[contract.ListOutput]() }
func (o *ReleaseListOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }
func (*ReleaseCandidatesOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[contract.CandidatesOutput]()
}
func (o *ReleaseCandidatesOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }
func (*ReleaseResultOutput) JSONWireType() reflect.Type         { return reflect.TypeFor[contract.Result]() }
func (o *ReleaseResultOutput) MarshalJSON() ([]byte, error)     { return json.Marshal(o.Response) }
func (*ReleaseHistoryOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[contract.HistoryOutput]()
}
func (o *ReleaseHistoryOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type ReleaseListComponent struct {
	Contract xdatly.Component[ReleaseListInput, ReleaseListOutput] `component:"release_list,path=/v1/studio/catalog/releases/list,method=POST,connector=studio,handler=NewReleaseList" mcp:"[{\"kind\":\"tool\",\"name\":\"releases/list\"}]" caseFormat:"lc"`
}
type ReleaseCandidatesComponent struct {
	Contract xdatly.Component[ReleaseCandidatesInput, ReleaseCandidatesOutput] `component:"release_candidates,path=/v1/studio/catalog/releases/candidates,method=POST,connector=studio,handler=NewReleaseCandidates" mcp:"[{\"kind\":\"tool\",\"name\":\"releases/candidates\"}]" caseFormat:"lc"`
}
type ReleasePublishComponent struct {
	Contract xdatly.Component[ReleasePublishInput, ReleaseResultOutput] `component:"release_publish,path=/v1/studio/catalog/releases/publish,method=POST,connector=studio,handler=NewReleasePublish" mcp:"[{\"kind\":\"tool\",\"name\":\"releases/publish\"}]" caseFormat:"lc"`
}
type ReleasePromoteComponent struct {
	Contract xdatly.Component[ReleasePromoteInput, ReleaseResultOutput] `component:"release_promote,path=/v1/studio/catalog/releases/promote,method=POST,connector=studio,handler=NewReleasePromote" mcp:"[{\"kind\":\"tool\",\"name\":\"releases/promote\"}]" caseFormat:"lc"`
}
type ReleaseHistoryComponent struct {
	Contract xdatly.Component[ReleaseHistoryInput, ReleaseHistoryOutput] `component:"release_history,path=/v1/studio/catalog/releases/history,method=POST,connector=studio,handler=NewReleaseHistory" mcp:"[{\"kind\":\"tool\",\"name\":\"releases/history\"}]" caseFormat:"lc"`
}

var ReleaseListDatlyLinkedType = reflect.TypeFor[ReleaseListComponent]()
var ReleaseCandidatesDatlyLinkedType = reflect.TypeFor[ReleaseCandidatesComponent]()
var ReleasePublishDatlyLinkedType = reflect.TypeFor[ReleasePublishComponent]()
var ReleasePromoteDatlyLinkedType = reflect.TypeFor[ReleasePromoteComponent]()
var ReleaseHistoryDatlyLinkedType = reflect.TypeFor[ReleaseHistoryComponent]()

func (ReleaseListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReleaseList" {
		return custom.Factory(NewReleaseList)
	}
	return nil
}
func (ReleaseCandidatesComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReleaseCandidates" {
		return custom.Factory(NewReleaseCandidates)
	}
	return nil
}
func (ReleasePublishComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReleasePublish" {
		return custom.Factory(NewReleasePublish)
	}
	return nil
}
func (ReleasePromoteComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReleasePromote" {
		return custom.Factory(NewReleasePromote)
	}
	return nil
}
func (ReleaseHistoryComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReleaseHistory" {
		return custom.Factory(NewReleaseHistory)
	}
	return nil
}

func NewReleaseList() xhandler.Contract[ReleaseListInput, ReleaseListOutput] {
	return &handler[ReleaseListInput, ReleaseListOutput]{run: runReleaseList}
}
func NewReleaseCandidates() xhandler.Contract[ReleaseCandidatesInput, ReleaseCandidatesOutput] {
	return &handler[ReleaseCandidatesInput, ReleaseCandidatesOutput]{run: runReleaseCandidates}
}
func NewReleasePublish() xhandler.Contract[ReleasePublishInput, ReleaseResultOutput] {
	return &handler[ReleasePublishInput, ReleaseResultOutput]{run: runReleasePublish}
}
func NewReleasePromote() xhandler.Contract[ReleasePromoteInput, ReleaseResultOutput] {
	return &handler[ReleasePromoteInput, ReleaseResultOutput]{run: runReleasePromote}
}
func NewReleaseHistory() xhandler.Contract[ReleaseHistoryInput, ReleaseHistoryOutput] {
	return &handler[ReleaseHistoryInput, ReleaseHistoryOutput]{run: runReleaseHistory}
}

func runReleaseList(ctx context.Context, session xhandler.Session, in *ReleaseListInput, out *ReleaseListOutput) (resultErr error) {
	defer func() { resultErr = releaseOperationError(ctx, "list", resultErr) }()
	if in == nil || out == nil || in.Request.Validate() != nil {
		return fail(400, "invalid release request")
	}
	ctx, manager, err := releaseManager(ctx, session, in.Jwt, in.Auth, in.NamespaceID, in.Request.Namespace)
	if err != nil {
		return err
	}
	out.Response, err = manager.List(ctx, in.Request)
	return releaseContractError(err)
}
func runReleaseCandidates(ctx context.Context, session xhandler.Session, in *ReleaseCandidatesInput, out *ReleaseCandidatesOutput) (resultErr error) {
	defer func() { resultErr = releaseOperationError(ctx, "candidates", resultErr) }()
	if in == nil || out == nil || in.Request.Validate() != nil {
		return fail(400, "invalid release request")
	}
	ctx, manager, err := releaseManager(ctx, session, in.Jwt, in.Auth, in.NamespaceID, in.Request.Namespace)
	if err != nil {
		return err
	}
	out.Response, err = manager.Candidates(ctx, in.Request)
	return releaseContractError(err)
}
func runReleasePublish(ctx context.Context, session xhandler.Session, in *ReleasePublishInput, out *ReleaseResultOutput) (resultErr error) {
	defer func() { resultErr = releaseOperationError(ctx, "publish", resultErr) }()
	if in == nil || out == nil || in.Request.Validate() != nil {
		return fail(400, "invalid release request")
	}
	ctx, manager, err := releaseManager(ctx, session, in.Jwt, in.Auth, in.NamespaceID, in.Request.Namespace)
	if err != nil {
		return err
	}
	out.Response, err = manager.Publish(ctx, in.Request)
	return releaseContractError(err)
}
func runReleasePromote(ctx context.Context, session xhandler.Session, in *ReleasePromoteInput, out *ReleaseResultOutput) (resultErr error) {
	defer func() { resultErr = releaseOperationError(ctx, "promote", resultErr) }()
	if in == nil || out == nil || in.Request.Validate() != nil {
		return fail(400, "invalid release request")
	}
	ctx, manager, err := releaseManager(ctx, session, in.Jwt, in.Auth, in.NamespaceID, in.Request.Namespace)
	if err != nil {
		return err
	}
	out.Response, err = manager.Promote(ctx, in.Request)
	return releaseContractError(err)
}
func runReleaseHistory(ctx context.Context, session xhandler.Session, in *ReleaseHistoryInput, out *ReleaseHistoryOutput) (resultErr error) {
	defer func() { resultErr = releaseOperationError(ctx, "history", resultErr) }()
	if in == nil || out == nil || in.Request.Validate() != nil {
		return fail(400, "invalid release request")
	}
	ctx, manager, err := releaseManager(ctx, session, in.Jwt, in.Auth, in.NamespaceID, in.Request.Namespace)
	if err != nil {
		return err
	}
	out.Response, err = manager.History(ctx, in.Request)
	return releaseContractError(err)
}

// Expected release contract failures retain their recoverable wire status.
func releaseContractError(err error) error {
	switch {
	case errors.Is(err, contract.ErrConflict):
		return fail(409, "Release changed. Reload and try again.")
	case errors.Is(err, contract.ErrInvalid):
		return fail(400, "Invalid release request.")
	default:
		return err
	}
}

// Preserve actionable client errors while retaining unexpected causes in server logs.
func releaseOperationError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	var wire *xresponse.Error
	if errors.As(err, &wire) && wire.Code < 500 {
		return wire
	}
	slog.ErrorContext(ctx, "release operation failed", "operation", operation, "error", err)
	if errors.As(err, &wire) {
		return wire
	}
	return fail(500, "Release "+operation+" failed. See the server log for the underlying error.")
}
