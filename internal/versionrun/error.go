package versionrun

import (
	"context"
	"errors"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
)

// PublicError preserves the SDK error category while suppressing source SQL,
// connection material, and compiler internals from HTTP/MCP error messages.
func PublicError(err error, operation string) error {
	var sdkErr *sdk.Error
	if errors.As(err, &sdkErr) {
		switch sdkErr.Code {
		case sdk.ErrorNotFound:
			return publisherguard.PublicError(404, operation+" target was not found", err)
		case sdk.ErrorUnauthorized:
			return publisherguard.PublicError(401, operation+" requires authentication", err)
		case sdk.ErrorForbidden:
			return publisherguard.PublicError(403, operation+" is not permitted for this version", err)
		case sdk.ErrorInvalidArgument:
			if len(sdkErr.Diagnostics) > 0 {
				public := publisherguard.PublicError(400, operation+" input or reader contract is invalid", err)
				var response *xresponse.Error
				if errors.As(public, &response) {
					response.Payload = map[string]any{"message": operation + " input or reader contract is invalid", "diagnostics": sdkErr.Diagnostics}
				}
				return public
			}
			return publisherguard.PublicError(400, operation+" input or reader contract is invalid", err)
		case sdk.ErrorConflict:
			return publisherguard.PublicError(409, operation+" conflicts with the reader contract", err)
		case sdk.ErrorUnavailable:
			return publisherguard.PublicError(503, operation+" is unavailable", err)
		}
	}
	// Runtime input lifecycle errors retain their typed response cause inside an
	// internal SDK wrapper. Preserve the category without exposing its payload.
	var native *xresponse.Error
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && errors.As(err, &native) {
		switch native.Code {
		case 400:
			return publisherguard.PublicError(400, operation+" input or reader contract is invalid", err)
		case 401:
			return publisherguard.PublicError(401, operation+" requires authentication", err)
		case 403:
			return publisherguard.PublicError(403, operation+" is not permitted for this version", err)
		case 404:
			return publisherguard.PublicError(404, operation+" target was not found", err)
		case 409:
			return publisherguard.PublicError(409, operation+" conflicts with the reader contract", err)
		case 503:
			return publisherguard.PublicError(503, operation+" is unavailable", err)
		}
	}
	return publisherguard.PublicError(502, operation+" could not be executed", err)
}
