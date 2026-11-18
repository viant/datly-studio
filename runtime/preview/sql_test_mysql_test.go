package preview_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/datly/typecatalog"
	sqlxtypes "github.com/viant/sqlx/types"
	"github.com/viant/x"
)

type sqlFixtureMarker struct{ Label string }

func TestSQLNativeMySQLBitProjection(t *testing.T) {
	dsn := os.Getenv("STUDIO_METADATA_MYSQL_DSN")
	if dsn == "" {
		t.Skip("isolated MySQL fixture not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	name := fmt.Sprintf("studio_sql_bit_%d", time.Now().UnixNano())
	if _, err = db.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP DATABASE `" + name + "`")
	if _, err = db.ExecContext(ctx, "CREATE TABLE `"+name+"`.records(id BIGINT UNSIGNED PRIMARY KEY, flag BIT(1) NOT NULL, optional_flag BIT(1) NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "INSERT INTO `"+name+"`.records VALUES (1,b'0',NULL),(2,b'1',b'0'),(3,b'0',b'1')"); err != nil {
		t.Fatal(err)
	}
	authoring, err := host.NewAuthoring(host.AuthoringConfig{ModulePath: "example.com/fixture"})
	if err != nil {
		t.Fatal(err)
	}
	types, err := authoring.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}

	marker := reflect.TypeFor[sqlFixtureMarker]()
	if err = types.Register(typecatalog.TypeOriginPackage, x.NewType(marker)); err != nil {
		t.Fatal(err)
	}
	snapshotTypes, err := authoring.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	markerKey := marker.PkgPath() + "." + marker.Name()
	if _, found, err := snapshotTypes.ResolveRuntimeType(typecatalog.PackageAuthority, markerKey); err != nil || found {
		t.Fatalf("request catalog changed snapshot: found=%v err=%v", found, err)
	}
	dynamic := preview.Dynamic{ModulePath: authoring.ModulePath(), Types: types}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.DBName = name
	connector := &sdk.Connector{Name: "fixture", Driver: "mysql", DSNTemplate: config.FormatDSN()}
	for _, projection := range []string{"flag AS enabled", "optional_flag AS enabled"} {
		t.Run(projection, func(t *testing.T) {
			result, err := dynamic.TestSQL(ctx, connector, sdk.SQLTestInput{SQL: "SELECT id, " + projection + " FROM `" + name + "`.records ORDER BY id", Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			fixtureDB, err := sql.Open("mysql", config.FormatDSN())
			if err != nil {
				t.Fatal(err)
			}
			defer fixtureDB.Close()
			query := "SELECT id, " + projection + " FROM `" + name + "`.records ORDER BY id"
			contract, err := transcribe.NewCompiler().RuntimeContractsInModule(ctx, authoring.ModulePath(), &transcribe.Source{
				Types: types, Scope: "example.com/fixture/inspection", Name: "reader", Connector: "fixture",
				ColumnRefiner: column.New(column.Connections{"fixture": fixtureDB}),
				Text:          "#package('example.com/fixture/inspection')\n#setting($_ = $connector('fixture'))\n#setting($_ = $route('/fixture','GET'))\n#define($_ = $Rows<[]*Record>(output/view))\nSELECT r.*, type(r,'Record') FROM (" + query + ") r",
			})
			if err != nil {
				t.Fatal(err)
			}
			rowsField, ok := contract.OutputType.FieldByName("Rows")
			if !ok {
				t.Fatal("missing native Rows output")
			}
			field, ok := rowsField.Type.Elem().Elem().FieldByName("Enabled")
			if !ok {
				t.Fatal("missing native Enabled field")
			}
			expectedType := reflect.TypeFor[sqlxtypes.BitBool]()
			if projection == "optional_flag AS enabled" {
				expectedType = reflect.PointerTo(expectedType)
			}
			if field.Type != expectedType {
				t.Fatalf("native field type=%v want %v", field.Type, expectedType)
			}
			empty, err := dynamic.TestSQL(ctx, connector, sdk.SQLTestInput{SQL: "SELECT " + projection + " FROM `" + name + "`.records WHERE 1=0", Limit: 10})
			if err != nil || empty == nil || empty.ReturnedRows != 0 {
				t.Fatalf("empty projection=%+v err=%v", empty, err)
			}
			var envelope struct {
				Rows []map[string]any `json:"rows"`
			}
			if err = json.Unmarshal(result.Data, &envelope); err != nil {
				t.Fatal(err)
			}
			if len(envelope.Rows) != 3 {
				t.Fatalf("result shape=%s", result.Data)
			}
			expected := []any{false, true, false}
			if projection == "optional_flag AS enabled" {
				expected = []any{nil, false, true}
			}
			for i, row := range envelope.Rows {
				if got := row["enabled"]; got != expected[i] {
					t.Fatalf("row %d enabled=%#v, want %#v; result=%s", i, got, expected[i], result.Data)
				}
			}
		})
	}

	if got, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, markerKey); err != nil || !found || got != marker {
		t.Fatalf("execution lost caller type: %v %v %v", got, found, err)
	}
	for _, key := range []string{"example.com/fixture/inspection.Record", "example.com/fixture/dynamic/sqltest/fixture.SQLTestRow"} {
		if _, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, key); err != nil || found {
			t.Fatalf("generated output %s mutated caller catalog: found=%v err=%v", key, found, err)
		}
	}
	resolved, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, "github.com/viant/sqlx/types.BitBool")
	if err != nil || !found || resolved != reflect.TypeFor[sqlxtypes.BitBool]() {
		t.Fatalf("native scanner type=%v found=%v err=%v", resolved, found, err)
	}
}
