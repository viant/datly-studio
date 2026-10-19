package nativeapp

import appui "github.com/viant/datly-studio/ui"

// Navigation returns the embedded definition consumed by the native UI.
func Navigation() []byte { return appui.NativeNavigation() }
