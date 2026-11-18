package readerinspection

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/transcribe"
)

func TestContributorProjectionUsesExistingSourcePermission(t *testing.T) {
	const source = `#setting($_ = $route('/records','GET'))
#define($_ = $Period<string>(query/period).Optional())
SELECT r.*, c.*, cache_warmup(r,'id','Period=PRIVATE_CONTRIBUTOR_CANARY') AS warmup,
cache_warmup(c,'id','Limit=0')
FROM records r LEFT JOIN children c ON c.parent_id=r.id`
	response := readerbuilder.New(readerbuilder.Config{Name: "Records"}).Apply(t.Context(), readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if len(response.Diagnostics) != 0 {
		t.Fatalf("native inspection: %+v", response.Diagnostics)
	}
	if len(response.Structure.TargetedWarmups) != 2 || response.Structure.TargetedWarmups[0].Editable || response.Structure.TargetedWarmups[0].Diagnostic == "" || response.Structure.TargetedWarmups[0].Contributors == nil || len(response.Structure.TargetedWarmups[0].Contributors) != 0 || !response.Structure.TargetedWarmups[1].Editable {
		t.Fatalf("invalid native fixture: %+v", response.Structure.TargetedWarmups)
	}
	response.Diagnostics = []*transcribe.Diagnostic{{Code: "FIXTURE", Message: "PRIVATE_DIAGNOSTIC_CANARY", Hint: "PRIVATE_HINT_CANARY"}}
	component, err := json.Marshal(response.Structure.Component)
	if err != nil {
		t.Fatal(err)
	}
	version := &sdk.ReportVersion{ReportID: "records", VersionNo: 2, SourceRevision: 7, AuthoredDQL: source, GeneratedDQL: source, ComponentSpec: component, CompileDiagnostics: json.RawMessage(`[{"message":"PRIVATE_COMPILE_CANARY"}]`)}
	for _, allowed := range []bool{false, true} {
		result := Project(version, &response, sdk.ReportCapabilities{CanUseDQL: allowed, CanRun: true, CanPublish: true})
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if result.Version.ReportID != "records" || result.Version.VersionNo != 2 || result.Version.SourceRevision != 7 {
			t.Fatal("snapshot identity changed")
		}
		for _, value := range []string{"targetedWarmups", "contributors", "PRIVATE_CONTRIBUTOR_CANARY", "PRIVATE_DIAGNOSTIC_CANARY", "PRIVATE_HINT_CANARY", "PRIVATE_COMPILE_CANARY", response.Structure.TargetedWarmups[0].Diagnostic} {
			if strings.Contains(string(raw), value) != allowed {
				t.Fatalf("source projection allowed=%v value=%q output=%s", allowed, value, raw)
			}
		}
		if allowed {
			var projected struct {
				TargetedWarmups []readerbuilder.TargetedWarmupPlan `json:"targetedWarmups"`
			}
			if err = json.Unmarshal(result.Structure, &projected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(projected.TargetedWarmups, response.Structure.TargetedWarmups) {
				t.Fatal("native metadata was altered")
			}
		} else if result.Version.AuthoredDQL != "" || result.Version.GeneratedDQL != "" || len(result.Version.ComponentSpec) != 0 || len(result.Version.CompileDiagnostics) != 0 {
			t.Fatal("source-bearing version fields leaked")
		}
	}
}
