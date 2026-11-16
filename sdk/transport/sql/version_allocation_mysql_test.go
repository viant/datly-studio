package sqltransport

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

// The DSN must identify an empty, task-owned MySQL database configured with
// ANSI_QUOTES, matching the generated SQL contract.
func TestVersionAllocationIndependentConnectionsMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_ALLOCATION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires task-owned STUDIO_ALLOCATION_MYSQL_DSN")
	}
	base, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx := sdk.WithPrincipal(base, sdk.Principal{Subject: "owner"})
	open := func() *sql.DB {
		t.Helper()
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(4)
		t.Cleanup(func() { db.Close() })
		if err = db.PingContext(ctx); err != nil {
			t.Fatal(err)
		}
		return db
	}
	first, second := open(), open()
	ddl, err := schema.MySQLDDL()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	transports := []*Transport{{DB: first, Authorizer: allowAuthorizer{}}, {DB: second, Authorizer: allowAuthorizer{}}}
	clients := make([]sdk.Client, len(transports))
	for i, transport := range transports {
		clients[i], err = sdk.NewClient(transport)
		if err != nil {
			t.Fatal(err)
		}
	}
	connector, err := clients[0].Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "mysql"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.ExecContext(ctx, "UPDATE connectors SET status='active' WHERE name=?", connector.Name); err != nil {
		t.Fatal(err)
	}
	// Component setup is isolated fixture SQL; every allocation below runs the
	// real SDK/native graph and transaction, independently of component authoring.
	report := &sdk.Component{ID: "allocation-fixture", OwnerID: "owner", Namespace: "general"}
	namespaceID := namespaceaccess.ID(report.OwnerID, report.Namespace)
	now := time.Now().UTC()
	if _, err = first.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,created_at,updated_at) VALUES(?,?,'general','General','active',?,?)`, namespaceID, report.OwnerID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = first.ExecContext(ctx, `INSERT INTO components(id,namespace_id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES(?,?,'general','allocation','Allocation',?,'active',?,'fixture/allocation','allocation',?,?)`, report.ID, namespaceID, report.OwnerID, connector.Name, now, now); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		version *sdk.ReportVersion
		err     error
	}
	results := make(chan outcome, 8)
	var workers sync.WaitGroup
	for _, client := range clients {
		workers.Add(1)
		go func(client sdk.Client) {
			defer workers.Done()
			for i := 0; i < 4; i++ {
				v, e := client.Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "sql", AuthoredSQL: "SELECT 1"})
				results <- outcome{v, e}
			}
		}(client)
	}
	workers.Wait()
	close(results)
	seen := map[int]bool{}
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if seen[result.version.VersionNo] {
			t.Fatalf("duplicate allocation %d", result.version.VersionNo)
		}
		seen[result.version.VersionNo] = true
	}
	for n := 1; n <= 8; n++ {
		if !seen[n] {
			t.Fatalf("missing committed version %d: %v", n, seen)
		}
	}
	if _, err = first.ExecContext(ctx, "DELETE FROM component_versions WHERE report_id=? AND version_no=8", report.ID); err != nil {
		t.Fatal(err)
	}
	next, err := clients[1].Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "sql", AuthoredSQL: "SELECT 9"})
	if err != nil || next.VersionNo != 9 {
		t.Fatalf("create after highest deletion=%+v err=%v", next, err)
	}
	// Clone/removal use this same native lock reader before their snapshots.
	// Holding it must block a creator using an independent connection pool.
	tx, err := first.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	number, err := transports[0].nextVersionNo(ctx, tx, report.ID, namespaceaccess.ID(report.OwnerID, report.Namespace))
	if err != nil || number != 10 {
		t.Fatalf("lock=%d err=%v", number, err)
	}
	blocked := make(chan outcome, 1)
	go func() {
		v, e := clients[1].Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "structured"})
		blocked <- outcome{v, e}
	}()
	select {
	case result := <-blocked:
		t.Fatalf("creator escaped held component lock: %+v", result)
	case <-time.After(150 * time.Millisecond):
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	result := <-blocked
	if result.err != nil || result.version.VersionNo != 10 {
		t.Fatalf("creator after release=%+v err=%v", result.version, result.err)
	}
	imported, err := clients[0].Versions().LoadDQL(ctx, report.ID, sdk.LoadDQLInput{DQL: "SELECT 11"})
	if err != nil || imported.Version.VersionNo != 11 {
		t.Fatalf("import after creators=%+v err=%v", imported, err)
	}
	var high int
	if err = second.QueryRowContext(ctx, "SELECT last_version_no FROM components WHERE id=?", report.ID).Scan(&high); err != nil || high != 11 {
		t.Fatalf("committed high-water=%d err=%v", high, err)
	}
}
