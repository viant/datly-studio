package predicates

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/studio/host"
	_ "modernc.org/sqlite"
	"reflect"
	"testing"
)

func TestOrderPresenceBothBranchesAndPackageScan(t *testing.T) {
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil || !catalog.Contains(PackagePath, "OrderPresence") {
		t.Fatalf("package scan: %v", err)
	}
	found := false
	for _, item := range catalog.Descriptors() {
		if item.Package == PackagePath && item.TypeName == "OrderPresence" {
			found = true
			if item.Description == "" || item.Description != new(OrderPresence).Description() {
				t.Fatalf("description: %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("described example missing")
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"CREATE TABLE predicate_presence_customers(id INTEGER PRIMARY KEY)",
		"CREATE TABLE predicate_presence_orders(customer_id INTEGER,status TEXT)",
		"CREATE TABLE predicate_presence_exclusions(customer_id INTEGER,active INTEGER)",
		"INSERT INTO predicate_presence_exclusions VALUES(10,0),(30,1),(40,1)",
		"INSERT INTO predicate_presence_customers VALUES(10),(20),(30),(40)",
		"INSERT INTO predicate_presence_orders VALUES(10,'paid'),(10,'paid'),(20,'pending'),(30,'paid')",
	} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		mode string
		ids  []int
	}{{"eligible", []int{10}}, {"exists", []int{10, 30}}, {"not_exists", []int{20, 40}}, {"exists", []int{10, 30}}} {
		criteria, err := (&OrderPresence{}).Compute(context.Background(), tc.mode)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := db.Query("SELECT c.id FROM predicate_presence_customers c WHERE "+criteria.Expression+" ORDER BY c.id", criteria.Placeholders...)
		if err != nil {
			t.Fatal(err)
		}
		var ids []int
		for rows.Next() {
			var id int
			if err = rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		if !reflect.DeepEqual(ids, tc.ids) {
			t.Fatalf("%s: got %v want %v", tc.mode, ids, tc.ids)
		}
	}
	for _, bad := range []any{nil, true, "", "exists OR 1=1"} {
		if _, err := (&OrderPresence{}).Compute(context.Background(), bad); err == nil {
			t.Fatalf("accepted invalid selector %v", bad)
		}
	}
}
