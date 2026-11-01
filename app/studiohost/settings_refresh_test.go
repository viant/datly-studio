package studiohost_test

import (
	"encoding/json"
	"testing"
	"time"
)

type namespaceControlResult struct {
	IsError           bool
	StructuredContent struct {
		NamespaceID string `json:"namespaceId"`
		ETag        int64  `json:"etag"`
		Host        struct {
			Status   string
			Revision int64
		}
		Settings struct {
			Security struct{ RequireInternalComponents bool }
		}
	}
}

func verifyNamespaceSettingsRefresh(t *testing.T, call func(string, map[string]any) []byte) {
	t.Helper()
	invoke := func(name string, arguments map[string]any) *namespaceControlResult {
		t.Helper()
		body := call("tools/call", map[string]any{"name": name, "arguments": arguments})
		var wire struct {
			Result *namespaceControlResult
			Error  any
		}
		if err := json.Unmarshal(body, &wire); err != nil || wire.Error != nil || wire.Result == nil {
			t.Fatalf("namespace control response: %s (%v)", body, err)
		}
		if wire.Result.IsError && name != "studio.sdk.runtime.status" {
			t.Fatalf("namespace control %s failed: %s", name, body)
		}
		return wire.Result
	}
	created := invoke("studio.sdk.namespaces.create", map[string]any{"name": "policy_probe", "title": "Policy probe", "mcpEnabled": true})
	if created.IsError || len(created.StructuredContent.NamespaceID) != 64 {
		t.Fatalf("namespace creation failed: %+v", created)
	}
	arguments := map[string]any{"namespaceId": created.StructuredContent.NamespaceID}
	var before *namespaceControlResult
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		before = invoke("studio.sdk.runtime.status", arguments)
		if !before.IsError && before.StructuredContent.Host.Status == "ready" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if before == nil || before.IsError || before.StructuredContent.Host.Status != "ready" {
		t.Fatal("policy probe runtime did not become ready")
	}
	settings := invoke("studio.sdk.namespaces.settings_get", map[string]any{"name": "policy_probe"})
	if settings.IsError || settings.StructuredContent.ETag <= 0 {
		t.Fatal("settings were unavailable")
	}
	updated := invoke("studio.sdk.namespaces.settings_update", map[string]any{"name": "policy_probe", "input": map[string]any{"etag": settings.StructuredContent.ETag, "settings": map[string]any{"security": map[string]any{"requireInternalComponents": true}}}})
	if updated.IsError || !updated.StructuredContent.Settings.Security.RequireInternalComponents {
		t.Fatalf("post-commit refresh failed: %+v", updated)
	}
	after := invoke("studio.sdk.runtime.status", arguments)
	if after.IsError || after.StructuredContent.Host.Revision <= before.StructuredContent.Host.Revision {
		t.Fatalf("settings commit did not refresh the running generation: before=%+v after=%+v", before, after)
	}
}
