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

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk"
	xexec "github.com/viant/xdatly/exec"
	xresponse "github.com/viant/xdatly/response"
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
	if string(publicJSON) != `{"message":"reader preview could not be executed"}` {
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
