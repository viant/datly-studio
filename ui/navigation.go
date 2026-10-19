package ui

import _ "embed"

//go:embed src/nativeNavigation.json
var nativeNavigation []byte

// NativeNavigation returns an owned copy of the native UI menu definition.
func NativeNavigation() []byte { return append([]byte(nil), nativeNavigation...) }
