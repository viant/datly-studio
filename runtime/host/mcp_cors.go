package host

import (
	"net/http"
	"strings"

	mcpprotocol "github.com/viant/mcp/server"
)

// browserMCPCORS keeps browser responses under the configured MCP policy.
// The streamable transport can replace its outer CORS origin with a wildcard;
// this wrapper restores the exact origin at the response boundary. It does
// not replace MCP authentication or tool authorization.
func browserMCPCORS(policy *mcpprotocol.Cors, next http.Handler) http.Handler {
	return browserCORS(policy, next, "/mcp")
}

// browserHTTPCORS also covers authentication denials before the Datly HTTP
// gateway can attach route CORS headers.
func browserHTTPCORS(policy *mcpprotocol.Cors, next http.Handler) http.Handler {
	return browserCORS(policy, next, "")
}

func browserCORS(policy *mcpprotocol.Cors, next http.Handler, exactPath string) http.Handler {
	if policy == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if exactPath != "" && r.URL.Path != exactPath {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !corsContains(policy.AllowOrigins, origin) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodOptions {
			next.ServeHTTP(&exactOriginWriter{ResponseWriter: w, origin: origin, allowCredentials: policy.AllowCredentials}, r)
			return
		}
		method := r.Header.Get("Access-Control-Request-Method")
		if method == "" || !corsContains(policy.AllowMethods, method) {
			http.Error(w, "method not allowed", http.StatusForbidden)
			return
		}
		for _, name := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
			name = strings.TrimSpace(name)
			if name != "" && !corsContains(policy.AllowHeaders, name) {
				http.Error(w, "header not allowed", http.StatusForbidden)
				return
			}
		}
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		w.Header().Add("Vary", "Access-Control-Request-Headers")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", method)
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(policy.AllowHeaders, ", "))
		if policy.AllowCredentials != nil && *policy.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

type exactOriginWriter struct {
	http.ResponseWriter
	origin           string
	allowCredentials *bool
	wroteHeader      bool
}

func (w *exactOriginWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.Header().Set("Access-Control-Allow-Origin", w.origin)
	w.Header().Add("Vary", "Origin")
	if w.allowCredentials != nil && *w.allowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Del("Access-Control-Allow-Credentials")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *exactOriginWriter) Write(payload []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

func (w *exactOriginWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *exactOriginWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func corsContains(values []string, candidate string) bool {
	for _, value := range values {
		if value == "*" || strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}
