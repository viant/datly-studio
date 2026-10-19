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
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/cmd/command"
	"github.com/viant/datly/exec"
	nativeauth "github.com/viant/datly/runtime/auth"
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
	middleware, err := prepareMiddleware(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if middleware == nil && options.LinkedArtifact == nil && !options.RequireComponentBinding {
		return (command.Service{}).Run(ctx, args, stdout, stderr)
	}
	return runBound(ctx, args, stdout, stderr, options, middleware)
}

// New constructs the native Studio server without parsing CLI arguments or
// spawning processes. The caller serves it, waits for readiness, and shuts it
// down through Datly's Server lifecycle. Configuration must already be resolved.
// A provider factory is process-wide; register it once across hosted services.
func New(ctx context.Context, cfg *config.Config, options Options, diagnostics io.Writer) (*standalone.Server, error) {
	if ctx == nil {
		return nil, fmt.Errorf("native Studio context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("native Studio configuration is required")
	}
	middleware, err := prepareMiddleware(options)
	if err != nil {
		return nil, err
	}
	return newServer(ctx, cfg, options, middleware, diagnostics)
}

func prepareMiddleware(options Options) (func(http.Handler) http.Handler, error) {
	middleware := options.RequestMiddleware
	if options.Authoring != nil {
		snapshot, err := host.NewAuthoring(*options.Authoring)
		if err != nil {
			return nil, err
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
			return nil, err
		}
	}
	return middleware, nil
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
	if err := serveConfig(ctx, cfg, binding, middleware, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// serveConfig owns the native server lifecycle independently of CLI parsing.
func serveConfig(ctx context.Context, cfg *config.Config, binding Options, middleware func(http.Handler) http.Handler, stdout, diagnostics io.Writer) error {
	server, err := newServer(ctx, cfg, binding, middleware, diagnostics)
	if err != nil {
		return err
	}
	return server.Serve(ctx, stdout)
}

func newServer(ctx context.Context, cfg *config.Config, binding Options, middleware func(http.Handler) http.Handler, diagnostics io.Writer) (*standalone.Server, error) {
	// Reuse Datly's verifier configuration for dynamically authored JwtClaim inputs.
	if cfg.JWTValidator != nil {
		jwtCodec, err := nativeauth.New(ctx, &nativeauth.Config{JWTValidator: cfg.JWTValidator, ClaimPolicy: cfg.JWTClaims})
		if err != nil {
			return nil, err
		}
		codecFactory := studiocodecs.New(jwtCodec)
		previous := middleware
		middleware = func(next http.Handler) http.Handler {
			if previous != nil {
				next = previous(next)
			}
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(studiocodecs.WithFactory(r.Context(), codecFactory)))
			})
		}
	}
	options := standalone.Options{Config: cfg, LinkedArtifact: binding.LinkedArtifact, RequireComponentBinding: binding.RequireComponentBinding, RequireLinked: true, Diagnostics: diagnostics}
	if middleware != nil {
		// Fail closed on published versions without trusted HTTP wrapping support.
		field := reflect.ValueOf(&options).Elem().FieldByName("HTTPMiddleware")
		value := reflect.ValueOf(middleware)
		if !field.IsValid() || !field.CanSet() || !value.Type().AssignableTo(field.Type()) {
			return nil, fmt.Errorf("configured request middleware requires a Datly release with standalone HTTPMiddleware support")
		}
		field.Set(value)
	}
	return standalone.New(ctx, options)
}
