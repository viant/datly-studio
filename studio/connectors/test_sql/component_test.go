package test_sql

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/datly/transcribe"
	xresponse "github.com/viant/xdatly/response"
	"strings"
	"testing"
	"time"
)

func TestSQLExecutionTimeoutBounds(t *testing.T) {
	if duration, err := executionTimeout(nil); err != nil || duration != time.Minute {
		t.Fatalf("default: %v %v", duration, err)
	}
	for _, tc := range []struct {
		seconds int
		want    time.Duration
		invalid bool
	}{{0, 0, false}, {1, time.Second, false}, {300, 5 * time.Minute, false}, {-1, 0, true}, {301, 301 * time.Second, false}, {int(^uint(0) >> 1), 0, true}} {
		got, err := executionTimeout(&tc.seconds)
		if tc.invalid {
			if xresponse.ErrorStatusCode(err, 0) != 400 {
				t.Fatalf("%d: %v", tc.seconds, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("%d: %s %v", tc.seconds, got, err)
		}
	}
}
func TestSQLTimeoutPreservesDeadlineAndDoesNotExposeDriverSecrets(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := executionError(ctx, errors.New("private-driver-secret"), 60*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 504 || !strings.Contains(err.Error(), "60 seconds") || strings.Contains(err.Error(), "private-driver-secret") {
		t.Fatalf("timeout error: %v", err)
	}
	err = executionError(context.Background(), context.DeadlineExceeded, 300*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 504 {
		t.Fatal(err)
	}
	err = executionError(context.Background(), errors.New("private-driver-secret"), 60*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 400 || strings.Contains(err.Error(), "private-driver-secret") {
		t.Fatal(err)
	}
}

type completedJobFailure struct{ reason string }

func (e completedJobFailure) Error() string          { return "CANARY_PASSWORD SQL: SELECT CANARY_DATA" }
func (e completedJobFailure) BigQueryReason() string { return e.reason }

func TestSQLFailureProjectsOnlyStructuredSafeReason(t *testing.T) {
	for _, tc := range []struct{ reason, want string }{{"accessDenied", "runtime_discovery_access_denied"}, {"notFound", "runtime_discovery_source_not_found"}, {"invalidQuery", "runtime_discovery_query_invalid"}, {"CANARY_TOKEN", ""}} {
		err := executionError(context.Background(), completedJobFailure{tc.reason}, 60*time.Second)
		var wire *xresponse.Error
		if !errors.As(err, &wire) || xresponse.ErrorStatusCode(err, 0) != 400 {
			t.Fatal(err)
		}
		raw, marshalErr := json.Marshal(wire.Payload)
		if marshalErr != nil || strings.Contains(string(raw), "CANARY") {
			t.Fatalf("native secrets leaked: %s %v", raw, marshalErr)
		}
		if tc.want != "" && !strings.Contains(string(raw), tc.want) {
			t.Fatalf("reason absent: %s", raw)
		}
		if tc.want == "" && strings.Contains(string(raw), "diagnostics") {
			t.Fatalf("unknown driver reason classified: %s", raw)
		}
	}
}

func TestSQLFailureDoesNotExposeArbitraryCompilerDiagnostics(t *testing.T) {
	cause := &transcribe.CompileError{Diagnostics: []*transcribe.Diagnostic{{Code: "runtime_discovery_query", Message: "CANARY_COMPILER_SECRET"}}}
	err := executionError(context.Background(), cause, 60*time.Second)
	var wire *xresponse.Error
	if !errors.As(err, &wire) {
		t.Fatal(err)
	}
	raw, marshalErr := json.Marshal(wire.Payload)
	if marshalErr != nil || strings.Contains(string(raw), "CANARY") || strings.Contains(string(raw), "diagnostics") {
		t.Fatalf("untrusted compiler diagnostic escaped: %s %v", raw, marshalErr)
	}
}
