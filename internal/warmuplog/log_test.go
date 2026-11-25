package warmuplog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/logtest"
	"github.com/viant/datly-studio/sdk"
)

func TestFailureSuppressionAndRecovery(t *testing.T) {
	logs := logtest.CaptureLogs(t)
	ctx := WithScheduler(t.Context())
	state := suppressionFrom(ctx)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state.now = func() time.Time { return now }
	scope := Scope{ScheduleID: "schedule", ComponentID: "reader", VersionNo: 1}
	err := &sdk.Error{Code: sdk.ErrorForbidden, Message: "password=canary SELECT private", Cause: errors.New("Bearer token-canary")}
	for i := 0; i < 100; i++ {
		Failure(ctx, scope, "authorization", "rejected", err)
	}
	if len(logs.Entries()) != 1 {
		t.Fatal("repeated failures were not suppressed")
	}
	now = now.Add(summaryInterval)
	Failure(ctx, scope, "authorization", "rejected", err)
	entries := logs.Entries()
	if len(entries) != 2 || entries[1]["repetitions"] != float64(101) {
		t.Fatalf("summary: %+v", entries)
	}
	// A different stage and schedule are independent; recovery is stage-specific.
	Failure(ctx, scope, "admission_commit", "rejected", errors.New("dsn-canary"))
	Recovered(ctx, scope, "authorization")
	if got := logs.Entries(); len(got) != 4 || got[3]["stage"] != "authorization" || got[3]["repetitions"] != float64(101) {
		t.Fatalf("recovery: %+v", got)
	}
	Recovered(ctx, scope, "authorization")
	if len(logs.Entries()) != 4 {
		t.Fatal("success without a previous failure logged recovery")
	}
	// Expiry/removal is not evidence of success.
	now = now.Add(idleExpiry)
	Recovered(ctx, scope, "admission_commit")
	if len(logs.Entries()) != 4 || len(state.failures) != 0 {
		t.Fatal("expiry fabricated recovery or retained state")
	}
	data, _ := json.Marshal(logs.Entries())
	for _, secret := range []string{"canary", "SELECT", "Bearer", "password"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret in logs: %s", data)
		}
	}
}

func TestClassificationChangesAndCredentialIsolation(t *testing.T) {
	logs := logtest.CaptureLogs(t)
	ctx := WithScheduler(t.Context())
	scope := Scope{ScheduleID: "schedule"}
	for i := 0; i < 20; i++ {
		for _, code := range []sdk.ErrorCode{sdk.ErrorForbidden, sdk.ErrorUnavailable} {
			Failure(ctx, scope, "authorization", "rejected", &sdk.Error{Code: code})
		}
	}
	if len(logs.Entries()) != 2 {
		t.Fatal("classification alternation flooded logs or hid a new classification")
	}
	for i := 0; i < 10; i++ {
		Failure(ctx, Scope{}, "credentials", "credentials", errors.New("login private response"))
	}
	other := WithScheduler(context.WithoutCancel(t.Context()))
	Failure(other, Scope{}, "credentials", "credentials", errors.New("another private response"))
	if len(logs.Entries()) != 4 {
		t.Fatal("credential suppression leaked across schedulers")
	}
	Recovered(ctx, Scope{}, "credentials")
	if got := logs.Entries(); got[4]["repetitions"] != float64(10) {
		t.Fatalf("credential recovery: %+v", got)
	}
}

func TestBoundedSuppressionOverflow(t *testing.T) {
	logs := logtest.CaptureLogs(t)
	ctx := WithScheduler(t.Context())
	state := suppressionFrom(ctx)
	for round := 0; round < 2; round++ {
		for i := 0; i < maxKeys+50; i++ {
			Failure(ctx, Scope{ScheduleID: fmt.Sprint(i)}, "authorization", "rejected", errors.New("private"))
		}
	}
	if len(state.failures) > maxKeys || len(logs.Entries()) > maxKeys+1 {
		t.Fatalf("unbounded suppression: keys=%d logs=%d", len(state.failures), len(logs.Entries()))
	}
	// Global credentials remain trackable after the schedule capacity is filled.
	Failure(ctx, Scope{}, "credentials", "credentials", errors.New("private"))
	Recovered(ctx, Scope{}, "credentials")
	entries := logs.Entries()
	if entries[len(entries)-1]["msg"] != "Studio warmup operation recovered" {
		t.Fatal("capacity hid credential recovery")
	}
}

func TestEverySupportedFailureClassificationRemainsVisible(t *testing.T) {
	logs := logtest.CaptureLogs(t)
	ctx := WithScheduler(t.Context())
	scope := Scope{ScheduleID: "schedule"}
	failures := []error{context.DeadlineExceeded, context.Canceled, errors.New("private")}
	for _, code := range []sdk.ErrorCode{sdk.ErrorForbidden, sdk.ErrorUnauthorized, sdk.ErrorNotFound, sdk.ErrorInvalidArgument, sdk.ErrorConflict, sdk.ErrorUnavailable, sdk.ErrorInternal} {
		failures = append(failures, &sdk.Error{Code: code})
	}
	for repeat := 0; repeat < 2; repeat++ {
		for _, failure := range failures {
			Failure(ctx, scope, "authorization", "rejected", failure)
		}
	}
	if got := logs.Entries(); len(got) != len(failures) {
		t.Fatalf("classification changes hidden or repeated: %+v", got)
	}
}
