package host

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/internal/warmuplog"
	"github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	xcodec "github.com/viant/xdatly/codec"
	"log/slog"
	"strings"
	"time"
)

const warmupSchedulerPollInterval = time.Second

// RunWarmupScheduler is owned by the process host, not an HTTP/MCP session.
// The provider renews the existing signed system identity; it does not acquire
// permissions by being a scheduler. Nil credentials leave scheduling disabled.
// Already accepted jobs retain the ordinary warmup deadline and recovery rules.
func RunWarmupScheduler(ctx context.Context, db *sql.DB, config AuthoringConfig, credentials sdk.SystemCredentialProvider, factory xcodec.Factory, observe func(error)) (runErr error) {
	ctx = warmuplog.WithScheduler(ctx)
	started, loopExited := false, false
	shutdown := "clean"
	defer func() {
		event := "Studio warmup scheduler stopped"
		if !started {
			event = "Studio warmup scheduler start failed"
		}

		if !started || runErr != nil {
			shutdown = "failed"
		} else if !loopExited && shutdown == "clean" {
			shutdown = "aborted"
		}
		fields := []any{"poll_interval_ms", warmupSchedulerPollInterval.Milliseconds(), "outcome", shutdown}
		level := slog.LevelInfo
		if shutdown != "clean" {
			level = slog.LevelError
		}
		if runErr != nil {
			code, message := warmuplog.Diagnostic(runErr)
			fields = append(fields, "error_code", code, "diagnostic", message)
		}

		warmuplog.Emit(context.WithoutCancel(ctx), level, event, warmuplog.Scope{}, fields...)
	}()
	if credentials == nil {
		return fmt.Errorf("warmup scheduling requires a service credential provider")
	}
	authoring, e := NewAuthoring(config)
	if e != nil {
		return e
	}
	defer authoring.Close()
	authorizer, e := authorization.NewSDKAuthorizer(db)
	if e != nil {
		return e
	}
	defer func() {
		if e := authorizer.Close(context.WithoutCancel(ctx)); e != nil {
			shutdown = "cleanup_failed"
			warmuplog.Failure(context.WithoutCancel(ctx), warmuplog.Scope{}, "shutdown", "Studio warmup scheduler cleanup failed", e)
		}
	}()
	types, e := authoring.RuntimeTypes()
	if e != nil {
		return e
	}
	timeout := authoring.WarmupTimeout().String()
	engine := &sqltransport.Transport{DB: db, Authorizer: authorizer, WarmupOperationTimeout: &timeout, SystemCredentialProvider: credentials,
		Warmup: preview.Dynamic{StudioDB: db, ModulePath: authoring.ModulePath(), Types: types, Const: authoring.Constants(), Observability: authoring.Observability(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies(), Predicates: authoring.PredicateCatalog()}}
	ctx = codecs.WithFactory(ctx, factory)
	ticker := time.NewTicker(warmupSchedulerPollInterval)
	defer ticker.Stop()
	started = true
	warmuplog.Emit(ctx, slog.LevelInfo, "Studio warmup scheduler started", warmuplog.Scope{}, "poll_interval_ms", warmupSchedulerPollInterval.Milliseconds())
	for {
		if ctx.Err() != nil {
			loopExited = true
			return nil
		}
		service, e := sdk.WithSystemIdentity(ctx, credentials)
		if e != nil && ctx.Err() == nil {
			warmuplog.Failure(ctx, warmuplog.Scope{}, "credentials", "Studio warmup scheduler credential failure", e)
		}
		if e == nil {
			warmuplog.Recovered(ctx, warmuplog.Scope{}, "credentials")
			// System providers return a complete Authorization header. The native
			// warmup continuation retains its token separately, as the HTTP owner does.
			credential, _ := sdk.VerifiedCredentialFromContext(service)
			credential.Bearer = strings.TrimSpace(strings.TrimPrefix(credential.Bearer, "Bearer "))
			service = sdk.WithVerifiedCredential(service, credential)
			e = engine.RunDueWarmupSchedules(service)
		}
		if e != nil && ctx.Err() == nil && observe != nil {
			observe(e)
		}
		select {
		case <-ctx.Done():
			loopExited = true
			return nil
		case <-ticker.C:
		}
	}
}
