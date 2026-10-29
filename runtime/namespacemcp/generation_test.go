package namespacemcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type generationHTTP struct {
	generation atomic.Int64
	closed     atomic.Int32
	reload     func(context.Context, int64) error
}

func (h *generationHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, h.generation.Load())
}
func (h *generationHTTP) Close(context.Context) error { h.closed.Add(1); return nil }
func (h *generationHTTP) Reload(ctx context.Context, generation int64) error {
	if h.reload != nil {
		if err := h.reload(ctx, generation); err != nil {
			return err
		}
	}
	h.generation.Store(generation)
	return nil
}
func generationPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}
func generationBody(t *testing.T, endpoint Endpoint) string {
	t.Helper()
	response, err := (&http.Client{Timeout: time.Second}).Get(endpoint.URL())
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	return string(body)
}

func TestRebindAndReloadBuildsBeforeRegistrationAndPreservesLastGood(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(fmt.Sprint("shared=", shared), func(t *testing.T) {
			ctx := context.Background()
			id, other := strings.Repeat("a", 64), strings.Repeat("b", 64)
			entered, release := make(chan struct{}), make(chan struct{})
			var sources []*generationHTTP
			factory := func(ctx context.Context, endpoint Endpoint) (http.Handler, error) {
				require.True(t, DeferredInitialLoad(ctx, id))
				require.False(t, DeferredInitialLoad(ctx, other))
				source := &generationHTTP{reload: func(ctx context.Context, generation int64) error {
					if generation == 2 {
						close(entered)
						select {
						case <-release:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					if generation == 9 {
						return errors.New("candidate failed")
					}
					return nil
				}}
				sources = append(sources, source)
				return source, nil
			}
			var manager *Manager
			var err error
			port := generationPort(t)
			if shared {
				manager, err = NewSharedHTTP(factory, "127.0.0.1:0")
				port = 0
			} else {
				manager, err = NewHTTP(factory)
			}
			require.NoError(t, err)
			defer manager.Close(ctx)
			result := make(chan error, 1)
			go func() { _, err := manager.RebindAndReload(ctx, id, port, 2); result <- err }()
			<-entered
			_, exists := manager.Get(id)
			require.False(t, exists, "unbuilt candidate must not be published")
			close(release)
			require.NoError(t, <-result)
			endpoint, exists := manager.Get(id)
			require.True(t, exists)
			require.Equal(t, "2", generationBody(t, endpoint))
			unchanged, err := manager.RebindAndReload(ctx, id, port, 3)
			require.NoError(t, err)
			require.Equal(t, endpoint, unchanged)
			require.Len(t, sources, 1, "warm reload should reuse the source")
			require.Equal(t, "3", generationBody(t, endpoint))
			_, err = manager.RebindAndReload(ctx, id, port, 9)
			require.ErrorContains(t, err, "candidate failed")
			require.Equal(t, "3", generationBody(t, endpoint))
			target := generationPort(t)
			_, err = manager.RebindAndReload(ctx, id, target, 9)
			require.ErrorContains(t, err, "candidate failed")
			retained, exists := manager.Get(id)
			require.True(t, exists)
			require.Equal(t, endpoint, retained)
			require.EqualValues(t, 1, sources[1].closed.Load())
			require.Equal(t, "3", generationBody(t, endpoint))
			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", target))
			require.NoError(t, err, "failed build must release its reserved port")
			require.NoError(t, listener.Close())
			replacement, err := manager.RebindAndReload(ctx, id, target, 4)
			require.NoError(t, err)
			require.Equal(t, "4", generationBody(t, replacement))
			require.EqualValues(t, 1, sources[0].closed.Load())
		})
	}
}

func TestRebindAndReloadColdFailureCancellationAndUnsupportedSource(t *testing.T) {
	ctx := context.Background()
	id := strings.Repeat("c", 64)
	var source *generationHTTP
	manager, err := NewHTTP(func(ctx context.Context, endpoint Endpoint) (http.Handler, error) {
		source = &generationHTTP{reload: func(ctx context.Context, generation int64) error { return errors.New("build failed") }}
		return source, nil
	})
	require.NoError(t, err)
	defer manager.Close(ctx)
	port := generationPort(t)
	_, err = manager.RebindAndReload(ctx, id, port, -1)
	require.Error(t, err)
	require.Nil(t, source)
	_, err = manager.RebindAndReload(ctx, id, port, 2)
	require.ErrorContains(t, err, "build failed")
	require.EqualValues(t, 1, source.closed.Load())
	_, exists := manager.Get(id)
	require.False(t, exists)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = manager.RebindAndReload(cancelled, id, port, 2)
	require.ErrorIs(t, err, context.Canceled)
	var closed atomic.Int32
	unsupported, err := NewHTTP(func(ctx context.Context, endpoint Endpoint) (http.Handler, error) {
		return &protectedHTTP{id: id, closed: &closed}, nil
	})
	require.NoError(t, err)
	defer unsupported.Close(ctx)
	_, err = unsupported.RebindAndReload(ctx, id, 0, 2)
	require.ErrorContains(t, err, "reload is unavailable")
	require.EqualValues(t, 1, closed.Load())
	_, exists = unsupported.Get(id)
	require.False(t, exists)
}

func TestRebindAndReloadSerializesLifecycleWithoutBlockingOtherNamespaces(t *testing.T) {
	ctx := context.Background()
	a, b := strings.Repeat("d", 64), strings.Repeat("e", 64)
	entered, release := make(chan struct{}), make(chan struct{})
	var source *generationHTTP
	manager, err := NewSharedHTTP(func(ctx context.Context, endpoint Endpoint) (http.Handler, error) {
		h := &generationHTTP{}
		if endpoint.NamespaceID == a {
			source = h
			h.reload = func(ctx context.Context, generation int64) error { close(entered); <-release; return nil }
		}
		return h, nil
	}, "127.0.0.1:0")
	require.NoError(t, err)
	defer manager.Close(ctx)
	built := make(chan error, 1)
	go func() { _, err := manager.RebindAndReload(ctx, a, 0, 2); built <- err }()
	<-entered
	stopped := make(chan error, 1)
	go func() { stopped <- manager.Stop(ctx, a) }()
	select {
	case err := <-stopped:
		t.Fatalf("Stop passed an incomplete build: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	_, err = manager.RebindAndReload(ctx, b, 0, 3)
	require.NoError(t, err)
	close(release)
	require.NoError(t, <-built)
	require.NoError(t, <-stopped)
	require.EqualValues(t, 1, source.closed.Load())
	_, exists := manager.Get(a)
	require.False(t, exists)
	_, exists = manager.Get(b)
	require.True(t, exists)
}

func TestRebindAndReloadRejectsCandidateAfterCloseOrCancellation(t *testing.T) {
	for _, closing := range []bool{false, true} {
		t.Run(fmt.Sprint("close=", closing), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			id := strings.Repeat("f", 64)
			entered, release := make(chan struct{}), make(chan struct{})
			source := &generationHTTP{reload: func(context.Context, int64) error { close(entered); <-release; return nil }}
			manager, err := NewHTTP(func(ctx context.Context, endpoint Endpoint) (http.Handler, error) { return source, nil })
			require.NoError(t, err)
			defer manager.Close(context.Background())
			done := make(chan error, 1)
			go func() { _, err := manager.RebindAndReload(ctx, id, 0, 2); done <- err }()
			<-entered
			if closing {
				require.NoError(t, manager.Close(context.Background()))
			} else {
				cancel()
			}
			close(release)
			require.Error(t, <-done)
			require.EqualValues(t, 1, source.closed.Load())
			_, exists := manager.Get(id)
			require.False(t, exists)
		})
	}
}
