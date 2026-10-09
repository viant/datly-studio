// Package datlycmd exposes the original Studio native command assembly to embedding hosts.
package datlycmd

import (
	"context"
	"fmt"
	"io"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/bigquery"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly/cmd/command"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/pg"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

// Options contains process-owned startup bindings. A nil factory preserves an
// existing registration, or the generic provider when none was registered.
// Hosts sharing this process must register once and pass nil on later calls.
type Options struct {
	AccessProviderFactory accessprovider.ProviderFactory
}

// Run runs the original native command with the existing process bindings.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return RunWithOptions(ctx, args, stdout, stderr, Options{})
}

// RunWithOptions installs the optional host factory before running the original
// command. Duplicate or late registrations fail before command execution.
// Factory registration is process-wide and immutable, including after Run exits.
func RunWithOptions(ctx context.Context, args []string, stdout, stderr io.Writer, options Options) int {
	if options.AccessProviderFactory != nil {
		if err := accessprovider.RegisterEnvironmentFactory(options.AccessProviderFactory); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	return (command.Service{}).Run(ctx, args, stdout, stderr)
}
