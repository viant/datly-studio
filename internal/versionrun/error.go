package versionrun

import (
	"errors"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
)

// PublicError preserves the SDK error category while suppressing source SQL,
// connection material, and compiler internals from HTTP/MCP error messages.
func PublicError(err error, operation string) error {
	var sdkErr *sdk.Error
	if errors.As(err, &sdkErr) {
		switch sdkErr.Code {
		case sdk.ErrorNotFound:
			return publisherguard.PublicError(404, operation+" target was not found")
		case sdk.ErrorForbidden:
			return publisherguard.PublicError(403, operation+" is not permitted for this version")
		case sdk.ErrorInvalidArgument:
			return publisherguard.PublicError(400, operation+" input or reader contract is invalid")
		case sdk.ErrorConflict:
			return publisherguard.PublicError(409, operation+" conflicts with the reader contract")
		case sdk.ErrorUnavailable:
			return publisherguard.PublicError(503, operation+" is unavailable")
		}
	}
	return publisherguard.PublicError(502, operation+" could not be executed")
}
