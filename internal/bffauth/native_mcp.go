package bffauth

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ProxyNativeMCP forwards explicit credentials to the existing native verifier.
// Its target is host-owned. Browser requests without Authorization retain the
// ordinary BFF cookie path; a supplied invalid credential never selects it.
func (s *Service) ProxyNativeMCP(target *url.URL, mount string) (http.Handler, error) {
	if target == nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return nil, fmt.Errorf("native MCP proxy target is invalid")
	}
	cookie, err := s.Proxy(target, mount)
	if err != nil {
		return nil, err
	}
	mount = "/" + strings.Trim(strings.TrimSpace(mount), "/")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = stripUpstreamCORS
	original := proxy.Director
	proxy.Director = func(request *http.Request) {
		original(request)
		request.URL.Path = "/mcp"
		request.Header.Del("Cookie")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "/"+strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, mount), "/") != "/mcp" {
			http.NotFound(w, r)
			return
		}
		supplied, exists := r.Header[http.CanonicalHeaderKey("Authorization")]
		if !exists {
			cookie.ServeHTTP(w, r)
			return
		}
		if len(supplied) != 1 {
			http.Error(w, "Bearer credential required", http.StatusUnauthorized)
			return
		}
		parts := strings.Fields(supplied[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Bearer credential required", http.StatusUnauthorized)
			return
		}
		forward := r.Clone(r.Context())
		forward.Header = make(http.Header)
		for _, name := range []string{"Accept", "Content-Type", "Mcp-Session-Id", "Mcp-Protocol-Version", "Mcp-Method", "Mcp-Name", "Last-Event-ID", "X-Request-ID", "X-Studio-Namespace"} {
			for _, value := range r.Header.Values(name) {
				forward.Header.Add(name, value)
			}
		}
		forward.Header.Set("Authorization", "Bearer "+parts[1])
		proxy.ServeHTTP(w, forward)
	}), nil
}
