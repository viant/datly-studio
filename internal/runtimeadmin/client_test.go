package runtimeadmin

import (
	"context"
	"encoding/json"
	"github.com/viant/datly-studio/sdk"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminClientUsesConfiguredOriginAndServerHeldToken(t *testing.T) {
	var statusCalls, reloadCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Studio-Runtime-Token") != "server-secret" {
			t.Errorf("runtime admin token was not sent")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /_studio/status":
			statusCalls++
			_, _ = w.Write([]byte(`{"authenticationMode":"authenticated","status":"ready","revision":9}`))
		case "POST /_studio/reload":
			reloadCalls++
			var body struct {
				Generation int64 `json:"generation"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Generation != 12 {
				t.Errorf("reload body=%+v err=%v", body, err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected admin route %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := New(server.URL, "server-secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	host, err := client.ProbeRuntime(context.Background())
	if err != nil || host == nil || host.Status != "ready" || host.Revision != 9 || host.AuthenticationMode != "authenticated" {
		t.Fatalf("runtime host=%+v err=%v", host, err)
	}
	if err = client.Reload(context.Background(), 12); err != nil {
		t.Fatal(err)
	}
	if statusCalls != 1 || reloadCalls != 1 {
		t.Fatalf("status calls=%d reload calls=%d", statusCalls, reloadCalls)
	}
}

func TestAdminClientRejectsMissingTokenAndNonOriginURL(t *testing.T) {
	for _, input := range []struct{ url, token string }{
		{"http://127.0.0.1:8082", ""},
		{"http://127.0.0.1:8082/unsafe", "secret"},
		{"http://user:pass@127.0.0.1:8082", "secret"},
		{"file:///tmp/socket", "secret"},
	} {
		if client, err := New(input.url, input.token, nil); err == nil || client != nil {
			t.Fatalf("accepted runtime origin=%q token-present=%t", input.url, input.token != "")
		}
	}
}

func TestAdminReloadCarriesSelectedNamespace(t *testing.T) {
	id := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			NamespaceID string `json:"namespaceId"`
			Generation  int64  `json:"generation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.NamespaceID != id || payload.Generation != 7 {
			t.Errorf("namespace reload payload=%+v err=%v", payload, err)
		}
		if r.Header.Get("X-Studio-Runtime-Token") != "deployment-token" {
			t.Error("deployment token missing")
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	client, err := New(server.URL, "deployment-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Reload(sdk.WithNamespaceSelection(context.Background(), id), 7); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeProbeCarriesNamespaceSelection(t *testing.T) {
	id := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_studio/status" || r.URL.Query().Get("namespaceId") != id {
			t.Error("namespace status selection missing")
		}
		if r.Header.Get("X-Studio-Runtime-Token") != "deployment-token" {
			t.Error("deployment token missing")
		}
		_, _ = w.Write([]byte(`{"namespaceId":"` + id + `","mcpUrl":"http://127.0.0.1:8591/mcp","status":"ready","authenticationMode":"required","revision":3}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "deployment-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.ProbeRuntime(sdk.WithNamespaceSelection(context.Background(), id))
	if err != nil || status.Revision != 3 || status.Status != "ready" || status.NamespaceID != id || status.MCPURL != "http://127.0.0.1:8591/mcp" {
		t.Fatalf("namespace probe=%+v err=%v", status, err)
	}
}

func TestRuntimeProbeRejectsWrongNamespaceAndInvalidEndpoint(t *testing.T) {
	id := strings.Repeat("b", 64)
	for _, payload := range []map[string]string{
		{"namespaceId": strings.Repeat("c", 64), "mcpUrl": "http://127.0.0.1:8591/mcp"},
		{"namespaceId": id, "mcpUrl": "file:///tmp/mcp"},
		{"namespaceId": id, "mcpUrl": "http://user:password@127.0.0.1:8591/mcp"},
		{"namespaceId": id, "mcpUrl": "http://127.0.0.1:8591/mcp?token=secret"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload["status"] = "ready"
			_ = json.NewEncoder(w).Encode(payload)
		}))
		client, err := New(server.URL, "deployment-token", nil)
		if err != nil {
			t.Fatal(err)
		}
		status, err := client.ProbeRuntime(sdk.WithNamespaceSelection(context.Background(), id))
		server.Close()
		if err == nil || status != nil {
			t.Fatal("accepted mismatched namespace or invalid endpoint")
		}
	}
}
