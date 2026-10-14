package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testSink struct{ records chan Record }

func (s testSink) Write(_ context.Context, r Record) error { s.records <- r; return nil }
func TestCapture(t *testing.T) {
	records := make(chan Record, 1)
	l := New(Config{Enabled: true, Sanitize: RedactJSON, Identity: func(*http.Request) Identity { return Identity{Subject: "verified"} }}, testSink{records})
	handler := l.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":{"password":"private","value":2}}`))
		w.(http.Flusher).Flush()
	}))
	request := httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"method":"tools/call","params":{"name":"read","token":"private"}}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if err := l.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	record := <-records
	if strings.Contains(string(record.RequestBody), "private") || strings.Contains(string(record.ResponseBody), "private") {
		t.Fatal("unredacted secret")
	}
	if record.Subject != "verified" || record.ToolName != "read" || !strings.Contains(response.Body.String(), "private") {
		t.Fatal("capture changed response or identity")
	}
}
func TestDisabled(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	l := New(Config{}, nil)
	if l.Wrap(next) == nil {
		t.Fatal("nil handler")
	}
	if err := l.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
