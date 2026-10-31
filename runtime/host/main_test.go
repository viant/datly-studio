package host

import (
	"fmt"
	"os"
	"testing"
)

// Host tests must not seed the checkout or the user's configured workspace.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "datly-studio-host-workspace-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = os.Setenv("AGENTLY_WORKSPACE", root); err != nil {
		_ = os.RemoveAll(root)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}
