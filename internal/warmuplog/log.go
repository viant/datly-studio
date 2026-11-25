// Package warmuplog owns Studio warmup lifecycle diagnostics, not execution policy.
package warmuplog

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/viant/datly-studio/sdk"
	xhandler "github.com/viant/xdatly/handler"
)

const (
	summaryInterval = time.Minute
	idleExpiry      = 15 * time.Minute
	maxKeys         = 1024
	// Diagnostic has ten possible noncredential classifications. Keep every
	// supported classification visible without accepting arbitrary error text.
	maxClasses = 10
	maxText    = 512
)

type Scope struct {
	ScheduleID, NamespaceID, ComponentID, RunID string
	VersionNo                                   int
}
type scopeKey struct{}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}
func FromContext(ctx context.Context) Scope { value, _ := ctx.Value(scopeKey{}).(Scope); return value }
func safe(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(s, "?"))
	if len(s) > maxText {
		s = s[:maxText]
		s = strings.ToValidUTF8(s, "")
	}
	return s
}
func (s Scope) fields() []any {
	fields := []any{}
	for _, item := range []struct{ key, value string }{{"schedule_id", s.ScheduleID}, {"namespace_id", s.NamespaceID}, {"component_id", s.ComponentID}, {"run_id", s.RunID}} {
		if item.value != "" {
			fields = append(fields, item.key, safe(item.value))
		}
	}
	if s.VersionNo > 0 {
		fields = append(fields, "version_no", s.VersionNo)
	}
	return fields
}
func Emit(ctx context.Context, level slog.Level, event string, scope Scope, fields ...any) {
	slog.Log(ctx, level, event, append(scope.fields(), fields...)...)
}

// Diagnostic never forwards error text, SDK messages, credentials or remote bodies.
func Diagnostic(err error) (string, string) {
	var conflict *xhandler.Conflict
	if errors.As(err, &conflict) {
		return "conflict", "Warmup state changed before the operation completed."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "The server-owned warmup deadline was exceeded."
	}
	if errors.Is(err, context.Canceled) {
		return "canceled", "The operation was canceled."
	}
	var se *sdk.Error
	if errors.As(err, &se) {
		switch se.Code {
		case sdk.ErrorForbidden, sdk.ErrorUnauthorized:
			return string(se.Code), "Warmup authorization is unavailable or denied."
		case sdk.ErrorNotFound:
			return string(se.Code), "The warmup target is unavailable."
		case sdk.ErrorInvalidArgument:
			return string(se.Code), "The warmup request or reader contract is invalid."
		case sdk.ErrorConflict:
			return string(se.Code), "Warmup state changed before the operation completed."
		case sdk.ErrorUnavailable:
			return string(se.Code), "A required warmup service is unavailable."
		case sdk.ErrorInternal:
			return string(se.Code), "The warmup operation could not complete."
		}
	}
	return "operation_failed", "The warmup operation could not complete."
}

type suppressionKey struct{ schedule, stage string }
type failure struct {
	lastSeen, lastSummary time.Time
	count                 uint64
	classes               map[string]uint64
	reported              map[string]bool
	level                 slog.Level
}
type suppression struct {
	mu            sync.Mutex
	now           func() time.Time
	failures      map[suppressionKey]*failure
	overflowCount uint64
	overflowAt    time.Time
}
type suppressionContextKey struct{}

func WithScheduler(ctx context.Context) context.Context {
	return context.WithValue(ctx, suppressionContextKey{}, &suppression{now: time.Now, failures: make(map[suppressionKey]*failure)})
}
func suppressionFrom(ctx context.Context) *suppression {
	s, _ := ctx.Value(suppressionContextKey{}).(*suppression)
	return s
}

// Failure logs each new classification once per summary period, with bounded
// class counts. Overflow is aggregated rather than evicting and relogging keys.
func Failure(ctx context.Context, scope Scope, stage, event string, err error, fields ...any) {
	code, message := Diagnostic(err)
	if stage == "credentials" {
		code, message = "credentials_unavailable", "Scheduler service credentials are unavailable."
	}
	level := slog.LevelError
	var conflict *xhandler.Conflict
	if errors.As(err, &conflict) && (stage == "admission_commit" || stage == "worker_start_write") {
		level = slog.LevelDebug
		event = "Studio warmup admission contended"
		if strings.HasPrefix(stage, "worker_") {
			event = "Studio warmup worker state contended"
		}
	}
	extra := append([]any{"stage", stage, "error_code", code, "diagnostic", safe(message)}, fields...)
	state := suppressionFrom(ctx)
	if state == nil {
		Emit(ctx, level, event, scope, extra...)
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	now := state.now()
	state.expire(now)
	key := suppressionKey{scope.ScheduleID, stage}
	entry := state.failures[key]
	if entry == nil {
		// Reserve capacity for scheduler-wide credentials/scan/cleanup failures.
		limit := maxKeys
		if scope.ScheduleID != "" {
			limit -= 4
		}
		if len(state.failures) >= limit {
			state.overflowCount++
			if state.overflowAt.IsZero() || now.Sub(state.overflowAt) >= summaryInterval {
				Emit(ctx, slog.LevelWarn, "Studio warmup diagnostic capacity summary", Scope{}, "repetitions", state.overflowCount, "tracked_keys", len(state.failures))
				state.overflowAt = now
				state.overflowCount = 0
			}
			return
		}
		entry = &failure{level: level, lastSummary: now, classes: map[string]uint64{}, reported: map[string]bool{}}
		state.failures[key] = entry
	}
	if level > entry.level {
		entry.level = level
	}
	entry.lastSeen = now
	entry.count++
	class := code
	if _, ok := entry.classes[class]; !ok && len(entry.classes) >= maxClasses {
		class = "other"
	}
	entry.classes[class]++
	if now.Sub(entry.lastSummary) >= summaryInterval {
		Emit(ctx, entry.level, "Studio warmup failure summary", scope, append(extra, "repetitions", entry.count, "failure_counts", copyCounts(entry.classes))...)
		entry.lastSummary = now
		entry.reported = map[string]bool{class: true}
	} else if !entry.reported[class] && class != "other" {
		Emit(ctx, level, event, scope, append(extra, "repetitions", entry.count)...)
		entry.reported[class] = true
	}
}
func copyCounts(in map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (s *suppression) expire(now time.Time) {
	for key, entry := range s.failures {
		if now.Sub(entry.lastSeen) >= idleExpiry {
			delete(s.failures, key)
		}
	}
}

// Recovered requires success at this same owning stage. Expiry is not recovery.
func Recovered(ctx context.Context, scope Scope, stage string) {
	state := suppressionFrom(ctx)
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.expire(state.now())
	key := suppressionKey{scope.ScheduleID, stage}
	entry := state.failures[key]
	if entry == nil {
		return
	}
	delete(state.failures, key)
	if entry.level == slog.LevelDebug {
		return
	}
	Emit(ctx, slog.LevelInfo, "Studio warmup operation recovered", scope, "stage", stage, "repetitions", entry.count, "failure_counts", copyCounts(entry.classes))
}

// Completion is invocation-local evidence from the native finalizer. Retried
// attempts replace earlier evidence; no transaction outcome is inferred.
type Completion struct {
	mu      sync.Mutex
	outcome *xhandler.Outcome
}
type completionKey struct{}

func CaptureCompletion(ctx context.Context) (context.Context, *Completion) {
	c := &Completion{}
	return context.WithValue(ctx, completionKey{}, c), c
}
func RecordCompletion(ctx context.Context, outcome xhandler.Outcome) {
	c, _ := ctx.Value(completionKey{}).(*Completion)
	if c == nil {
		return
	}
	copy := outcome.Clone()
	c.mu.Lock()
	c.outcome = &copy
	c.mu.Unlock()
}
func (c *Completion) Fields() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.outcome == nil {
		return nil
	}
	fields := []any{"transaction_state", string(c.outcome.State())}
	for _, unit := range c.outcome.Transactions {
		if unit.State == xhandler.TransactionCommitUnknown {
			return append(fields, "commit_uncertain", true)
		}
	}
	return fields
}
