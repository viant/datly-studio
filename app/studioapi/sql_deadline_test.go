package studioapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type executionDeadlineWriter struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *executionDeadlineWriter) SetWriteDeadline(value time.Time) error {
	w.deadlines = append(w.deadlines, value)
	return nil
}
func TestExplicitExecutionDeadline(t *testing.T) {
	for _, tc := range []struct {
		path  string
		clear bool
	}{{"/v1/studio/sdk/connectors.test_sql", true}, {"/v1/studio/sdk-mcp/mcp", true}, {"/v1/studio/mcp/mcp", true}, {"/v1/studio/sdk/connectors.list", false}, {"/", false}} {
		t.Run(tc.path, func(t *testing.T) {
			w := &executionDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
			called := false
			explicitExecutionDeadline(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(w, httptest.NewRequest("POST", tc.path, nil))
			if !called || (len(w.deadlines) == 1) != tc.clear {
				t.Fatalf("forwarded=%t deadlines=%v", called, w.deadlines)
			}
			if tc.clear && !w.deadlines[0].IsZero() {
				t.Fatal("SQL retained a hidden BFF deadline")
			}
		})
	}
}
