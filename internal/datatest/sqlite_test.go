package datatest

import (
	"context"
	"testing"
)

func TestHydrateAndAssertRows(t *testing.T) {
	ctx := context.Background()
	db := OpenSQLite(t, "hydrate", "studio")
	if err := Hydrate(ctx, db, Table{Name: "connectors", Rows: []Row{
		{"name": "alpha", "driver": "sqlite", "owner_id": "owner-a", "status": "active", "etag": 1, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
		{"name": "beta", "driver": "mysql", "owner_id": "owner-b", "status": "disabled", "etag": 2, "created_at": "2026-09-17 11:00:00", "updated_at": "2026-09-17 11:00:00"},
	}}); err != nil {
		t.Fatal(err)
	}
	AssertRows(t, ctx, db, "SELECT name, etag FROM connectors ORDER BY name", nil,
		Row{"name": "alpha", "etag": 1}, Row{"name": "beta", "etag": 2})
}

func TestJSONHydrationPhaseValidatesAndNormalizes(t *testing.T) {
	ctx := context.Background()
	db := OpenSQLite(t, "json_hydrate", "studio")
	phase := HydrationPhase{JSON: []byte(`{
  "tables": [
    {"name":"connectors","rows":[
      {"name":"json","driver":"sqlite","owner_id":"owner","status":"active","options_json":{"max_open":2},"created_at":"2026-09-17 10:00:00","updated_at":"2026-09-17 10:00:00"}
    ]}
  ]
}`)}
	if err := phase.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	AssertRowsJSON(t, ctx, db, "SELECT name,options_json,etag FROM connectors", nil, []byte(`[{"name":"json","options_json":"{\"max_open\":2}","etag":1}]`))

	for _, invalid := range [][]byte{
		[]byte(`{"tables":[{"name":"missing","rows":[{"id":1}]}]}`),
		[]byte(`{"tables":[{"name":"connectors","rows":[{"unknown":1}]}]}`),
		[]byte(`{"tables":[{"name":"connectors","rows":[{"name":"x"}]}]}`),
	} {
		if err := (HydrationPhase{JSON: invalid}).Apply(ctx, db); err == nil {
			t.Fatalf("invalid hydration dataset accepted: %s", invalid)
		}
	}
}
