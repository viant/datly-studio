package warmupschedule

import (
	"testing"
	"time"
)

func TestNextDueCoalescesWithoutDrift(t *testing.T) {
	due := time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ now, want time.Time }{{due.Add(-time.Second), due}, {due, due.Add(time.Hour)}, {due.Add(3*time.Hour + 17*time.Minute), due.Add(4 * time.Hour)}} {
		got, e := NextDue(due, tc.now, 3600)
		if e != nil || !got.Equal(tc.want) {
			t.Fatalf("next=%v want=%v err=%v", got, tc.want, e)
		}
	}
}
