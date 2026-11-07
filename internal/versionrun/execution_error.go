package versionrun

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/viant/datly-studio/internal/providererror"
	xexec "github.com/viant/xdatly/exec"
)

// ExecutionError records server-side diagnostics before projecting a public error.
func ExecutionError(ctx context.Context, err error, operation, reportID string, versionNo int) error {
	fields := []any{"operation", operation, "reportId", reportID, "versionNo", versionNo, "errorType", fmt.Sprintf("%T", err)}
	if execution := xexec.GetContext(ctx); execution != nil {
		fields = append(fields, "reqTraceId", execution.SnapshotForLogging().TraceID)
	}
	if provider := providererror.FromError(err); provider != nil {
		fields = append(fields, "providerError", provider)
	}
	slog.ErrorContext(ctx, "Studio version execution failed", fields...)
	return PublicError(err, operation)
}
