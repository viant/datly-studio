package host

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// AdminHandler exposes only namespace-specific reload to the deployment token.
// Group mode has no unscoped reload operation or public namespace mutation API.
func (g *NamespaceGroup) AdminHandler(token string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
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
