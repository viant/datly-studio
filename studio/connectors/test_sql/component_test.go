package test_sql

import (
	"context"
	"errors"
	xresponse "github.com/viant/xdatly/response"
	"strings"
	"testing"
	"time"
)

func TestSQLExecutionTimeoutBounds(t *testing.T) {
	if duration, err := executionTimeout(nil); err != nil || duration != time.Minute {
		t.Fatalf("default: %v %v", duration, err)
	}
	for _, tc := range []struct {
		seconds int
		want    time.Duration
		invalid bool
	}{{0, 0, false}, {1, time.Second, false}, {300, 5 * time.Minute, false}, {-1, 0, true}, {301, 301 * time.Second, false}, {int(^uint(0) >> 1), 0, true}} {
		got, err := executionTimeout(&tc.seconds)
		if tc.invalid {
			if xresponse.ErrorStatusCode(err, 0) != 400 {
				t.Fatalf("%d: %v", tc.seconds, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("%d: %s %v", tc.seconds, got, err)
		}
	}
}
func TestSQLTimeoutPreservesDeadlineAndDoesNotExposeDriverSecrets(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := executionError(ctx, errors.New("private-driver-secret"), 60*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 504 || !strings.Contains(err.Error(), "60 seconds") || strings.Contains(err.Error(), "private-driver-secret") {
		t.Fatalf("timeout error: %v", err)
	}
	err = executionError(context.Background(), context.DeadlineExceeded, 300*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 504 {
		t.Fatal(err)
	}
	err = executionError(context.Background(), errors.New("private-driver-secret"), 60*time.Second)
	if xresponse.ErrorStatusCode(err, 0) != 400 || strings.Contains(err.Error(), "private-driver-secret") {
		t.Fatal(err)
	}
}
