package studioapi

import (
	"errors"
	"net/http"
)

func configuredHTTPMiddleware(middleware func(http.Handler) http.Handler, handler http.Handler) (http.Handler, error) {
	if handler == nil {
		return nil, errors.New("Studio HTTP handler is required")
	}
	if middleware == nil {
		return handler, nil
	}
	wrapped := middleware(handler)
	if wrapped == nil {
		return nil, errors.New("Studio HTTP middleware returned no handler")
	}
	return wrapped, nil
}
