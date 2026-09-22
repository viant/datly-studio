package bffauth

import (
	"context"
	"sync"
	"time"
)

type memoryStore struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func newMemoryStore() *memoryStore { return &memoryStore{sessions: map[string]session{}} }

func (m *memoryStore) Put(_ context.Context, id string, value session) error {
	m.mu.Lock()
	m.sessions[id] = value
	m.mu.Unlock()
	return nil
}

func (m *memoryStore) Get(_ context.Context, id string) (session, bool, error) {
	m.mu.RLock()
	value, ok := m.sessions[id]
	m.mu.RUnlock()
	return value, ok, nil
}

func (m *memoryStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	return nil
}

func (m *memoryStore) DeleteExpired(_ context.Context, now time.Time) error {
	m.mu.Lock()
	for id, value := range m.sessions {
		if !now.Before(value.expiresAt) {
			delete(m.sessions, id)
		}
	}
	m.mu.Unlock()
	return nil
}
