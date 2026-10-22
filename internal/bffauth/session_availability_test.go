package bffauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/httptransport"
)

type unavailableSessionStore struct {
	SessionStore
	failure error
}

func (s *unavailableSessionStore) Get(ctx context.Context, id string) (session, bool, error) {
	if s.failure != nil {
		return session{}, false, s.failure
	}
	return s.SessionStore.Get(ctx, id)
}

type unavailableTransport struct{ calls int }

func (t *unavailableTransport) Invoke(context.Context, string, any, any) error { t.calls++; return nil }

func TestSessionStoreOutageFailsClosedAndRecoversWithoutLogin(t *testing.T) {
	store := &unavailableSessionStore{SessionStore: newMemoryStore()}
	service, err := New(Config{Store: store}, verifierStub{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := service.Exchange(context.Background(), "test-subject")
	if err != nil {
		t.Fatal(err)
	}
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { upstreamCalls++; w.WriteHeader(http.StatusNoContent) }))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	proxy, err := service.Proxy(target, "/runtime")
	if err != nil {
		t.Fatal(err)
	}
	transport := &unavailableTransport{}
	gateway := httptransport.Gateway{Config: httptransport.Config{Mode: httptransport.Authenticated, Authenticator: service}, Transport: transport}
	handlers := []struct {
		name, path string
		handler    http.Handler
		recovered  int
	}{
		{"me", "/v1/studio/auth/me", http.HandlerFunc(service.handleMe), http.StatusOK},
		{"proxy", "/runtime/read", proxy, http.StatusNoContent},
		{"sdk", httptransport.PathPrefix + sdk.OperationComponentList, gateway, http.StatusOK},
	}
	for _, item := range handlers {
		t.Run(item.name, func(t *testing.T) {
			for _, phase := range []struct {
				name    string
				failure error
				cookie  string
				status  int
			}{
				{"outage", errors.New("private database connection detail"), id, http.StatusServiceUnavailable},
				{"recovery", nil, id, item.recovered},
				{"missing", nil, "", http.StatusUnauthorized},
				{"unknown", nil, "unknown", http.StatusUnauthorized},
			} {
				store.failure = phase.failure
				beforeProxy, beforeSDK := upstreamCalls, transport.calls
				req := httptest.NewRequest(http.MethodPost, item.path, nil)
				if phase.cookie != "" {
					req.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: phase.cookie})
				}
				response := httptest.NewRecorder()
				item.handler.ServeHTTP(response, req)
				if response.Code != phase.status {
					t.Fatalf("%s status=%d body=%s", phase.name, response.Code, response.Body.String())
				}
				if strings.Contains(response.Body.String(), "private database") {
					t.Fatal("internal cause leaked")
				}
				if response.Header().Get("Set-Cookie") != "" {
					t.Fatal("session cookie changed")
				}
				if phase.status >= 400 && (upstreamCalls != beforeProxy || transport.calls != beforeSDK) {
					t.Fatal("failed authentication dispatched request")
				}
			}
		})
	}
}
