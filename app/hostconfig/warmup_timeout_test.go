package hostconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/studio/host"
)

func TestLoadWarmupOperationTimeout(t *testing.T) {
	for _, tc := range []struct {
		name, setting string
		want          time.Duration
	}{
		{"omitted", "", 5 * time.Minute},
		{"configured", "WarmupOperationTimeout: 15m\n", 15 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "datastudio.yaml")
			if err := os.WriteFile(path, []byte(base+tc.setting), 0600); err != nil {
				t.Fatal(err)
			}
			config, err := Load(t.Context(), path)
			if err != nil {
				t.Fatal(err)
			}
			if (config.WarmupOperationTimeout == nil) != (tc.setting == "") {
				t.Fatal("timeout omission was not preserved")
			}
			snapshot, err := host.NewAuthoring(host.AuthoringConfig{Config: host.Config{WarmupOperationTimeout: config.WarmupOperationTimeout}})
			if err != nil {
				t.Fatal(err)
			}
			defer snapshot.Close()
			if got := snapshot.WarmupTimeout(); got != tc.want {
				t.Fatalf("native timeout=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestDecodeRejectsInvalidWarmupOperationTimeout(t *testing.T) {
	for _, value := range []string{"''", "'0'", "0s", "-1m", "invalid", "999999999999999999h"} {
		t.Run(value, func(t *testing.T) {
			_, err := Decode([]byte(base + "WarmupOperationTimeout: " + value + "\n"))
			if err == nil || !strings.Contains(err.Error(), "WarmupOperationTimeout") {
				t.Fatalf("expected native duration validation, got %v", err)
			}
		})
	}
}
