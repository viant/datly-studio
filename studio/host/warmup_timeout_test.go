package host_test

import (
	"testing"
	"time"

	"github.com/viant/datly-studio/studio/host"
)

func TestWarmupOperationTimeoutConfiguration(t *testing.T) {
	for _, value := range []string{"", "0", "0s", "-1s", "not-a-duration", "999999999999999999999h"} {
		t.Run("invalid/"+value, func(t *testing.T) {
			if snapshot, err := host.NewAuthoring(host.AuthoringConfig{Config: host.Config{WarmupOperationTimeout: &value}}); err == nil {
				snapshot.Close()
				t.Fatal("invalid timeout accepted")
			}
		})
	}
	for _, value := range []*string{nil, durationSetting("10m")} {
		snapshot, err := host.NewAuthoring(host.AuthoringConfig{Config: host.Config{WarmupOperationTimeout: value}})
		if err != nil {
			t.Fatal(err)
		}
		want := 5 * time.Minute
		if value != nil {
			want = 10 * time.Minute
			*value = "1s"
		}
		if snapshot.WarmupTimeout() != want {
			t.Fatalf("timeout=%v want=%v", snapshot.WarmupTimeout(), want)
		}
		snapshot.Close()
	}
}

func durationSetting(value string) *string { return &value }
