package host

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNamespaceReadersShareNativeConnectorPool(t *testing.T) {
	ctx := context.Background()
	definitions := make([]definition, 40)
	for i := range definitions {
		definitions[i] = definition{reportID: fmt.Sprintf("reader-%d", i), scope: "example.com/reports", name: "reader", connector: "main", dql: "SELECT * FROM records"}
	}
	active := map[string]connectorDefinition{"main": {name: "main", driver: "sqlite", dsn: "file:" + filepath.Join(t.TempDir(), "shared.db")}, "unused": {name: "unused", driver: "missing-driver", dsn: "must-not-open"}}
	pool, err := openRuntimeConnectorSet(ctx, definitions, active, nil, nil)
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
	pool, err := openRuntimeConnectorSet(ctx, definitions, active, nil, nil)
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
	if rejected, err := openRuntimeConnectorSet(ctx, definitions, map[string]connectorDefinition{"main": candidate}, nil, nil); err == nil || rejected != nil {
		t.Fatal("invalid candidate configuration was accepted")
	}
	if err = reader.sql.DB.PingContext(ctx); err != nil {
		t.Fatalf("candidate failure closed active pool: %v", err)
	}
}

func TestIndependentInputViewConnectorIsOpened(t *testing.T) {
	for _, fixture := range []struct {
		kind     string
		embedded bool
	}{{"view", false}, {"data_view", false}, {"view", true}, {"data_view", true}} {
		t.Run(fmt.Sprintf("%s/embedded=%v", fixture.kind, fixture.embedded), func(t *testing.T) {
			ctx := t.Context()
			reader := definition{reportID: "reader", scope: "example.com/reports", name: "reader", connector: "main", dql: fmt.Sprintf(`#setting($_ = $route('/records', 'GET'))
#package('example.com/reports')
#define($_ = $Context<?>(%s/context).Cardinality('One').WithConnector('lookup') /* SELECT name FROM settings */)
SELECT records.*, use_connector(records, 'main') FROM (SELECT id FROM records) records`, fixture.kind)}
			var resources fs.FS
			if fixture.embedded {
				reader.dql = strings.ReplaceAll(reader.dql, "SELECT name FROM settings", "SELECT settings.* FROM (${embed:sql/context.sql}) settings")
				reader.dql = strings.ReplaceAll(reader.dql, "SELECT id FROM records", "${embed:sql/records.sql}")
				resources = fstest.MapFS{
					"sql/context.sql": &fstest.MapFile{Data: []byte("SELECT name FROM settings")},
					"sql/records.sql": &fstest.MapFile{Data: []byte("SELECT id FROM records")},
				}
			}
			resourceFS := func(definition) fs.FS { return resources }
			if names := referencedRuntimeConnectors(ctx, reader, []string{"main", "lookup", "unused"}, resources, nil); !reflect.DeepEqual(names, []string{"lookup", "main"}) {
				t.Fatalf("connector identity closure = %v", names)
			}
			active := map[string]connectorDefinition{
				"main":   {name: "main", driver: "sqlite", dsn: "file:" + filepath.Join(t.TempDir(), "main.db")},
				"lookup": {name: "lookup", driver: "sqlite", dsn: "file:" + filepath.Join(t.TempDir(), "lookup.db")},
				"unused": {name: "unused", driver: "missing-driver", dsn: "must-not-open"},
			}
			pool, err := openRuntimeConnectorSet(ctx, []definition{reader}, active, nil, resourceFS)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			if len(pool.connections) != 2 || pool.connections["main"] == nil || pool.connections["lookup"] == nil {
				t.Fatalf("independent view connector omitted: %v", pool.connections)
			}
			if err := pool.connections["lookup"].PingContext(ctx); err != nil {
				t.Fatal(err)
			}
			delete(active, "lookup")
			if candidate, err := openRuntimeConnectorSet(ctx, []definition{reader}, active, nil, resourceFS); err == nil || candidate != nil {
				t.Fatal("missing explicitly declared input-view connector was accepted")
			}
		})
	}
}
