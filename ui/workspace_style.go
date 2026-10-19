package ui

import (
	_ "embed"
	"encoding/json"
)

//go:embed src/studio.css
var workspaceCSS string

//go:embed src/resourceAccess.css
var accessCSS string

// NativeWorkspaceStyle exports the same stylesheet sources as the native UI.
// The consuming workspace owns scoping and lifecycle of the style resource.
func NativeWorkspaceStyle() []byte {
	data, _ := json.Marshal(map[string]any{"schemaVersion": 1, "css": workspaceCSS + "\n" + accessCSS})
	return data
}
