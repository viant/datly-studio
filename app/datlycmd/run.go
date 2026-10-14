// Package datlycmd exposes the original Studio native command assembly to embedding hosts.
package datlycmd

import (
	"context"
	"fmt"
	"io"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
	_ "github.com/viant/bigquery"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/cmd/command"
	"github.com/viant/datly/exec"
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
	Authoring               *host.AuthoringConfig
	RequestMiddleware       func(http.Handler) http.Handler
	LinkedArtifact          *exec.LinkedArtifact
	RequireComponentBinding bool
	AccessProviderFactory   accessprovider.ProviderFactory
}

// Run runs the original native command with the existing process bindings.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return RunWithOptions(ctx, args, stdout, stderr, Options{})
}

// RunWithOptions installs the optional host factory before running the original
// command. Duplicate or late registrations fail before command execution.
// Factory registration is process-wide and immutable, including after Run exits.
func RunWithOptions(ctx context.Context, args []string, stdout, stderr io.Writer, options Options) int {
	middleware := options.RequestMiddleware
	if options.Authoring != nil {
		snapshot, err := host.NewAuthoring(*options.Authoring)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		prior := middleware
		middleware = func(next http.Handler) http.Handler {
			if prior != nil {
				next = prior(next)
			}
			return snapshot.Middleware(next)
		}
	}
	if options.AccessProviderFactory != nil {
		if err := accessprovider.RegisterEnvironmentFactory(options.AccessProviderFactory); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	return (command.Service{HTTPMiddleware: middleware, LinkedArtifact: options.LinkedArtifact, RequireComponentBinding: options.RequireComponentBinding}).Run(ctx, args, stdout, stderr)
}
