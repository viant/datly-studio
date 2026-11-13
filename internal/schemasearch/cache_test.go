package schemasearch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/sdk"
)

func fixtureScope() Scope {
	return Scope{CallerIdentity: "issuer:subject", ConnectorIdentity: "source", CredentialIdentity: "credential-v1", ConnectorRevision: 1, Catalog: "catalog", Schema: "dataset"}
}
func fixtureMetadata() *Metadata {
	return &Metadata{Complete: true, Tables: []sdk.TableDetail{{Table: sdk.DatabaseTable{Name: "records", Schema: "dataset"}, Columns: []sdk.DatabaseColumn{{Name: "id"}}}}}
}
func TestSnapshotReuseExpiryRefreshAndScopeIsolation(t *testing.T) {
	cache := NewCache(CacheOptions{TTL: time.Minute})
	defer cache.Close()
	now := time.Now()
	cache.now = func() time.Time { return now }
	calls := 0
	load := func(context.Context) (*Metadata, error) { calls++; return fixtureMetadata(), nil }
	key := fixtureScope()
	first, err := cache.Get(context.Background(), key, false, load)
	require.NoError(t, err)
	warm, err := cache.Get(context.Background(), key, false, load)
	require.NoError(t, err)
	require.Same(t, first, warm)
	require.Equal(t, 1, calls)
	now = now.Add(time.Minute)
	expired, err := cache.Get(context.Background(), key, false, load)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, expired.ID)
	refreshed, err := cache.Get(context.Background(), key, true, load)
	require.NoError(t, err)
	require.NotEqual(t, expired.ID, refreshed.ID)
	for _, mutate := range []func(*Scope){func(k *Scope) { k.CallerIdentity = "another" }, func(k *Scope) { k.CredentialIdentity = "credential-v2" }, func(k *Scope) { k.ConnectorRevision++ }, func(k *Scope) { k.Schema = "other" }} {
		other := key
		mutate(&other)
		value, e := cache.Get(context.Background(), other, false, load)
		require.NoError(t, e)
		require.NotEqual(t, refreshed.ID, value.ID)
	}
	require.Equal(t, 7, calls)
}
func TestSnapshotBuildCoalescingAndCancellation(t *testing.T) {
	cache := NewCache(CacheOptions{})
	defer cache.Close()
	var calls atomic.Int32
	start, release := make(chan struct{}), make(chan struct{})
	load := func(ctx context.Context) (*Metadata, error) {
		if calls.Add(1) == 1 {
			close(start)
		}
		select {
		case <-release:
			return fixtureMetadata(), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	var wg sync.WaitGroup
	results := make(chan *Snapshot, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, e := cache.Get(context.Background(), fixtureScope(), false, load)
			if e != nil {
				t.Error(e)
				return
			}
			results <- v
		}()
	}
	<-start
	require.Eventually(t, func() bool {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		f := cache.pending[fixtureScope()]
		return f != nil && f.users == 12
	}, time.Second, time.Millisecond)
	close(release)
	wg.Wait()
	close(results)
	require.EqualValues(t, 1, calls.Load())
	var first *Snapshot
	for v := range results {
		if first == nil {
			first = v
		}
		require.Same(t, first, v)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cache.Get(canceled, fixtureScope(), false, load)
	require.ErrorIs(t, err, context.Canceled)
}
func TestSnapshotFailuresAreNotEmptyCachedSuccess(t *testing.T) {
	cache := NewCache(CacheOptions{})
	defer cache.Close()
	failure := errors.New("provider denied")
	calls := 0
	load := func(context.Context) (*Metadata, error) { calls++; return nil, failure }
	for i := 0; i < 2; i++ {
		v, e := cache.Get(context.Background(), fixtureScope(), false, load)
		require.Nil(t, v)
		require.ErrorIs(t, e, failure)
	}
	require.Equal(t, 2, calls)
	_, err := cache.Get(context.Background(), Scope{}, false, load)
	require.Error(t, err)
	cache.Close()
	_, err = cache.Get(context.Background(), fixtureScope(), false, load)
	require.Error(t, err)
}
func TestSnapshotBudgetEvictionAndPartialState(t *testing.T) {
	cache := NewCache(CacheOptions{MaxEntries: 1})
	defer cache.Close()
	calls := 0
	load := func(context.Context) (*Metadata, error) {
		calls++
		m := fixtureMetadata()
		m.Complete = false
		m.Continuation = "provider-page"
		return m, nil
	}
	key := fixtureScope()
	first, e := cache.Get(context.Background(), key, false, load)
	require.NoError(t, e)
	require.False(t, first.Complete)
	require.Equal(t, "provider-page", first.Continuation)
	other := key
	other.Schema = "other"
	_, e = cache.Get(context.Background(), other, false, load)
	require.NoError(t, e)
	last, e := cache.Get(context.Background(), key, false, load)
	require.NoError(t, e)
	require.NotEqual(t, first.ID, last.ID)
	require.Equal(t, 3, calls)
	small := NewCache(CacheOptions{MaxBytes: 1})
	defer small.Close()
	_, e = small.Get(context.Background(), key, false, load)
	require.ErrorContains(t, e, "budget")
}

func TestCanceledWaiterDoesNotCancelSharedBuild(t *testing.T) {
	cache := NewCache(CacheOptions{})
	defer cache.Close()
	started, release := make(chan struct{}), make(chan struct{})
	load := func(ctx context.Context) (*Metadata, error) {
		close(started)
		select {
		case <-release:
			return fixtureMetadata(), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	first, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	secondResult := make(chan error, 1)
	go func() { _, err := cache.Get(first, fixtureScope(), false, load); firstResult <- err }()
	<-started
	go func() { _, err := cache.Get(context.Background(), fixtureScope(), false, load); secondResult <- err }()
	require.Eventually(t, func() bool { cache.mu.Lock(); defer cache.mu.Unlock(); return cache.pending[fixtureScope()].users == 2 }, time.Second, time.Millisecond)
	cancel()
	require.ErrorIs(t, <-firstResult, context.Canceled)
	close(release)
	require.NoError(t, <-secondResult)
}
