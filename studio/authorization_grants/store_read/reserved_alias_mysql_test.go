package store_read

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// The normal offline suite has no MySQL dependency. Qualification explicitly
// supplies a private config file and exact synthetic owner tuple; this test
// performs SELECTs only and never creates or changes permissions or fixtures.
func TestAuthorizationGrantReaderMySQLReservedAlias(t *testing.T) {
	path := os.Getenv("DATLY_GRANT_MYSQL_CONFIG")
	if path == "" {
		t.Skip("opt-in native MySQL qualification requires a private Studio config")
	}
	report, subject := os.Getenv("DATLY_GRANT_MYSQL_REPORT_ID"), os.Getenv("DATLY_GRANT_MYSQL_SUBJECT")
	if report == "" || subject == "" {
		t.Fatal("an explicit synthetic report/owner tuple is required")
	}
	var config struct{ Store struct{ Driver, DSN string } }
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read private Studio config")
	}
	if json.Unmarshal(raw, &config) != nil || config.Store.Driver != "mysql" || config.Store.DSN == "" {
		t.Fatal("private Studio config must identify MySQL")
	}
	db, err := sql.Open("mysql", config.Store.DSN)
	if err != nil {
		t.Fatal("cannot open qualification connector")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assets := resource.New()
	if err = assets.Register(GrantDatlyResourceNamespace, GrantDatlyResources); err != nil {
		t.Fatal(err)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err = sqlComponent.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	compiled, target, err := readercomponent.Compile(reflect.TypeOf(GrantComponent{}), "store_read", reflect.TypeOf(Input{}), reflect.TypeOf(Output{}), assets, sqlComponent)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{compiled}, druntime.WithResources(assets))
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown(context.Background())
	for _, tc := range []struct {
		name, subject string
		rows          int
	}{{"exact-owner", subject, 1}, {"foreign-subject", "synthetic-non-owner-no-grant", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			in := &Input{SubjectId: tc.subject, SubjectType: "user", IsLive: true, Has: &InputHas{SubjectId: true, SubjectType: true, IsLive: true}}
			in.SetReportId(report)
			in.SetGrantSource("owner")
			value, err := runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Input: in})
			if err != nil {
				t.Fatal(err)
			}
			out, ok := value.(*Output)
			if !ok || out == nil {
				t.Fatalf("unexpected native output %T", value)
			}
			if len(out.Grants) != tc.rows {
				t.Fatalf("got %d grants, expected %d", len(out.Grants), tc.rows)
			}
			for _, g := range out.Grants {
				if g == nil || g.ReportId != report || g.SubjectId != subject || g.SubjectType != "user" || g.GrantSource != "owner" || !g.IsLive || !g.CanView || !g.CanEdit || !g.CanPublish {
					t.Fatal("native owner grant changed identity or capabilities")
				}
			}
		})
	}
}
