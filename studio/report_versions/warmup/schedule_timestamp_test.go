package warmup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

func testWarmupScheduleTimestamps(ctx context.Context, t *testing.T, db *sql.DB, namespace, bearer, httpAddress, mcpAddress string) {
	t.Helper()
	for _, transport := range []string{"HTTP", "MCP"} {
		t.Run(transport, func(t *testing.T) {
			invoke := func(t *testing.T, operation string, input map[string]any) ([]byte, bool) {
				t.Helper()
				args := map[string]any{"namespaceId": namespace, "reportId": "named-root", "versionNo": 1, "input": input}
				address := "http://" + httpAddress + "/v1/studio/sdk/versions.warmup_schedule_" + operation
				var payload any = args
				if transport == "MCP" {
					address = "http://" + mcpAddress + "/mcp"
					payload = map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "studio.sdk.versions.warmup_schedule_" + operation, "arguments": args}}
				}
				body, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", bearer)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json, text/event-stream")
				req.Header.Set("X-Studio-Namespace", namespace)
				response, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				data, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if transport == "HTTP" {
					if operation == "save" && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusBadRequest {
						t.Fatalf("unexpected save status %d: %s", response.StatusCode, data)
					}
					return data, response.StatusCode == http.StatusOK
				}
				var wire struct {
					Error  json.RawMessage `json:"error"`
					Result struct {
						IsError bool            `json:"isError"`
						Data    json.RawMessage `json:"structuredContent"`
					} `json:"result"`
				}
				if err = json.Unmarshal(data, &wire); err != nil {
					t.Fatalf("MCP response: %s: %v", data, err)
				}
				ok := response.StatusCode == http.StatusOK && len(wire.Error) == 0 && !wire.Result.IsError
				if ok {
					return wire.Result.Data, true
				}
				return data, false
			}
			invalid := []struct {
				name    string
				value   any
				omitted bool
			}{
				{name: "time only", value: "23:14"}, {name: "malformed", value: "not-a-date"},
				{name: "empty", value: ""}, {name: "missing", omitted: true}, {name: "null", value: nil},
				{name: "zero", value: "0001-01-01T00:00:00Z"}, {name: "year zero", value: "0000-01-01T00:00:00Z"},
				{name: "offset into year zero", value: "0001-01-01T00:30:00+01:00"},
				{name: "rounds to zero", value: "0001-01-01T00:00:00.000000001Z"},
			}
			reject := func(t *testing.T, revision int64) {
				for _, tc := range invalid {
					t.Run(tc.name, func(t *testing.T) {
						input := map[string]any{"expectedRevision": revision, "intervalSeconds": 60, "enabled": true}
						if !tc.omitted {
							input["firstRunAt"] = tc.value
						}
						data, ok := invoke(t, "save", input)
						if ok || !strings.Contains(string(data), "firstRunAt") {
							t.Errorf("expected field-specific rejection: %s", data)
						}
						var count int
						if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_warmup_schedules WHERE report_id='named-root'").Scan(&count); err != nil {
							t.Fatal(err)
						}
						if revision == 0 && count != 0 {
							t.Fatal("rejected create persisted a schedule")
						}
						if revision != 0 {
							var actual int64
							var interval int64
							var enabled bool
							var first, next time.Time
							if err := db.QueryRowContext(ctx, "SELECT revision,first_run_at,next_due_at,interval_seconds,enabled FROM component_warmup_schedules WHERE report_id='named-root'").Scan(&actual, &first, &next, &interval, &enabled); err != nil {
								t.Fatal(err)
							}
							want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
							if actual != revision || !first.Equal(want) || !next.Equal(want) || interval != 3600 || !enabled {
								t.Fatalf("rejected update changed schedule: revision=%d first=%v next=%v", actual, first, next)
							}
						}
					})
				}
			}
			if !t.Run("create", func(t *testing.T) { reject(t, 0) }) {
				return
			}
			for i, stamp := range []string{"2027-01-02T03:04:05.123456Z", "2027-01-02T08:34:05.123456+05:30", "2020-01-02T03:04:05Z"} {
				data, ok := invoke(t, "save", map[string]any{"expectedRevision": i, "intervalSeconds": 3600, "firstRunAt": stamp, "enabled": true})
				var result sdk.WarmupScheduleResult
				if !ok || json.Unmarshal(data, &result) != nil || result.Schedule == nil {
					t.Fatalf("valid save: %s", data)
				}
				want, err := time.Parse(time.RFC3339Nano, stamp)
				if err != nil {
					t.Fatal(err)
				}
				if result.Schedule.Revision != int64(i+1) || !result.Schedule.FirstRunAt.Equal(want) || !result.Schedule.NextDueAt.Equal(want) {
					t.Fatalf("timestamp changed: %+v", result.Schedule)
				}
			}
			t.Run("update", func(t *testing.T) { reject(t, 3) })
			if data, ok := invoke(t, "delete", map[string]any{"expectedRevision": 3}); !ok {
				t.Fatalf("delete schedule: %s", data)
			}
		})
	}
}
