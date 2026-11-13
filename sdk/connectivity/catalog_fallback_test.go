package connectivity

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/viant/datly-studio/sdk"
	sqlitemeta "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/sqlx/metadata/registry"
	"github.com/viant/sqlx/metadata/sink"
	"modernc.org/sqlite"
	"reflect"
)

var catalogDenied = errors.New("catalog metadata unavailable")
var fallbackLog struct {
	sync.Mutex
	queries []string
}

type fallbackDriver struct{ sqlite.Driver }
type fallbackConn struct{ driver.Conn }

func (d *fallbackDriver) Open(name string) (driver.Conn, error) {
	c, e := d.Driver.Open(name)
	return &fallbackConn{c}, e
}
func (c *fallbackConn) Prepare(q string) (driver.Stmt, error) {
	fallbackLog.Lock()
	fallbackLog.queries = append(fallbackLog.queries, q)
	fallbackLog.Unlock()
	if strings.Contains(q, "pragma_table_info") {
		return nil, catalogDenied
	}
	return c.Conn.Prepare(q)
}
func (c *fallbackConn) QueryContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	fallbackLog.Lock()
	fallbackLog.queries = append(fallbackLog.queries, q)
	fallbackLog.Unlock()
	if strings.Contains(q, "pragma_table_info") {
		return nil, catalogDenied
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, a)
}
func init() {
	sql.Register("catalog_fallback_sqlite", &fallbackDriver{})
	typ := reflect.TypeOf(fallbackDriver{})
	if err := registry.RegisterDriver(*sqlitemeta.SQLite3(), typ.PkgPath(), typ.Name()); err != nil {
		panic(err)
	}
}

func TestCatalogNativeZeroRowFallback(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "source.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE "odd; name"(id INTEGER PRIMARY KEY, label TEXT); INSERT INTO "odd; name" VALUES(1,'business row');`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	fallbackLog.Lock()
	fallbackLog.queries = nil
	fallbackLog.Unlock()
	result, err := (SQLCatalog{}).Table(context.Background(), &sdk.Connector{Name: "source", Driver: "catalog_fallback_sqlite", DSNTemplate: dsn}, sdk.TableDetailInput{Table: "odd; name"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Severity != "warning" || result.Diagnostics[0].Code != "column_metadata_fallback" {
		t.Fatalf("missing primary failure warning: %+v", result.Diagnostics)
	}
	if len(result.Columns) != 2 || result.MetadataSource != "zero_row_probe" || result.ConstraintsKnown {
		t.Fatalf("result=%+v", result)
	}
	fallbackLog.Lock()
	defer fallbackLog.Unlock()
	probes := 0
	for _, q := range fallbackLog.queries {
		if strings.HasPrefix(q, "SELECT *") {
			probes++
			if q != `SELECT * FROM "main"."odd; name" WHERE 1=0` {
				t.Fatalf("unsafe probe: %s", q)
			}
		}
	}
	if probes != 1 {
		t.Fatalf("probe count=%d queries=%v", probes, fallbackLog.queries)
	}
}

func TestColumnFallbackCancellationAndFailureRetainPrimary(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = probeTableColumns(ctx, db, &sink.Session{Schema: "main"}, "missing", catalogDenied)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, catalogDenied) {
		t.Fatal(err)
	}
	_, err = probeTableColumns(context.Background(), db, &sink.Session{Schema: "main"}, "missing", catalogDenied)
	var recovery *ColumnDiscoveryError
	var native *sqlite.Error
	if !errors.As(err, &recovery) || recovery.Primary != catalogDenied || !errors.As(err, &native) {
		t.Fatalf("lost metadata or native probe cause: %v", err)
	}
	if !errors.Is(err, catalogDenied) || !strings.Contains(err.Error(), "zero-row probe") {
		t.Fatal(err)
	}
}

func TestMissingTableIsNotSuccessfulEmptySchema(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "missing.db")
	_, err := (SQLCatalog{}).Table(context.Background(), &sdk.Connector{Name: "fixture", Driver: "sqlite", DSNTemplate: dsn}, sdk.TableDetailInput{Table: "missing_table"})
	var failure *ColumnDiscoveryError
	if !errors.As(err, &failure) {
		t.Fatalf("expected metadata and probe failure, got %v", err)
	}
	if !strings.Contains(failure.Primary.Error(), "no columns") || failure.Probe == nil {
		t.Fatalf("lost discovery evidence: %+v", failure)
	}
}
