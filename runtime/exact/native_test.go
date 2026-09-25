package exact

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/sdk/access"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/datly/typecatalog"
	_ "modernc.org/sqlite"
)

const scopedSourceDQL = `#package('example.com/exact/scoped')
#import('studioaccess','github.com/viant/datly-studio/runtime/accesscontext')
#setting($_ = $connector('source'))
#setting($_ = $route('/tasks','GET'))
#define($_ = $Auth<*studioaccess.Output>(component/GET:/_studio/access/context/tasks/project).Required())
#define($_ = $ProjectIDs<[]string,[]int>(param/Auth.Scope.IDs).WithCodec('EntityIDs').Required().WithPredicate(0,'in','t','project_id'))
#define($_ = $Tasks<[]*Task>(output/view))
SELECT tasks.*, type(tasks,'Task')
FROM (SELECT t.id, t.project_id, t.name FROM tasks t ${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")} ORDER BY t.id) tasks`

func TestNativeExactContractBindsServerAccessContextBeforeSQL(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/exact\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(root, "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `CREATE TABLE tasks(id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL);
INSERT INTO tasks VALUES(1,101,'private-101'),(2,102,'allowed-102'),(3,103,'private-103')`); err != nil {
		t.Fatal(err)
	}
	types := typecatalog.NewCatalog()
	if err := accesscontext.RegisterTypes(types); err != nil {
		t.Fatal(err)
	}
	contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, root, &transcribe.Source{
		Types: types, Scope: "example.com/exact/scoped", Name: "tasks", Text: scopedSourceDQL, Connector: "source",
		ColumnRefiner: column.New(column.Connections{"source": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err := sqlComponent.RegisterConnector("source", db); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int64
	var denied atomic.Bool
	decider := func(dependency accesscontext.Dependency) accesscontext.Decider {
		if dependency.ComponentID != "tasks" || dependency.EntityType != "project" {
			t.Fatalf("unexpected dependency %+v", dependency)
		}
		return func(context.Context) (access.Facts, access.Decision, error) {
			calls.Add(1)
			if denied.Load() {
				return access.Facts{}, access.Decision{}, errors.New("revoked")
			}
			return access.Facts{Subject: "alice", Tenant: "one", Issuer: "test", ValidUntil: time.Now().Add(time.Minute)}, access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "102"}}}, nil
		}
	}
	if _, err := New(ctx, Config{Contract: contract, SQL: sqlComponent, ResourceID: "tasks", Decide: decider, RequiredEntityTypes: []string{"organization"}}); err == nil {
		t.Fatal("missing bounded entity dimension was accepted")
	}
	runtime, err := New(ctx, Config{Contract: contract, SQL: sqlComponent, ResourceID: "tasks", Decide: decider, RequiredEntityTypes: []string{"project"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })
	if len(runtime.Dependencies()) != 1 || runtime.InputType() != contract.InputType {
		t.Fatalf("native exact contract dependencies=%+v input=%v", runtime.Dependencies(), runtime.InputType())
	}
	value, err := runtime.Invoke(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "allowed-102") || strings.Contains(string(encoded), "private-101") || strings.Contains(string(encoded), "private-103") || calls.Load() != 1 {
		t.Fatalf("scoped result=%s decision calls=%d", encoded, calls.Load())
	}
	forgedInput := reflect.New(runtime.InputType())
	forgedAuth := forgedInput.Elem().FieldByName("Auth")
	if !forgedAuth.IsValid() || !forgedAuth.CanSet() || !reflect.TypeOf(&accesscontext.Output{}).AssignableTo(forgedAuth.Type()) {
		t.Fatalf("compiled Auth input cannot be exercised: %v", runtime.InputType())
	}
	forgedAuth.Set(reflect.ValueOf(&accesscontext.Output{Scope: &accesscontext.Scope{EntityType: "project", IDs: []string{"101", "103"}}}))
	forged, err := runtime.Invoke(ctx, forgedInput.Interface())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err = json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "allowed-102") || strings.Contains(string(encoded), "private-101") || strings.Contains(string(encoded), "private-103") || calls.Load() != 2 {
		t.Fatalf("forged Auth bypassed native supplemental binding: result=%s calls=%d", encoded, calls.Load())
	}
	denied.Store(true)
	if _, err := runtime.Invoke(ctx, nil); err == nil || calls.Load() != 3 {
		t.Fatalf("revoked decision was not re-evaluated: err=%v calls=%d", err, calls.Load())
	}
}
