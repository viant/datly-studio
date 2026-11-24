package sqltransport

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

func TestWarmupScheduleTimestampValidation(t *testing.T) {
	f := newImportFixture(t)
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	invalid := []struct {
		name    string
		value   any
		omitted bool
	}{
		{name: "malformed", value: "not-a-date"}, {name: "time only", value: "23:14"},
		{name: "missing", omitted: true}, {name: "empty", value: ""}, {name: "null", value: nil},
		{name: "typed zero", value: time.Time{}}, {name: "typed year zero", value: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "offset into year zero", value: "0001-01-01T00:30:00+01:00"},
		{name: "rounds to zero", value: "0001-01-01T00:00:00.000000001Z"},
	}
	reject := func(t *testing.T, revision int64) {
		for _, tc := range invalid {
			t.Run(tc.name, func(t *testing.T) {
				var err error
				if stamp, ok := tc.value.(time.Time); ok {
					_, err = f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, version.VersionNo, sdk.SaveWarmupScheduleInput{ExpectedRevision: revision, IntervalSeconds: 60, FirstRunAt: stamp, Enabled: true})
				} else {
					input := map[string]any{"expectedRevision": revision, "intervalSeconds": 60, "enabled": true}
					if !tc.omitted {
						input["firstRunAt"] = tc.value
					}
					var out sdk.WarmupScheduleResult
					err = f.transport.Invoke(f.ctx, sdk.OperationWarmupScheduleSave, map[string]any{"reportId": f.report.ID, "versionNo": version.VersionNo, "input": input}, &out)
				}
				var se *sdk.Error
				if !errors.As(err, &se) || se.Code != sdk.ErrorInvalidArgument || se.Field != "firstRunAt" || !strings.Contains(se.Message, "firstRunAt") {
					t.Errorf("expected timestamp field error, got %v", err)
				}
				got, err := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, version.VersionNo)
				if err != nil {
					t.Fatal(err)
				}
				if revision == 0 {
					if got.Schedule != nil {
						t.Fatalf("rejected create persisted %+v", got.Schedule)
					}
					return
				}
				want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
				if got.Schedule == nil || got.Schedule.Revision != revision || !got.Schedule.FirstRunAt.Equal(want) || !got.Schedule.NextDueAt.Equal(want) || got.Schedule.IntervalSeconds != 3600 || !got.Schedule.Enabled {
					t.Fatalf("rejected update changed schedule %+v", got.Schedule)
				}
			})
		}
	}
	if !t.Run("create", func(t *testing.T) { reject(t, 0) }) {
		return
	}
	for i, stamp := range []string{"2027-01-02T03:04:05.123456Z", "2027-01-02T08:34:05.123456+05:30", "2020-01-02T03:04:05Z"} {
		first, err := time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			t.Fatal(err)
		}
		result, err := f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, version.VersionNo, sdk.SaveWarmupScheduleInput{ExpectedRevision: int64(i), IntervalSeconds: 3600, FirstRunAt: first, Enabled: true})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, version.VersionNo)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []*sdk.WarmupSchedule{result.Schedule, got.Schedule} {
			if s == nil || s.Revision != int64(i+1) || !s.FirstRunAt.Equal(first) || !s.NextDueAt.Equal(first) {
				t.Fatalf("timestamp changed %+v", s)
			}
		}
	}
	t.Run("update", func(t *testing.T) { reject(t, 3) })
}
