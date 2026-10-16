// Package datlycmd exposes the original Studio native command assembly to embedding hosts.
package datlycmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"reflect"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
	_ "github.com/viant/bigquery"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/cmd/command"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
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
	if middleware == nil && options.LinkedArtifact == nil && !options.RequireComponentBinding {
		return (command.Service{}).Run(ctx, args, stdout, stderr)
	}
	return runBound(ctx, args, stdout, stderr, options, middleware)
}

// runBound uses the published standalone binding API. Command.Service does not
// forward these options in older supported Datly versions.
func runBound(ctx context.Context, args []string, stdout, stderr io.Writer, binding Options, middleware func(http.Handler) http.Handler) int {
	if len(args) == 0 || (args[0] != "run" && args[0] != "start") {
		fmt.Fprintln(stderr, "usage: datly run|start -conf configuration-URL")
		return 2
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	var location, constantURL string
	flags.StringVar(&location, "conf", "", "standalone configuration URL")
	flags.StringVar(&location, "c", "", "standalone configuration URL")
	flags.StringVar(&constantURL, "const", "", "instance constants file")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if location == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "a configuration URL and no positional arguments are required")
		return 2
	}
	cfg, err := (config.Loader{ConstURL: constantURL}).Load(ctx, location)
	if err == nil {
		resolved, e := cfg.ResolveConstants()
		err = e
		if err == nil {
			err = resolved.Validate()
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	options := standalone.Options{Config: cfg, LinkedArtifact: binding.LinkedArtifact, RequireComponentBinding: binding.RequireComponentBinding, RequireLinked: true, Diagnostics: stderr}
	if middleware != nil {
		// Fail closed on published versions without trusted HTTP wrapping support.
		field := reflect.ValueOf(&options).Elem().FieldByName("HTTPMiddleware")
		value := reflect.ValueOf(middleware)
		if !field.IsValid() || !field.CanSet() || !value.Type().AssignableTo(field.Type()) {
			fmt.Fprintln(stderr, "configured request middleware requires a Datly release with standalone HTTPMiddleware support")
			return 1
		}
		field.Set(value)
	}
	server, err := standalone.New(ctx, options)
	if err == nil {
		err = server.Serve(ctx, stdout)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
