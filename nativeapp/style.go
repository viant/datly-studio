package nativeapp

import appui "github.com/viant/datly-studio/ui"

// WorkspaceStyle returns the native workspace stylesheet as a style resource.
func WorkspaceStyle() []byte { return appui.NativeWorkspaceStyle() }
