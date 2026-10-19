package host

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestNamespaceReadersShareNativeConnectorPool(t *testing.T) {
	ctx := context.Background()
	definitions := make([]definition, 40)
	for i := range definitions {
		definitions[i] = definition{reportID: fmt.Sprintf("reader-%d", i), scope: "example.com/reports", name: "reader", connector: "main", dql: "SELECT * FROM records"}
	}
	active := map[string]connectorDefinition{"main": {name: "main", driver: "sqlite", dsn: "file:" + filepath.Join(t.TempDir(), "shared.db")}, "unused": {name: "unused", driver: "missing-driver", dsn: "must-not-open"}}
	pool, err := openRuntimeConnectorSet(ctx, definitions, active, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := pool.sources(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	for range definitions {
		reader, err := pool.sources(ctx, "main")
		if err != nil {
			t.Fatal(err)
		}
		if reader.sql.DB != first.sql.DB || reader.connections["main"] != first.sql.DB {
			t.Fatal("reader opened a separate connector pool")
		}
		var value int
		if err = reader.sql.DB.QueryRowContext(ctx, "SELECT 1").Scan(&value); err != nil || value != 1 {
			t.Fatalf("read failed: %v", err)
		}
	}
	if first.sql.DB.Stats().MaxOpenConnections != 0 {
		t.Fatal("an application connection limit was imposed")
	}
	if len(pool.connections) != 1 || first.sql.DB.Stats().OpenConnections != 1 {
		t.Fatalf("unexpected pools/connections: pools=%d stats=%+v", len(pool.connections), first.sql.DB.Stats())
	}
	if err = pool.Close(); err != nil {
		t.Fatal(err)
	}
	if err = first.sql.DB.PingContext(ctx); err == nil {
		t.Fatal("retired generation did not close its owned pool")
	}
}

func TestFailedCandidateDoesNotCloseActiveGenerationPool(t *testing.T) {
	ctx := context.Background()
	definitions := []definition{{reportID: "reader", connector: "main", dql: "SELECT * FROM records"}}
	active := map[string]connectorDefinition{"main": {name: "main", driver: "sqlite", dsn: "file:" + filepath.Join(t.TempDir(), "shared.db")}}
	pool, err := openRuntimeConnectorSet(ctx, definitions, active, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	reader, err := pool.sources(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	candidate := active["main"]
	candidate.options = []byte(`{"sqliteAttachments":{"bad-name":"missing.db"}}`)
	if rejected, err := openRuntimeConnectorSet(ctx, definitions, map[string]connectorDefinition{"main": candidate}, nil); err == nil || rejected != nil {
		t.Fatal("invalid candidate configuration was accepted")
	}
	if err = reader.sql.DB.PingContext(ctx); err != nil {
		t.Fatalf("candidate failure closed active pool: %v", err)
	}
}
