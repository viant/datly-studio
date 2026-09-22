package datatest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// GeneratedModule is an isolated module used to compile and execute generated
// Datly component acceptance tests.
type GeneratedModule struct {
	Root       string
	ModulePath string
}

// NewGeneratedModule creates an isolated module linked to the current Studio
// and Datly source trees. It resolves Datly through Go rather than hardcoding a
// checkout path.
func NewGeneratedModule(t testing.TB) *GeneratedModule {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve datatest source path")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	command := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/viant/datly")
	command.Dir = projectRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	payload, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve Datly module: %v\n%s", err, payload)
	}
	datlyRoot := strings.TrimSpace(string(payload))
	root := t.TempDir()
	const modulePath = "github.com/viant/datly-studio/testgen"
	goMod := fmt.Sprintf(`module %s

go 1.25.8

require (
	github.com/viant/datly-studio v0.0.0
	github.com/viant/datly v0.0.0
)

replace github.com/viant/datly-studio => %s
replace github.com/viant/datly => %s
`, modulePath, projectRoot, datlyRoot)
	if err = os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write generated go.mod: %v", err)
	}
	return &GeneratedModule{Root: root, ModulePath: modulePath}
}

// Write writes one generated-module source file.
func (m *GeneratedModule) Write(t testing.TB, relative string, content []byte) {
	t.Helper()
	target := filepath.Join(m.Root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create generated directory: %v", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write generated file %s: %v", relative, err)
	}
}

// Test compiles and runs every generated package.
func (m *GeneratedModule) Test(t testing.TB) {
	t.Helper()
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = m.Root
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated Datly component tests: %v\n%s", err, output)
	}
}

// AssertFilesAbsent guards the compact generated-package contract.
func AssertFilesAbsent(t testing.TB, directory string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(directory, name)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("obsolete generated implementation file remains: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect generated file %s: %v", path, err)
		}
	}
}
