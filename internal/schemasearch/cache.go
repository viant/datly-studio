package schemasearch

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/viant/datly-studio/sdk"
)

// Scope is constructed by the authorized server boundary, never from tool input.
// CredentialIdentity is an opaque fingerprint/version, not credential material.
type Scope struct {
	CallerIdentity, ConnectorIdentity, CredentialIdentity string
	ConnectorRevision                                     int64
	Catalog, Schema                                       string
	Table, MetadataCursor                                 string
}
type Metadata struct {
	Tables       []sdk.TableDetail
	Complete     bool
	Continuation string
}
type Snapshot struct {
	Index        *Index
	Complete     bool
	Continuation string
	ExpiresAt    time.Time
	ID           string
}
type Loader func(context.Context) (*Metadata, error)
type CacheOptions struct {
	TTL, BuildTimeout                         time.Duration
	MaxEntries, MaxBytes, MaxConcurrentBuilds int
}
type cachedSnapshot struct {
	key   Scope
	value *Snapshot
	bytes int
}
type pendingSnapshot struct {
	done     chan struct{}
	cancel   context.CancelFunc
	users    int
	finished bool
	value    *Snapshot
	err      error
}

// Cache owns metadata indexes for one host. Authorization is deliberately not
// cached here: callers must authorize and resolve current source identity first.
type Cache struct {
	slots    chan struct{}
	identity string
	closed   bool
	mu       sync.Mutex
	options  CacheOptions
	entries  map[Scope]*list.Element
	lru      *list.List
	pending  map[Scope]*pendingSnapshot
	bytes    int
	sequence atomic.Uint64
	now      func() time.Time
}

func NewCache(options CacheOptions) *Cache {
	if options.TTL <= 0 {
		options.TTL = 5 * time.Minute
	}
	if options.BuildTimeout <= 0 {
		options.BuildTimeout = 30 * time.Second
	}
	if options.MaxEntries <= 0 {
		options.MaxEntries = 16
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = 64 << 20
	}
	if options.MaxConcurrentBuilds <= 0 {
		options.MaxConcurrentBuilds = 4
	}
	return &Cache{slots: make(chan struct{}, options.MaxConcurrentBuilds), identity: uuid.NewString(), options: options, entries: map[Scope]*list.Element{}, pending: map[Scope]*pendingSnapshot{}, lru: list.New(), now: time.Now}
}
func (c *Cache) Get(ctx context.Context, key Scope, refresh bool, load Loader) (*Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if key.CallerIdentity == "" || key.ConnectorIdentity == "" || key.CredentialIdentity == "" || key.Schema == "" || load == nil {
		return nil, fmt.Errorf("schema snapshot requires trusted caller, connector, credential and database/dataset scope")
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("schema snapshot owner is closed")
	}
	if e := c.entries[key]; e != nil {
		held := e.Value.(*cachedSnapshot)
		if !refresh && c.now().Before(held.value.ExpiresAt) {
			c.lru.MoveToFront(e)
			c.mu.Unlock()
			return held.value, nil
		}
		c.remove(e)
	}
	flight := c.pending[key]
	if flight == nil {
		buildCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.options.BuildTimeout)
		flight = &pendingSnapshot{done: make(chan struct{}), cancel: cancel}
		c.pending[key] = flight
		go c.build(buildCtx, key, flight, load)
	}
	flight.users++
	c.mu.Unlock()
	defer c.release(key, flight)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-flight.done:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return flight.value, flight.err
	}
}
func (c *Cache) release(key Scope, f *pendingSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f.users--
	if f.users == 0 && !f.finished {
		f.cancel()
		if c.pending[key] == f {
			delete(c.pending, key)
		}
	}
}
func (c *Cache) build(ctx context.Context, key Scope, f *pendingSnapshot, load Loader) {
	defer f.cancel()
	var metadata *Metadata
	var err error
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
		metadata, err = load(ctx)
	case <-ctx.Done():
		err = ctx.Err()
	}
	var value *Snapshot
	size := 0
	if err == nil && metadata == nil {
		err = fmt.Errorf("schema metadata loader returned no snapshot")
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		size = estimatedIndexBytes(metadata)
		if size > c.options.MaxBytes {
			err = fmt.Errorf("schema metadata exceeds the bounded snapshot budget; narrow discovery scope")
		}
	}
	if err == nil {
		id := fmt.Sprintf("schema:%s:%d", c.identity, c.sequence.Add(1))
		value = &Snapshot{Index: New(id, metadata.Tables), ID: id, Complete: metadata.Complete, Continuation: metadata.Continuation}
		err = ctx.Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil && !c.closed && c.pending[key] == f && f.users > 0 {
		value.ExpiresAt = c.now().Add(c.options.TTL)
		for c.lru.Len() >= c.options.MaxEntries || c.bytes+size > c.options.MaxBytes {
			c.remove(c.lru.Back())
		}
		if e := c.entries[key]; e != nil {
			c.remove(e)
		}
		c.entries[key] = c.lru.PushFront(&cachedSnapshot{key: key, value: value, bytes: size})
		c.bytes += size
	}
	f.value, f.err, f.finished = value, err, true
	if c.pending[key] == f {
		delete(c.pending, key)
	}
	close(f.done)
}
func (c *Cache) remove(e *list.Element) {
	if e == nil {
		return
	}
	item := e.Value.(*cachedSnapshot)
	delete(c.entries, item.key)
	c.bytes -= item.bytes
	c.lru.Remove(e)
}

// Conservative admission estimate includes token/index expansion, not just raw
// JSON. It is a retention budget, not a precise allocator accounting API.
func estimatedIndexBytes(m *Metadata) int {
	n := 256
	for _, t := range m.Tables {
		n += 512 + 32*(len(t.Table.Catalog)+len(t.Table.Schema)+len(t.Table.Name)+len(t.Table.Comment))
		for _, col := range t.Columns {
			n += 512 + 32*(len(col.Name)+len(col.Description)+len(t.Table.Catalog)+len(t.Table.Schema)+len(t.Table.Name))
		}
	}
	return n
}

// Close cancels owned builds and releases retained metadata on host shutdown.
func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for _, flight := range c.pending {
		flight.cancel()
	}
	c.entries = map[Scope]*list.Element{}
	c.lru.Init()
	c.bytes = 0
}
