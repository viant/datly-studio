package sdk

import (
	"context"
	"time"
)

const (
	OperationWarmupScheduleGet    = "versions.warmup_schedule_get"
	OperationWarmupScheduleSave   = "versions.warmup_schedule_save"
	OperationWarmupScheduleDelete = "versions.warmup_schedule_delete"
)

// WarmupSchedule follows the saved content of an exact numbered version.
// LastRunID is the latest admitted or coalesced run, not occurrence history.
type WarmupSchedule struct {
	ScheduleID      string    `json:"scheduleId"`
	NamespaceID     string    `json:"namespaceId"`
	ReportID        string    `json:"reportId"`
	VersionNo       int       `json:"versionNo"`
	IntervalSeconds int64     `json:"intervalSeconds"`
	FirstRunAt      time.Time `json:"firstRunAt"`
	NextDueAt       time.Time `json:"nextDueAt"`
	Enabled         bool      `json:"enabled"`
	Revision        int64     `json:"revision"`
	LastRunID       *string   `json:"lastRunId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	CreatedBy       string    `json:"createdBy"`
	UpdatedAt       time.Time `json:"updatedAt"`
	UpdatedBy       string    `json:"updatedBy"`
}
type SaveWarmupScheduleInput struct {
	ExpectedRevision int64     `json:"expectedRevision"`
	IntervalSeconds  int64     `json:"intervalSeconds"`
	FirstRunAt       time.Time `json:"firstRunAt"`
	Enabled          bool      `json:"enabled"`
}
type DeleteWarmupScheduleInput struct {
	ExpectedRevision int64 `json:"expectedRevision"`
}
type WarmupScheduleResult struct {
	Schedule *WarmupSchedule `json:"schedule,omitempty"`
}

func (c versionClient) GetWarmupSchedule(ctx context.Context, id string, v int) (*WarmupScheduleResult, error) {
	return invoke[WarmupScheduleResult](ctx, c.transport, OperationWarmupScheduleGet, versionIdentity{id, v})
}
func (c versionClient) SaveWarmupSchedule(ctx context.Context, id string, v int, input SaveWarmupScheduleInput) (*WarmupScheduleResult, error) {
	return invoke[WarmupScheduleResult](ctx, c.transport, OperationWarmupScheduleSave, struct {
		versionIdentity
		Input SaveWarmupScheduleInput `json:"input"`
	}{versionIdentity{id, v}, input})
}
func (c versionClient) DeleteWarmupSchedule(ctx context.Context, id string, v int, input DeleteWarmupScheduleInput) (*WarmupScheduleResult, error) {
	return invoke[WarmupScheduleResult](ctx, c.transport, OperationWarmupScheduleDelete, struct {
		versionIdentity
		Input DeleteWarmupScheduleInput `json:"input"`
	}{versionIdentity{id, v}, input})
}
