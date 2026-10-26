package main

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	connectorusage "github.com/viant/datly-studio/studio/connectors/store_usage"
	namespaceusage "github.com/viant/datly-studio/studio/namespaces/store_usage"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// The caller supplies an isolated Studio fixture database. Only SELECTs run.
func TestUsageReadersMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("STUDIO_TEST_MYSQL_DSN not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, "SET SESSION sql_mode=CONCAT(@@sql_mode,',ANSI_QUOTES')"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"namespace", "connector"} {
		t.Run(kind, func(t *testing.T) {
			resources := resource.New()
			var holder, inputType, outputType reflect.Type
			var input any
			if kind == "namespace" {
				if err = resources.Register(namespaceusage.UsageDatlyResourceNamespace, namespaceusage.UsageDatlyResources); err != nil {
					t.Fatal(err)
				}
				holder, inputType, outputType = reflect.TypeFor[namespaceusage.UsageComponent](), reflect.TypeFor[namespaceusage.Input](), reflect.TypeFor[namespaceusage.Output]()
				v := &namespaceusage.Input{}
				v.SetOwnerId("usage-regression-missing-owner")
				v.SetName("usage_regression_missing")
				input = v
			} else {
				if err = resources.Register(connectorusage.UsageDatlyResourceNamespace, connectorusage.UsageDatlyResources); err != nil {
					t.Fatal(err)
				}
				holder, inputType, outputType = reflect.TypeFor[connectorusage.UsageComponent](), reflect.TypeFor[connectorusage.Input](), reflect.TypeFor[connectorusage.Output]()
				v := &connectorusage.Input{}
				v.SetName("usage_regression_missing")
				input = v
			}
			component := &dsql.SQLComponent{DB: db}
			if err = component.RegisterConnector("studio", db); err != nil {
				t.Fatal(err)
			}
			registration, target, err := readercomponent.Compile(holder, "usage", inputType, outputType, resources, component)
			if err != nil {
				t.Fatal(err)
			}
			runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Shutdown(context.Background())
			actual, err := runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Input: input})
			if err != nil {
				t.Fatal(err)
			}
			rows := reflect.ValueOf(actual).Elem().FieldByName("Usages")
			if rows.Len() != 1 || rows.Index(0).Elem().FieldByName("Used").Int() != 0 {
				t.Fatalf("unexpected usage count: %+v", actual)
			}
		})
	}
}
