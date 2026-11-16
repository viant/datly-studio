// Package componentremoval owns the shared native component/version removal operation.
package componentremoval

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/reports/store_removal"
	snapshot "github.com/viant/datly-studio/studio/reports/store_removal_snapshot"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// Request is server-owned execution metadata after native credential binding.
type Request struct {
	NamespaceID, Actor, ComponentID string
	VersionNo                       int
	ExpectedSourceRevision, ETag    int64
}

func Execute(ctx context.Context, session xhandler.Session, request Request) (*sdk.DeletionResult, error) {
	if session == nil || session.Binder() == nil {
		return nil, fmt.Errorf("native removal session is unavailable")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, fmt.Errorf("native component invoker is unavailable")
	}
	value, found, err = session.Binder().Lookup(ctx, xhandler.TransactionStarterKey)
	if err != nil {
		return nil, err
	}
	starter, ok := value.(xhandler.TransactionStarter)
	if !found || !ok {
		return nil, fmt.Errorf("managed removal transaction is unavailable")
	}
	if err = starter.Start(ctx); err != nil {
		return nil, err
	}
	input := &snapshot.Input{}
	input.SetReportId(request.ComponentID)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[snapshot.ReportComponent](), "report", "GET", "/_studio/reports/store-removal-snapshot"), Input: input, ReaderOptions: &exec.ReaderOptions{ForUpdate: []string{exec.RootView}}})
	if err != nil {
		return nil, err
	}
	observed, ok := value.(*snapshot.Output)
	if !ok || observed == nil {
		return nil, fmt.Errorf("removal snapshot returned %T", value)
	}
	if len(observed.Reports) != 1 || observed.Reports[0] == nil {
		return nil, publisherguard.PublicError(404, "Component not found")
	}
	current := observed.Reports[0]
	if current.Id != request.ComponentID || current.NamespaceId != request.NamespaceID || current.OwnerId != request.Actor {
		return nil, publisherguard.PublicError(403, "Only the component owner can permanently delete it or its versions")
	}
	if current.Etag == nil {
		return nil, fmt.Errorf("component has no concurrency token")
	}
	if request.VersionNo == 0 && request.ETag != *current.Etag {
		return nil, publisherguard.PublicError(409, "Component configuration changed; reload before deleting")
	}
	now := time.Now().UTC()
	row := &stored.RemovalReport{Id: current.Id, NamespaceId: current.NamespaceId, OwnerId: current.OwnerId, Etag: current.Etag, CurrentDraftVersion: current.CurrentDraftVersion, UpdatedAt: &now, ShouldDelete: request.VersionNo == 0, Has: &stored.RemovalReportHas{Id: true, NamespaceId: true, OwnerId: true, Etag: true, CurrentDraftVersion: true, UpdatedAt: true, ShouldDelete: true, Version: true, Deletion: true}}
	for _, version := range current.Version {
		if version == nil || request.VersionNo != 0 && version.VersionNo != request.VersionNo {
			continue
		}
		if request.VersionNo > 0 && (version.SourceRevision == nil || *version.SourceRevision != request.ExpectedSourceRevision) {
			return nil, publisherguard.PublicError(409, "Version source revision changed; reload before deleting")
		}
		selected := &stored.RemovalVersion{ReportId: version.ReportId, NamespaceId: version.NamespaceId, VersionNo: version.VersionNo, SourceRevision: version.SourceRevision, ShouldDelete: true, Has: &stored.RemovalVersionHas{ReportId: true, NamespaceId: true, VersionNo: true, SourceRevision: true, ShouldDelete: true, Warmup: true}}
		for _, run := range version.Warmup {
			if run == nil {
				continue
			}
			if run.Status == "accepted" || run.Status == "running" {
				return nil, publisherguard.PublicError(409, fmt.Sprintf("Version %d has active warmup work; wait for it to finish before deleting", version.VersionNo))
			}
			selected.Warmup = append(selected.Warmup, &stored.RemovalWarmup{RunId: run.RunId, ReportId: run.ReportId, VersionNo: run.VersionNo, Status: run.Status, ShouldDelete: true, Has: &stored.RemovalWarmupHas{RunId: true, ReportId: true, VersionNo: true, Status: true, ShouldDelete: true}})
		}
		row.Version = append(row.Version, selected)
	}
	if request.VersionNo > 0 && len(row.Version) != 1 {
		return nil, publisherguard.PublicError(404, "Version not found")
	}
	if request.VersionNo > 0 && row.CurrentDraftVersion != nil && *row.CurrentDraftVersion == request.VersionNo {
		row.CurrentDraftVersion = nil
	}
	if request.VersionNo == 0 {
		row.Deletion = []*stored.RemovalReceipt{{ComponentId: current.Id, NamespaceId: current.NamespaceId, OwnerId: current.OwnerId, DeletedAt: now, DeletedBy: request.Actor, Has: &stored.RemovalReceiptHas{ComponentId: true, NamespaceId: true, OwnerId: true, DeletedAt: true, DeletedBy: true}}}
	}
	write := &stored.Input{}
	write.SetReportId(request.ComponentID)
	write.SetActor(request.Actor)
	write.SetNamespaceId(request.NamespaceID)
	write.SetTargetVersion(request.VersionNo)
	write.SetReports([]*stored.RemovalReport{row})
	_, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ReportComponent](), "report", "PATCH", "/_studio/reports/store-removal"), Input: write})
	if err != nil {
		var public *xresponse.Error
		if errors.As(err, &public) {
			return nil, err
		}
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return nil, publisherguard.PublicError(409, conflict.Reason)
		}
		return nil, err
	}
	return &sdk.DeletionResult{ComponentID: current.Id, VersionNo: request.VersionNo, DeletedVersions: len(row.Version)}, nil
}
func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
