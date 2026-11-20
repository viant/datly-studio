package versionrun

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/viant/datly-studio/internal/providererror"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	xexec "github.com/viant/xdatly/exec"
	xresponse "github.com/viant/xdatly/response"
)

// ExecutionError records server-side diagnostics before projecting a public error.
func ExecutionError(ctx context.Context, err error, operation, reportID string, versionNo int) error {
	fields := []any{"operation", operation, "reportId", reportID, "versionNo", versionNo, "errorType", fmt.Sprintf("%T", err)}
	// Preserve bounded cause identities without logging arbitrary wrapper text.
	var causeTypes []string
	for cause, depth := err, 0; cause != nil && depth < 16; cause, depth = errors.Unwrap(cause), depth+1 {
		causeTypes = append(causeTypes, fmt.Sprintf("%T", cause))
	}
	fields = append(fields, "causeTypes", causeTypes)
	var sdkError *sdk.Error
	if errors.As(err, &sdkError) {
		fields = append(fields, "errorCode", sdkError.Code)
	}
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
	if (operation != "reader preview" && operation != "view test") || !errors.As(public, &response) || response.Code != 502 {
		if traceID != "" && errors.As(public, &response) {
			switch payload := response.Payload.(type) {
			case map[string]string:
				payload["requestId"] = traceID
			case map[string]any:
				payload["requestId"] = traceID
			}
		}
		return public
	}
	label := "Reader preview"
	if operation == "view test" {
		label = "View test"
	}
	message := label + " failed before results were returned."
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(context.Cause(ctx), context.DeadlineExceeded):
		message = label + " exceeded its execution deadline. Review query scope, filters, and query limits before retrying."
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		message = label + " was canceled before results were returned."
		// A canceled context does not identify a user action. Do not infer one
		// from a driver error or expose an arbitrary context cause.
	default:
		if status := providererror.AppendCatalogStatus("", err); status != "" {
			message = label + " failed with " + strings.TrimSpace(status) + "."
		}
	}
	if traceID != "" {
		message += " Check server logs using request ID " + traceID + " to identify the cause."
	} else if strings.HasSuffix(message, "before results were returned.") {
		message += " Check server logs to identify the cause."
	}
	return publisherguard.PublicError(response.Code, message, err)
}
