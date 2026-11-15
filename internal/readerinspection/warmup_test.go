package readerinspection

import (
	"github.com/viant/datly-studio/sdk"
	"testing"
)

func TestEffectiveWarmupSnapshot(t *testing.T) {
	for _, target := range []struct {
		report   string
		version  int
		revision int64
	}{{"records", 2, 6}, {"other", 2, 7}, {"records", 3, 7}, {"records", 2, 7}} {
		result := &sdk.ReaderInspection{Version: &sdk.ReportVersion{ReportID: "records", VersionNo: 2, SourceRevision: 7}}
		targets := []sdk.EffectiveWarmupTarget{}
		SetEffectiveWarmup(result, &sdk.EffectiveWarmupInspection{Status: "resolved", ReportID: target.report, VersionNo: target.version, SourceRevision: target.revision, Targets: &targets})
		if target.report == "records" && target.version == 2 && target.revision == 7 {
			if result.EffectiveWarmup.Status != "resolved" || len(result.Diagnostics) != 0 {
				t.Fatal("matching snapshot rejected")
			}
		} else {
			if result.EffectiveWarmup.Status != "unavailable" || result.EffectiveWarmup.Targets != nil || len(result.Diagnostics) != 1 {
				t.Fatal("stale snapshot attached")
			}
		}
	}
}
