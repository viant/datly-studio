package runtimeadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
