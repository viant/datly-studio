package catalog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/viant/xdatly/response"
)

func TestReleaseOperationLogsUnderlyingFailure(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	err := releaseOperationError(context.Background(), "list", errors.New("resolve release manager: missing release storage"))
	var wire *response.Error
	if !errors.As(err, &wire) || wire.Code != 500 {
		t.Fatalf("unexpected response: %v", err)
	}
	if !strings.Contains(logs.String(), "missing release storage") || !strings.Contains(logs.String(), `"operation":"list"`) {
		t.Fatalf("missing diagnostics: %s", logs.String())
	}
	if strings.Contains(err.Error(), "missing release storage") {
		t.Fatal("internal cause leaked in response")
	}
}

func TestReleaseOperationPreservesClientStatus(t *testing.T) {
	for _, code := range []int{400, 403, 404, 409} {
		original := fail(code, "synthetic client failure")
		got := releaseOperationError(context.Background(), "publish", fmt.Errorf("resolve release namespace: %w", original))
		if got != original {
			t.Fatalf("status %d was obscured: %v", code, got)
		}
	}
	if releaseOperationError(context.Background(), "list", nil) != nil {
		t.Fatal("success became failure")
	}
}
