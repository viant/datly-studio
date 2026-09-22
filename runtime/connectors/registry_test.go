package connectors

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

func TestRegistryBuilderBuild(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	builder := NewRegistryBuilder(WithNow(func() time.Time { return now }))

	source := []*sdk.Connector{
		{
			Name: "warehouse", Driver: "sqlite", Status: "active", OwnerID: "bob", ETag: 2,
			UpdatedAt: now.Add(-time.Minute), Options: json.RawMessage(`{"pool":"large"}`),
		},
		{
			Name: "analytics", Driver: "sqlite", Status: "active", OwnerID: "alice", ETag: 1,
			UpdatedAt: now, Options: json.RawMessage(`{"pool":"small"}`),
		},
	}
	snapshot, err := builder.Build(context.Background(), source)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if snapshot.Revision == "" {
		t.Fatalf("Build() revision is empty")
	}
	if snapshot.CreatedAt != now {
		t.Fatalf("Build() createdAt = %v, want %v", snapshot.CreatedAt, now)
	}
	if len(snapshot.Ordered) != 2 || snapshot.Ordered[0].Name != "analytics" {
		t.Fatalf("Build() ordered connectors = %+v", snapshot.Ordered)
	}

	got, ok := snapshot.Get("warehouse")
	if !ok || got.Name != "warehouse" {
		t.Fatalf("Get(warehouse) = %+v, %v", got, ok)
	}

	source[0].Options[0] = '['
	if string(snapshot.Connectors["warehouse"].Options) != `{"pool":"large"}` {
		t.Fatalf("input JSON mutation leaked into snapshot: %s", snapshot.Connectors["warehouse"].Options)
	}
}

func TestRegistryBuilderRejectsDuplicates(t *testing.T) {
	t.Parallel()

	builder := NewRegistryBuilder()
	_, err := builder.Build(context.Background(), []*sdk.Connector{
		{Name: "analytics"},
		{Name: "analytics"},
	})
	if err == nil {
		t.Fatalf("expected duplicate connector error")
	}
}
