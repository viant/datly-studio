package host

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// AdminHandler exposes namespace-specific runtime operations to the deployment token.
// Group mode has no unscoped reload operation or public namespace mutation API.
func (g *NamespaceGroup) AdminHandler(token string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/_studio/cache/invalidate" {
			g.namespaceCacheInvalidate(response, request, token)
			return
		}
		if request.URL.Path == "/_studio/connection" {
			g.namespaceConnection(response, request, token)
			return
		}
		if request.URL.Path == "/_studio/status" {
			g.namespaceStatus(response, request, token)
			return
		}
		if request.URL.Path != "/_studio/reload" {
			http.NotFound(response, request)
			return
		}
		if request.Method != http.MethodPost {
			response.Header().Set("Allow", http.MethodPost)
			http.Error(response, "method not allowed", 405)
			return
		}
		values := request.Header.Values("X-Studio-Runtime-Token")
		if len(values) != 1 || !validAdminToken(token, values[0]) {
			http.Error(response, "unauthorized", 401)
			return
		}
		var payload struct {
			NamespaceID string `json:"namespaceId"`
			Generation  int64  `json:"generation"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			http.Error(response, "invalid namespace reload request", 400)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(response, "invalid namespace reload request", 400)
			return
		}
		decoded, err := hex.DecodeString(payload.NamespaceID)
		if err != nil || len(decoded) != 32 || payload.NamespaceID != strings.ToLower(payload.NamespaceID) || payload.Generation < 0 {
			http.Error(response, "namespace ID and nonnegative generation are required", 400)
			return
		}
		if _, exists := g.Listeners.Get(payload.NamespaceID); !exists {
			http.Error(response, "namespace runtime is unavailable", 404)
			return
		}
		if err := g.Listeners.Reload(request.Context(), payload.NamespaceID, payload.Generation); err != nil {
			http.Error(response, "namespace runtime reload failed", 409)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]string{"status": "active", "namespaceId": payload.NamespaceID})
	})
}

func (g *NamespaceGroup) namespaceCacheInvalidate(response http.ResponseWriter, request *http.Request, token string) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	values := request.Header.Values("X-Studio-Runtime-Token")
	if len(values) != 1 || !validAdminToken(token, values[0]) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	var payload struct {
		NamespaceID string `json:"namespaceId"`
		ReportID    string `json:"reportId"`
		VersionNo   int    `json:"versionNo"`
		RoutePath   string `json:"routePath"`
		View        string `json:"view"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || decoder.Decode(new(any)) != io.EOF || payload.ReportID == "" || payload.VersionNo < 1 || payload.RoutePath == "" || payload.View == "" {
		http.Error(response, "invalid cache administration request", http.StatusBadRequest)
		return
	}
	decoded, err := hex.DecodeString(payload.NamespaceID)
	if err != nil || len(decoded) != 32 || payload.NamespaceID != strings.ToLower(payload.NamespaceID) {
		http.Error(response, "canonical namespace ID is required", http.StatusBadRequest)
		return
	}
	invalidated, err := g.Listeners.InvalidatePublishedCacheView(request.Context(), payload.NamespaceID, payload.ReportID, payload.VersionNo, payload.RoutePath, payload.View)
	if err != nil {
		http.Error(response, "exact published cache view is unavailable", http.StatusConflict)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(response).Encode(struct {
		NamespaceID string `json:"namespaceId"`
		Invalidated any    `json:"invalidated"`
	}{payload.NamespaceID, invalidated})
}

func (g *NamespaceGroup) namespaceStatus(response http.ResponseWriter, request *http.Request, token string) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		http.Error(response, "method not allowed", 405)
		return
	}
	values := request.Header.Values("X-Studio-Runtime-Token")
	if len(values) != 1 || !validAdminToken(token, values[0]) {
		http.Error(response, "unauthorized", 401)
		return
	}
	ids := request.URL.Query()["namespaceId"]
	if len(ids) != 1 {
		http.Error(response, "one namespace ID is required", 400)
		return
	}
	decoded, err := hex.DecodeString(ids[0])
	if err != nil || len(decoded) != 32 || ids[0] != strings.ToLower(ids[0]) {
		http.Error(response, "canonical namespace ID is required", 400)
		return
	}
	endpoint, mode, revision, ready := g.Listeners.RuntimeStatus(ids[0])
	if !ready {
		http.Error(response, "namespace runtime is unavailable", 404)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(struct {
		NamespaceID        string `json:"namespaceId"`
		AuthenticationMode string `json:"authenticationMode"`
		Status             string `json:"status"`
		Revision           int64  `json:"revision"`
		MCPURL             string `json:"mcpUrl"`
	}{ids[0], mode, "ready", revision, endpoint.URL()})
}

func (g *NamespaceGroup) namespaceConnection(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", 405)
		return
	}
	credentials := r.Header.Values("X-Studio-Runtime-Token")
	if len(credentials) != 1 || !validAdminToken(token, credentials[0]) {
		http.Error(w, "unauthorized", 401)
		return
	}
	ids := r.URL.Query()["namespaceId"]
	if len(ids) != 1 {
		http.Error(w, "one namespace ID is required", 400)
		return
	}
	decoded, err := hex.DecodeString(ids[0])
	if err != nil || len(decoded) != 32 || ids[0] != strings.ToLower(ids[0]) {
		http.Error(w, "canonical namespace ID is required", 400)
		return
	}
	if _, exists := g.Listeners.Get(ids[0]); !exists {
		http.Error(w, "namespace runtime is unavailable", 404)
		return
	}
	configuration, err := g.Listeners.Connection(r.Context(), ids[0])
	if err != nil {
		http.Error(w, "namespace connection export is unavailable", 503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		NamespaceID string          `json:"namespaceId"`
		Config      json.RawMessage `json:"config"`
	}{ids[0], configuration})
}
