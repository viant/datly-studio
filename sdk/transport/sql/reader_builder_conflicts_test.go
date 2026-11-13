package sqltransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
)

func TestReaderBuilderWarmupConflicts(t *testing.T) {
	f := newImportFixture(t)
	source := `#package('example.com/records/reader')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $mcp('owner.records','Read records'))
#setting($_ = $cache('records','1m'))
#setting($_ = $cache_warmup('ID','Period=test-hour','Limit=0','MaxCases=1'))
#define($_ = $Period<string>(query/period).Optional())
#define($_ = $Rows<[]*Row>(output/view))
SELECT records.*,type(records,'Row') FROM (SELECT 1 AS ID) records`
	source = strings.Replace(source, "$cache('records','1m')", fmt.Sprintf("$cache('records','1m').WithLocation('%s')", filepath.Join(t.TempDir(), "cache")), 1)
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.client.Components().Create(f.ctx, sdk.CreateComponentInput{Slug: "other", Title: "Other", DefaultConnectorName: "source"})
	if err != nil {
		t.Fatal(err)
	}
	otherVersion, err := f.client.Versions().Create(f.ctx, other.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	operation := json.RawMessage(`{"type":"setSetting","setting":{"name":"cache_warmup","args":["'ID'","'Period=lifetime'","'MaxCases=1'","'Limit=0'"]}}`)
	for _, tc := range []struct {
		name, state, field, message string
		revision                    int64
	}{
		{"matching revision duplicate name", "draft", "mcp.name", "already used by another reader", version.SourceRevision},
		{"stale revision", "draft", "sourceRevision", "version source revision does not match", version.SourceRevision + 1},
		{"published version", "published", "state", "published version cannot be mutated", version.SourceRevision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.db.ExecContext(f.ctx, "UPDATE component_versions SET state=? WHERE report_id=?", tc.state, f.report.ID); err != nil {
				t.Fatal(err)
			}
			result, err := f.client.Versions().ApplyReaderCommand(f.ctx, f.report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: tc.revision, Operation: operation})
			var detail *sdk.Error
			if !errors.As(err, &detail) || detail.Code != sdk.ErrorConflict || detail.Field != tc.field || !strings.Contains(detail.Message, tc.message) {
				t.Fatalf("conflict=%+v err=%v result=%+v", detail, err, result)
			}
			current, err := f.client.Versions().Get(f.ctx, f.report.ID, version.VersionNo)
			if err != nil || current.SourceRevision != version.SourceRevision || current.AuthoredDQL != source {
				t.Fatalf("rejected edit changed source: %+v err=%v", current, err)
			}
		})
	}
	// Fix only the other fixture's tool name; the exact same warmup edit is valid.
	if _, err := f.db.ExecContext(f.ctx, "UPDATE component_versions SET state='draft' WHERE report_id=?", f.report.ID); err != nil {
		t.Fatal(err)
	}
	otherSource := strings.ReplaceAll(source, "owner.records", "owner.other")
	if _, err := f.db.ExecContext(f.ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id=? AND version_no=?", otherSource, otherSource, other.ID, otherVersion.VersionNo); err != nil {
		t.Fatal(err)
	}
	result, err := f.client.Versions().ApplyReaderCommand(f.ctx, f.report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: version.SourceRevision, Operation: operation})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.Inspection == nil || result.Inspection.Version == nil || result.Inspection.Version.SourceRevision != version.SourceRevision+1 || !strings.Contains(result.Inspection.DQL, "lifetime") {
		t.Fatalf("valid warmup edit: %+v", result)
	}
	var runs int
	if err := f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM component_warmup_runs").Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("save executed warmup: runs=%d err=%v", runs, err)
	}
}
