package sqltransport

import (
	"errors"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
)

// Deletions are fixture setup here; allocation is exercised through the real
// SDK entry points and transcribed component-counter/version mutation graphs.
func TestVersionAllocationNeverReusesDeletedNumbers(t *testing.T) {
	f := newImportFixture(t)
	f.db.SetMaxOpenConns(1)
	create := func(want int) {
		t.Helper()
		v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "sql", AuthoredSQL: "SELECT 1"})
		if err != nil || v.VersionNo != want {
			t.Fatalf("create=%+v err=%v want=%d", v, err, want)
		}
	}
	create(1)
	create(2)
	if _, err := f.db.ExecContext(f.ctx, "DELETE FROM component_versions WHERE report_id=? AND version_no=2", f.report.ID); err != nil {
		t.Fatal(err)
	}
	create(3)
	if _, err := f.db.ExecContext(f.ctx, "DELETE FROM component_versions WHERE report_id=?", f.report.ID); err != nil {
		t.Fatal(err)
	}
	imported, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 4"})
	if err != nil || imported.Version.VersionNo != 4 {
		t.Fatalf("import=%+v err=%v", imported, err)
	}
	create(5)
	var high int
	if err = f.db.QueryRowContext(f.ctx, "SELECT last_version_no FROM components WHERE id=?", f.report.ID).Scan(&high); err != nil || high != 5 {
		t.Fatalf("high-water=%d err=%v", high, err)
	}
}

func TestVersionAllocationGraphRollsBackCounterAndChildren(t *testing.T) {
	f := newImportFixture(t)
	if _, err := f.db.ExecContext(f.ctx, `CREATE TRIGGER allocation_reject_file BEFORE INSERT ON component_resource_files BEGIN SELECT RAISE(ABORT,'fixture rejects file'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := f.client.Versions().LoadDQL(f.ctx, f.report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Cause == nil || !strings.Contains(sdkErr.Cause.Error(), "fixture rejects file") {
		t.Fatalf("failed import=%v", err)
	}
	var high, versions, files int
	if err = f.db.QueryRowContext(f.ctx, `SELECT last_version_no,(SELECT COUNT(*) FROM component_versions WHERE report_id=components.id),(SELECT COUNT(*) FROM component_resource_files WHERE report_id=components.id) FROM components WHERE id=?`, f.report.ID).Scan(&high, &versions, &files); err != nil || high != 0 || versions != 0 || files != 0 {
		t.Fatalf("rolled back graph: high=%d versions=%d files=%d err=%v", high, versions, files, err)
	}
	if _, err = f.db.ExecContext(f.ctx, "DROP TRIGGER allocation_reject_file"); err != nil {
		t.Fatal(err)
	}
	v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "structured"})
	if err != nil || v.VersionNo != 1 {
		t.Fatalf("create after rollback=%+v err=%v", v, err)
	}
}

func TestVersionHeadNativeLockRequiresTransaction(t *testing.T) {
	f := newImportFixture(t)
	if _, err := f.transport.nextVersionNo(f.ctx, nil, f.report.ID, namespaceaccess.ID(f.report.OwnerID, f.report.Namespace)); err == nil || !strings.Contains(err.Error(), "active transaction") {
		t.Fatalf("unmanaged lock=%v", err)
	}
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	next, err := f.transport.nextVersionNo(f.ctx, tx, f.report.ID, namespaceaccess.ID(f.report.OwnerID, f.report.Namespace))
	if err != nil || next != 1 {
		t.Fatalf("locked high-water next=%d err=%v", next, err)
	}
}
