package namespacemcp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type protectedHTTP struct {
	id     string
	closed *atomic.Int32
}

func (h *protectedHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/mcp" {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("Authorization") != "Bearer synthetic" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, _ = io.WriteString(w, h.id)
}
func (h *protectedHTTP) Close(context.Context) error { h.closed.Add(1); return nil }

func TestHTTPManagerSharesScopesAndPreservesListenerOnCollision(t *testing.T) {
	var closed atomic.Int32
	manager, err := NewSharedHTTP(func(_ context.Context, e Endpoint) (http.Handler, error) {
		return &protectedHTTP{id: e.NamespaceID, closed: &closed}, nil
	}, "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	aID, bID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	a, err := manager.Start(context.Background(), aID, 0)
	require.NoError(t, err)
	b, err := manager.Start(context.Background(), bID, 0)
	require.NoError(t, err)
	require.Equal(t, a.Port, b.Port)
	require.NotEqual(t, a.Path, b.Path)
	get := func(endpoint Endpoint, credential string, status int, body string) {
		t.Helper()
		request, e := http.NewRequest(http.MethodGet, endpoint.URL(), nil)
		require.NoError(t, e)
		request.Header.Set("Authorization", credential)
		response, e := (&http.Client{Timeout: time.Second}).Do(request)
		require.NoError(t, e)
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		require.NoError(t, e)
		require.Equal(t, status, response.StatusCode)
		if body != "" {
			require.Equal(t, body, string(raw))
		}
	}
	get(a, "Bearer synthetic", 200, aID)
	get(b, "Bearer synthetic", 200, bID)
	get(a, "", 401, "")
	unknown := a
	unknown.Path = "/namespaces/" + strings.Repeat("c", 64) + "/mcp"
	get(unknown, "Bearer synthetic", 404, "")
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })
	port := busy.Addr().(*net.TCPAddr).Port
	_, err = manager.Rebind(context.Background(), aID, port)
	require.Error(t, err)
	unchanged, ok := manager.Get(aID)
	require.True(t, ok)
	require.Equal(t, a, unchanged)
	get(a, "Bearer synthetic", 200, aID)
	require.NoError(t, busy.Close())
	dedicated, err := manager.Rebind(context.Background(), aID, port)
	require.NoError(t, err)
	require.Equal(t, port, dedicated.Port)
	get(dedicated, "Bearer synthetic", 200, aID)
	get(b, "Bearer synthetic", 200, bID)
	require.EqualValues(t, 1, closed.Load())
	require.NoError(t, manager.Stop(context.Background(), aID))
	_, ok = manager.Get(aID)
	require.False(t, ok)
	require.EqualValues(t, 2, closed.Load())
	get(b, "Bearer synthetic", 200, bID)
	require.NoError(t, manager.Close(context.Background()))
	require.EqualValues(t, 3, closed.Load())
}

func TestHTTPFactoryFailureReleasesReservedPort(t *testing.T) {
	_, err := NewHTTP(nil)
	require.Error(t, err)
	var reserved string
	manager, err := NewHTTP(func(_ context.Context, endpoint Endpoint) (http.Handler, error) {
		reserved = endpoint.Address
		return nil, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	_, err = manager.Start(context.Background(), strings.Repeat("d", 64), 0)
	require.Error(t, err)
	require.NotEmpty(t, reserved)
	listener, err := net.Listen("tcp", reserved)
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}

func TestHTTPManagerCommitsStateBeforeReplacingOrStopping(t *testing.T) {
	var closed atomic.Int32
	manager, err := NewHTTP(func(_ context.Context, endpoint Endpoint) (http.Handler, error) {
		return &protectedHTTP{id: endpoint.NamespaceID, closed: &closed}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	id := strings.Repeat("e", 64)
	original, err := manager.Start(context.Background(), id, 0)
	require.NoError(t, err)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := busy.Addr().(*net.TCPAddr).Port
	commits := 0
	_, err = manager.RebindWithCommit(context.Background(), id, port, func(Endpoint) error {
		commits++
		return nil
	})
	require.Error(t, err)
	require.Zero(t, commits, "port reservation must precede persistence")
	require.NoError(t, busy.Close())
	rejected := errors.New("stale settings revision")
	var reserved string
	_, err = manager.RebindWithCommit(context.Background(), id, port, func(endpoint Endpoint) error {
		commits++
		reserved = endpoint.Address
		other, collision := net.Listen("tcp", reserved)
		if other != nil {
			_ = other.Close()
		}
		require.Error(t, collision, "candidate port must remain reserved during commit")
		return rejected
	})
	require.ErrorIs(t, err, rejected)
	current, ok := manager.Get(id)
	require.True(t, ok)
	require.Equal(t, original, current)
	require.EqualValues(t, 1, closed.Load(), "rejected candidate handler must be closed")
	available, err := net.Listen("tcp", reserved)
	require.NoError(t, err, "rejected candidate port must be released")
	require.NoError(t, available.Close())
	replacement, err := manager.RebindWithCommit(context.Background(), id, port, func(endpoint Endpoint) error {
		commits++
		require.Equal(t, port, endpoint.Port)
		return nil
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, closed.Load(), "old handler closes only after successful commitment")
	_, err = manager.RebindWithCommit(context.Background(), id, port, func(endpoint Endpoint) error {
		commits++
		require.Equal(t, replacement, endpoint)
		return rejected
	})
	require.ErrorIs(t, err, rejected, "unchanged endpoint still commits reviewed state")
	err = manager.StopWithCommit(context.Background(), id, func(Endpoint) error { return rejected })
	require.ErrorIs(t, err, rejected)
	current, ok = manager.Get(id)
	require.True(t, ok)
	require.Equal(t, replacement, current)
	err = manager.StopWithCommit(context.Background(), id, func(endpoint Endpoint) error {
		commits++
		require.Equal(t, replacement, endpoint)
		return nil
	})
	require.NoError(t, err)
	_, ok = manager.Get(id)
	require.False(t, ok)
	require.Equal(t, 4, commits)
	require.EqualValues(t, 3, closed.Load())
}
