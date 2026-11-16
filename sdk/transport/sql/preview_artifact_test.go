package sqltransport

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/sdk/executionguard"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/compiledartifact"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/internal/resourceartifact"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/constant"
	"github.com/viant/sqlx/testutil/sqlfault"
)

var previewObservedDriverID atomic.Int64

// DriverContext retains the real driver's identity in SQLX's native observer.
type previewObservedDriver struct {
	driver.Driver
	before func(context.Context, sqlfault.Call) error
}

func (d *previewObservedDriver) OpenConnector(dsn string) (driver.Connector, error) {
	return &sqlfault.Connector{Base: d.Driver, DSN: dsn, Before: d.before}, nil
}

func TestPreviewArtifactSDKReuseAndRefresh(t *testing.T) {
	f := newImportFixture(t)
	ctx := f.ctx
	dsn := "file:" + filepath.Join(t.TempDir(), "source.db")
	business, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer business.Close()
	if _, err = business.ExecContext(ctx, `CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT);INSERT INTO records VALUES(1,'initial')`); err != nil {
		t.Fatal(err)
	}
	var discovery, opens atomic.Int64
	driverName := fmt.Sprintf("studio_preview_artifact_%d", previewObservedDriverID.Add(1))
	sql.Register(driverName, &previewObservedDriver{Driver: business.Driver(), before: func(ctx context.Context, call sqlfault.Call) error {
		if call.Phase == "query" && strings.Contains(strings.ToLower(call.SQL), "pragma_table_info") {
			discovery.Add(1)
		}
		return nil
	}})
	if _, err = f.db.ExecContext(ctx, `UPDATE connectors SET driver=?,dsn_template=?,secret_ref='owned-secret';UPDATE components SET component_scope='example.com/artifact/records',component_name='reader'`, driverName, dsn); err != nil {
		t.Fatal(err)
	}
	source := `#package('example.com/artifact/records')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#define($_ = $Tenant<int>(const/Tenant).Value('7'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT records.*,type(records,'Record') FROM (${embed:sql/records.sql}) records`
	version, err := f.client.Versions().Create(ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.db.ExecContext(ctx, `INSERT INTO component_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES(?,?,'sql','proof.sql','sql/records.sql',?,?,'hash',0,?)`, f.report.ID, version.VersionNo, []byte("SELECT * FROM records"), len("SELECT * FROM records"), f.now); err != nil {
		t.Fatal(err)
	}
	destination := dsn
	dynamic := preview.Dynamic{StudioDB: f.db, ModulePath: "example.com/artifact", Secrets: connectorsecret.ResolverFunc(func(ctx context.Context, template, reference string) (string, error) {
		opens.Add(1)
		return destination, nil
	})}
	f.transport.Preview = &dynamic
	run := func(refresh bool) *sdk.PreviewResult {
		t.Helper()
		result, err := f.client.Preview().Execute(ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{RefreshSchema: refresh})
		if err != nil {
			t.Fatal(err)
		}
		if result.Evidence.SourceRevision != version.SourceRevision {
			t.Fatal("source revision changed")
		}
		return result
	}
	artifacts := func() int {
		t.Helper()
		var count int
		if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=? AND namespace=?`, f.report.ID, resourceartifact.Namespace).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	rawArtifact := func() []byte {
		t.Helper()
		var data []byte
		if err := f.db.QueryRowContext(ctx, `SELECT content FROM component_resource_files WHERE report_id=? AND namespace=? ORDER BY resource_id LIMIT 1`, f.report.ID, resourceartifact.Namespace).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	// Input inspection uses the same frozen contract generation as execution.
	var inspected sdk.ReaderInspection
	err = f.transport.Invoke(ctx, sdk.OperationVersionInspect, map[string]any{"reportId": f.report.ID, "versionNo": version.VersionNo, "discoverColumns": false, "includePreviewInput": true, "includeEffectiveWarmup": false}, &inspected)
	if err != nil || inspected.PreviewInput == nil {
		t.Fatalf("input inspection: %+v %v", inspected, err)
	}
	if opens.Load() != 1 {
		t.Fatalf("inspection resolved the connector %d times", opens.Load())
	}
	if discovery.Load() == 0 || artifacts() != 1 {
		t.Fatalf("inspection did not compile/persist: discovery=%d artifacts=%d", discovery.Load(), artifacts())
	}
	discovery.Store(0)
	first := run(false)
	if discovery.Load() != 0 || !strings.Contains(string(first.Data), "initial") {
		t.Fatalf("preview did not reuse inspection artifact: %s calls=%d", first.Data, discovery.Load())
	}
	if _, err = business.ExecContext(ctx, `UPDATE records SET name='current';ALTER TABLE records ADD COLUMN extra TEXT;UPDATE records SET extra='schema-refresh'`); err != nil {
		t.Fatal(err)
	}
	repeated := run(false)
	if discovery.Load() != 0 || !strings.Contains(string(repeated.Data), "current") || strings.Contains(string(repeated.Data), "schema-refresh") {
		t.Fatalf("ordinary reuse shape/data: %s calls=%d", repeated.Data, discovery.Load())
	}
	refreshed := run(true)
	if discovery.Load() == 0 || !strings.Contains(string(refreshed.Data), "schema-refresh") || artifacts() != 1 {
		t.Fatalf("schema refresh: %s calls=%d", refreshed.Data, discovery.Load())
	}
	// An identical refresh must not issue UPDATE, even with a caller declining writes.
	prior := rawArtifact()
	if _, err = f.db.ExecContext(ctx, `CREATE TRIGGER reject_artifact_update BEFORE UPDATE ON component_resource_files WHEN OLD.namespace='__datly_compiled' BEGIN SELECT RAISE(ABORT,'unexpected artifact update'); END`); err != nil {
		t.Fatal(err)
	}
	run(true)
	if _, err = f.db.ExecContext(ctx, `DROP TRIGGER reject_artifact_update`); err != nil {
		t.Fatal(err)
	}
	if string(rawArtifact()) != string(prior) {
		t.Fatal("identical refresh changed stored bytes")
	}
	// Discovery failure returns an error and preserves the prior generation.
	if _, err = business.ExecContext(ctx, `ALTER TABLE records RENAME TO hidden_records`); err != nil {
		t.Fatal(err)
	}
	_, failure := f.client.Preview().Execute(ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{RefreshSchema: true})
	if _, err = business.ExecContext(ctx, `ALTER TABLE hidden_records RENAME TO records`); err != nil {
		t.Fatal(err)
	}
	if failure == nil || string(rawArtifact()) != string(prior) {
		t.Fatal("refresh failure fell back or changed artifact")
	}
	discovery.Store(0)
	run(false)
	if discovery.Load() != 0 {
		t.Fatal("failed refresh discarded prior generation")
	}

	// Pause A after live discovery and freezing, then publish a changed schema B.
	// A must execute its own fresh contract without overwriting the winner.
	if _, err = f.db.ExecContext(ctx, `CREATE TABLE proof_updates(n INTEGER); INSERT INTO proof_updates VALUES(0); CREATE TRIGGER count_artifact_updates AFTER UPDATE ON component_resource_files WHEN OLD.namespace='__datly_compiled' BEGIN UPDATE proof_updates SET n=n+1; END`); err != nil {
		t.Fatal(err)
	}
	discovery.Store(0)
	ready, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var paused atomic.Bool
	defer once.Do(func() { close(release) })
	raceCtx, cancelRace := context.WithTimeout(ctx, 5*time.Second)
	defer cancelRace()
	guarded := executionguard.With(raceCtx, func(ctx context.Context) error {
		if discovery.Load() > 0 && paused.CompareAndSwap(false, true) {
			close(ready)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	type outcome struct {
		result *sdk.PreviewResult
		err    error
	}
	finished := make(chan outcome, 1)
	go func() {
		result, err := f.client.Preview().Execute(guarded, f.report.ID, version.VersionNo, sdk.PreviewInput{RefreshSchema: true})
		finished <- outcome{result, err}
	}()
	select {
	case <-ready:
	case <-raceCtx.Done():
		t.Fatal("A did not finish discovery")
	}
	if _, err = business.ExecContext(ctx, `ALTER TABLE records ADD COLUMN concurrent TEXT;UPDATE records SET concurrent='concurrent-schema-B'`); err != nil {
		t.Fatal(err)
	}
	winner := run(true)
	persistedB := rawArtifact()
	if !strings.Contains(string(winner.Data), "concurrent-schema-B") {
		t.Fatal("second refresh did not execute B")
	}
	once.Do(func() { close(release) })
	var firstRace outcome
	select {
	case firstRace = <-finished:
	case <-raceCtx.Done():
		t.Fatal(raceCtx.Err())
	}
	if firstRace.err != nil || firstRace.result == nil || strings.Contains(string(firstRace.result.Data), "concurrent-schema-B") || !strings.Contains(string(firstRace.result.Data), "schema-refresh") {
		t.Fatalf("first refresh did not execute freshly compiled A: %+v %v", firstRace.result, firstRace.err)
	}
	var updates int
	if err = f.db.QueryRowContext(ctx, `SELECT n FROM proof_updates`).Scan(&updates); err != nil || updates != 1 || string(rawArtifact()) != string(persistedB) {
		t.Fatalf("identical refresh overwrote B: updates=%d err=%v", updates, err)
	}
	if _, err = f.db.ExecContext(ctx, `DROP TRIGGER count_artifact_updates`); err != nil {
		t.Fatal(err)
	}
	// Protected input enforcement still runs after a hit.
	_, err = f.client.Preview().Execute(ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{Input: json.RawMessage(`{"Tenant":9}`)})
	if err == nil {
		t.Fatal("cached contract accepted protected input")
	}
	// Exact-version admission remains ahead of capture, even on a hit.
	previousAuthorizer := f.transport.Authorizer
	f.transport.Authorizer = AuthorizerFunc(func(context.Context, AuthorizationRequest) error {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "owned denial"}
	})
	calls := opens.Load()
	_, err = f.client.Preview().Execute(ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{})
	f.transport.Authorizer = previousAuthorizer
	if err == nil || opens.Load() != calls {
		t.Fatal("cached artifact bypassed authorization")
	}
	// Constants and resource contents select new keys without changing source.
	dynamic.Const, _ = constant.New(map[string]string{"Tenant": "8"})
	discovery.Store(0)
	run(false)
	if discovery.Load() == 0 || artifacts() != 2 {
		t.Fatal("constant change reused previous artifact")
	}
	if _, err = f.db.ExecContext(ctx, `UPDATE component_resource_files SET content=? WHERE report_id=? AND resource_id='sql'`, []byte("SELECT id,name FROM records"), f.report.ID); err != nil {
		t.Fatal(err)
	}
	discovery.Store(0)
	run(false)
	if discovery.Load() == 0 || artifacts() != 3 {
		t.Fatal("resource change reused previous artifact")
	}
	// The same logical secret can move to a different physical database.
	destination = "file:" + filepath.Join(t.TempDir(), "moved.db")
	moved, err := sql.Open("sqlite", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Close()
	if _, err = moved.ExecContext(ctx, `CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT);INSERT INTO records VALUES(2,'moved-destination')`); err != nil {
		t.Fatal(err)
	}
	discovery.Store(0)
	result := run(false)
	if discovery.Load() == 0 || artifacts() != 4 || !strings.Contains(string(result.Data), "moved-destination") {
		t.Fatalf("resolved destination reused old artifact: %s", result.Data)
	}

	// Editing this same numbered version changes source identity; creating another
	// numbered version remains independently targeted even with the same route.
	changedSource := strings.Replace(source, "FROM (${embed:sql/records.sql}) records", "FROM (SELECT records.*,42 AS source_marker FROM (${embed:sql/records.sql}) records) records", 1)
	payload, _ := json.Marshal(map[string]string{"authoredDql": changedSource})
	edited, err := f.client.Versions().Apply(ctx, f.report.ID, version.VersionNo, sdk.EditCommand{Kind: "set_dql", ExpectedSourceRevision: version.SourceRevision, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	version = edited.Version
	discovery.Store(0)
	changed := run(false)
	if discovery.Load() == 0 || artifacts() != 5 || !strings.Contains(string(changed.Data), "42") {
		t.Fatalf("source edit retained stale contract: %s", changed.Data)
	}
	newVersionSource := strings.ReplaceAll(strings.ReplaceAll(changedSource, "${embed:sql/records.sql}", "SELECT id,name FROM records"), "42 AS source_marker", "43 AS source_marker")
	next, err := f.client.Versions().Create(ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: newVersionSource})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := f.client.Preview().Execute(ctx, f.report.ID, next.VersionNo, sdk.PreviewInput{})
	if err != nil || selected.Evidence.VersionNo != next.VersionNo || !strings.Contains(string(selected.Data), "43") {
		t.Fatalf("wrong numbered version: %+v %v", selected, err)
	}
	again := run(false)
	if strings.Contains(string(again.Data), "43") || !strings.Contains(string(again.Data), "42") {
		t.Fatal("version switch replaced the first version's artifact")
	}

	// A run-only grantee need not be able to browse the primary connector. Its
	// secret-only definition still resolves once at the actual connection owner.
	if _, err = f.db.ExecContext(ctx, `UPDATE connectors SET dsn_template=NULL; INSERT INTO component_acl(report_id,subject_type,subject_id,can_run,can_view) VALUES(?,'user','runner',1,0)`, f.report.ID); err != nil {
		t.Fatal(err)
	}
	callsBefore := opens.Load()
	runner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "runner"})
	delegated, err := f.client.Preview().Execute(runner, f.report.ID, version.VersionNo, sdk.PreviewInput{})
	if err != nil || delegated == nil || opens.Load() != callsBefore+1 || !strings.Contains(string(delegated.Data), "42") {
		t.Fatalf("secret-only run grant failed: %+v %v", delegated, err)
	}
	// Verified checksum/format failures remain errors, rather than cache misses.
	if _, err = f.db.ExecContext(ctx, `UPDATE component_resource_files SET content='broken' WHERE report_id=? AND namespace=?`, f.report.ID, resourceartifact.Namespace); err != nil {
		t.Fatal(err)
	}
	_, err = f.client.Preview().Execute(ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{})
	var public *sdk.Error
	if !errors.As(err, &public) || public.Code != sdk.ErrorInternal {
		t.Fatalf("corrupt artifact was silently recompiled: %v", err)
	}
}

func TestPreviewOversizeArtifactRetainsLiveExecution(t *testing.T) {
	f := newImportFixture(t)
	const scope = "example.com/oversize/records"
	source := fmt.Sprintf(`#package('%s')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $mcp('owned.records.read','%s'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT records.*,type(records,'Record') FROM (SELECT 1 AS id) records`, scope, strings.Repeat("x", (1<<20)+1))
	if _, err := f.db.ExecContext(f.ctx, `UPDATE connectors SET dsn_template=':memory:';UPDATE components SET component_scope=?,component_name='reader' WHERE id=?`, scope, f.report.ID); err != nil {
		t.Fatal(err)
	}
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	f.transport.Preview = &preview.Dynamic{StudioDB: f.db, ModulePath: "example.com/oversize"}
	result, err := f.client.Preview().Execute(f.ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{})
	if err != nil || result == nil || !strings.Contains(string(result.Data), "1") {
		t.Fatalf("oversize artifact prevented ordinary preview: %+v %v", result, err)
	}
	var count int
	if err = f.db.QueryRowContext(f.ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id=? AND namespace=?`, f.report.ID, resourceartifact.Namespace).Scan(&count); err != nil || count != 0 {
		t.Fatalf("oversize artifact persisted: count=%d err=%v", count, err)
	}
	_, err = f.client.Preview().Execute(f.ctx, f.report.ID, version.VersionNo, sdk.PreviewInput{RefreshSchema: true})
	if !errors.Is(err, compiledartifact.ErrTooLarge) {
		t.Fatalf("explicit refresh suppressed persistence failure: %v", err)
	}
}
