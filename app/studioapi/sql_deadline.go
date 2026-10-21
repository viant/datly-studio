package studioapi

import (
	"net/http"
	"strings"
	"time"
)

// SQL execution owns an explicit user-selected deadline. MCP can also stream
// beyond an HTTP write timeout; neither path should inherit the BFF's 60s cap.
func explicitExecutionDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/studio/sdk/connectors.test_sql" || strings.HasPrefix(r.URL.Path, "/v1/studio/sdk-mcp/") || strings.HasPrefix(r.URL.Path, "/v1/studio/mcp/") {
			_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		}
		next.ServeHTTP(w, r)
	})
}
