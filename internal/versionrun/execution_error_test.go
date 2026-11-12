package versionrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk"
	xexec "github.com/viant/xdatly/exec"
	xresponse "github.com/viant/xdatly/response"
	"google.golang.org/api/googleapi"
)

func TestExecutionErrorLogsNestedProviderCauseWithoutCredentials(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	ctx := xexec.WithContext(context.Background(), &xexec.Context{TraceID: "synthetic-trace"})
	cause := &mysql.MySQLError{Number: 1146, SQLState: [5]byte{'4', '2', 'S', '0', '2'}, Message: "Table 'synthetic.missing_records' doesn't exist password=PASSWORD_CANARY token=TOKEN_CANARY Authorization: Bearer BEARER_CANARY user:DSN_CANARY@tcp(host:3306)/db"}
	err := fmt.Errorf("wrapper WRAPPER_CANARY: %w", &sdk.Error{Code: sdk.ErrorInternal, Message: "private SDK_CANARY", Cause: cause})
	public := ExecutionError(ctx, err, "reader preview", "synthetic-reader", 7)
	var wire *xresponse.Error
	if !errors.As(public, &wire) || wire.Code != 502 {
		t.Fatalf("public category changed: %v", public)
	}
	publicJSON, _ := json.Marshal(wire.Payload)
	if string(publicJSON) != `{"message":"Reader preview failed with MySQL error 1146. Check server logs using request ID synthetic-trace to identify the cause."}` {
		t.Fatalf("public response changed: %s", publicJSON)
	}
	var event struct {
		Message       string                                             `json:"msg"`
		TraceID       string                                             `json:"reqTraceId"`
		ReportID      string                                             `json:"reportId"`
		VersionNo     int                                                `json:"versionNo"`
		ProviderError struct{ Provider, Code, SQLState, Message string } `json:"providerError"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("server event missing or invalid: %v", err)
	}
	if event.Message != "Studio version execution failed" || event.TraceID != "synthetic-trace" || event.ReportID != "synthetic-reader" || event.VersionNo != 7 || event.ProviderError.Provider != "MySQL" || event.ProviderError.Code != "1146" || event.ProviderError.SQLState != "42S02" || !strings.Contains(event.ProviderError.Message, "missing_records") {
		t.Fatalf("provider cause or correlation missing: %s", output.Bytes())
	}
	for _, secret := range []string{"PASSWORD_CANARY", "TOKEN_CANARY", "BEARER_CANARY", "DSN_CANARY", "WRAPPER_CANARY", "SDK_CANARY", "missing_records"} {
		if strings.Contains(string(publicJSON), secret) || strings.Contains(public.Error(), secret) {
			t.Fatalf("private detail reached public response: %s", publicJSON)
		}
		if secret != "missing_records" && strings.Contains(output.String(), secret) {
			t.Fatalf("credential reached server event: %s", output.Bytes())
		}
	}
	if !errors.Is(err, cause) {
		t.Fatal("original error chain changed")
	}
}

func TestExecutionErrorUnknownCauseLogsOnlyType(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	public := ExecutionError(context.Background(), errors.New("unknown PASSWORD_CANARY body=PAYLOAD_CANARY"), "reader preview", "synthetic-reader", 1)
	if !strings.Contains(output.String(), `"errorType":"*errors.errorString"`) {
		t.Fatalf("unknown failure type missing: %s", output.Bytes())
	}
	for _, secret := range []string{"PASSWORD_CANARY", "PAYLOAD_CANARY"} {
		if strings.Contains(output.String(), secret) || strings.Contains(public.Error(), secret) {
			t.Fatalf("unknown private detail leaked: %s", output.Bytes())
		}
	}
}

func TestPreviewExecutionPublicMessages(t *testing.T) {
	const interruption = "Preview was interrupted before results were returned. Check server logs to identify the cause."
	const deadline = "Reader preview exceeded its execution deadline. Review query scope, filters, and query limits before retrying."
	expired, expire := context.WithCancelCause(context.Background())
	expire(context.DeadlineExceeded)
	// A driver may return its own error after the execution context expires.
	deadlineCtx, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("private cancellation PASSWORD_CANARY"))
	for _, test := range []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{"wrapped deadline", context.Background(), fmt.Errorf("private wrapper: %w", &sdk.Error{Code: sdk.ErrorInternal, Cause: context.DeadlineExceeded}), deadline},
		{"driver error after deadline", deadlineCtx, errors.New("private driver error"), deadline},
		{"wrapped cancellation", context.Background(), fmt.Errorf("private wrapper: %w", context.Canceled), interruption},
		{"unknown cancellation cause", canceled, errors.New("private driver error"), interruption},
		{"deadline cancellation cause", expired, context.Canceled, deadline},
		{"unknown", context.Background(), errors.New("unknown PASSWORD_CANARY"), interruption},
		{"mysql", context.Background(), &mysql.MySQLError{Number: 1146, Message: "SELECT SQL_CANARY PASSWORD_CANARY"}, "Reader preview failed with MySQL error 1146."},
		{"provider status", context.Background(), &googleapi.Error{Code: 403, Message: "PASSWORD_CANARY", Errors: []googleapi.ErrorItem{{Reason: "accessDenied"}}}, "Reader preview failed with Google API error 403 (accessDenied)."},
		{"unrecognized provider reason", context.Background(), &googleapi.Error{Code: 400, Message: "PASSWORD_CANARY", Errors: []googleapi.ErrorItem{{Reason: "SQL_CANARY"}}}, "Reader preview failed with Google API error 400."},
	} {
		t.Run(test.name, func(t *testing.T) {
			public := ExecutionError(test.ctx, test.err, "reader preview", "synthetic-reader", 1)
			var response *xresponse.Error
			if !errors.As(public, &response) || response.Code != 502 {
				t.Fatalf("unexpected response: %v", public)
			}
			if got := response.Payload.(map[string]string)["message"]; got != test.want {
				t.Fatalf("message = %q, want %q", got, test.want)
			}
			if !errors.Is(public, test.err) {
				t.Fatal("trusted error chain lost")
			}
		})
	}
}

func TestPreviewExecutionPreservesExistingCategories(t *testing.T) {
	for _, code := range []sdk.ErrorCode{sdk.ErrorForbidden, sdk.ErrorInvalidArgument, sdk.ErrorNotFound, sdk.ErrorConflict, sdk.ErrorUnavailable} {
		err := &sdk.Error{Code: code, Message: "PASSWORD_CANARY", Cause: context.DeadlineExceeded}
		want := PublicError(err, "reader preview")
		got := ExecutionError(context.Background(), err, "reader preview", "synthetic-reader", 1)
		if got.Error() != want.Error() {
			t.Fatalf("category %s changed: got %v, want %v", code, got, want)
		}
	}
}
