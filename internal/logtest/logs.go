// Package logtest captures structured server logs in nonparallel tests.
package logtest

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"sync"
	"testing"
)

// Logs captures the existing structured server sink for nonparallel tests.
type Logs struct {
	mu      sync.Mutex
	entries []map[string]any
	changed chan struct{}
}

func CaptureLogs(t testing.TB) *Logs {
	t.Helper()
	previous, writer, flags := slog.Default(), log.Writer(), log.Flags()
	result := &Logs{changed: make(chan struct{}, 1)}
	slog.SetDefault(slog.New(slog.NewJSONHandler(result, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous); log.SetOutput(writer); log.SetFlags(flags) })
	return result
}
func (l *Logs) Write(data []byte) (int, error) {
	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		return 0, err
	}
	l.mu.Lock()
	l.entries = append(l.entries, entry)
	l.mu.Unlock()
	select {
	case l.changed <- struct{}{}:
	default:
	}
	return len(data), nil
}
func (l *Logs) Entries() []map[string]any {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]map[string]any(nil), l.entries...)
}
func (l *Logs) Wait(ctx context.Context, t testing.TB, event string) map[string]any {
	t.Helper()
	for {
		for _, entry := range l.Entries() {
			if entry["msg"] == event {
				return entry
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("missing log event %q: %v", event, ctx.Err())
			return nil
		case <-l.changed:
		}
	}
}
