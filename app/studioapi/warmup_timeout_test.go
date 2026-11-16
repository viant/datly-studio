package studioapi

import (
	"strings"
	"testing"

	"github.com/viant/datly-studio/studio/host"
)

func TestWarmupTimeoutRejectedBeforeHostStarts(t *testing.T) {
	for _, value := range []string{"0s", "-1m", "invalid", "999999999999999999h"} {
		err := Serve(t.Context(), nil, Options{HostConfig: host.Config{WarmupOperationTimeout: &value}})
		if err == nil || !strings.Contains(err.Error(), "WarmupOperationTimeout") {
			t.Fatalf("invalid host timeout accepted: %v", err)
		}
	}
}
