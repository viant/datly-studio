package versionrun

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/viant/datly-studio/internal/providererror"
	"github.com/viant/datly-studio/internal/publisherguard"
	xexec "github.com/viant/xdatly/exec"
	xresponse "github.com/viant/xdatly/response"
)

// ExecutionError records server-side diagnostics before projecting a public error.
func ExecutionError(ctx context.Context, err error, operation, reportID string, versionNo int) error {
	fields := []any{"operation", operation, "reportId", reportID, "versionNo", versionNo, "errorType", fmt.Sprintf("%T", err)}
	var traceID string
	if execution := xexec.GetContext(ctx); execution != nil {
		traceID = execution.SnapshotForLogging().TraceID
		fields = append(fields, "reqTraceId", traceID)
	}
	if provider := providererror.FromError(err); provider != nil {
		fields = append(fields, "providerError", provider)
	}
	slog.ErrorContext(ctx, "Studio version execution failed", fields...)
	public := PublicError(err, operation)
	var response *xresponse.Error
	// Preserve established authorization, input and availability categories, and
	// keep this presentation change scoped to the ordinary preview operation.
	if operation != "reader preview" || !errors.As(public, &response) || response.Code != 502 {
		return public
	}
	message := "Preview was interrupted before results were returned."
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(context.Cause(ctx), context.DeadlineExceeded):
		message = "Reader preview exceeded its execution deadline. Review query scope, filters, and query limits before retrying."
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		// A canceled context does not identify a user action. Do not infer one
		// from a driver error or expose an arbitrary context cause.
	default:
		if status := providererror.AppendCatalogStatus("", err); status != "" {
			message = "Reader preview failed with " + strings.TrimSpace(status) + "."
		}
	}
	if traceID != "" {
		message += " Check server logs using request ID " + traceID + " to identify the cause."
	} else if message == "Preview was interrupted before results were returned." {
		message += " Check server logs to identify the cause."
	}
	return publisherguard.PublicError(response.Code, message, err)
}
