package host

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	xcodec "github.com/viant/xdatly/codec"
	"strings"
	"time"
)

// RunWarmupScheduler is owned by the process host, not an HTTP/MCP session.
// The provider renews the existing signed system identity; it does not acquire
// permissions by being a scheduler. Nil credentials leave scheduling disabled.
// Already accepted jobs retain the ordinary warmup deadline and recovery rules.
func RunWarmupScheduler(ctx context.Context, db *sql.DB, config AuthoringConfig, credentials sdk.SystemCredentialProvider, factory xcodec.Factory, observe func(error)) error {
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
	defer authorizer.Close(context.WithoutCancel(ctx))
	types, e := authoring.RuntimeTypes()
	if e != nil {
		return e
	}
	timeout := authoring.WarmupTimeout().String()
	engine := &sqltransport.Transport{DB: db, Authorizer: authorizer, WarmupOperationTimeout: &timeout, SystemCredentialProvider: credentials,
		Warmup: preview.Dynamic{StudioDB: db, ModulePath: authoring.ModulePath(), Types: types, Const: authoring.Constants(), Observability: authoring.Observability(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies(), Predicates: authoring.PredicateCatalog()}}
	ctx = codecs.WithFactory(ctx, factory)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		service, e := sdk.WithSystemIdentity(ctx, credentials)
		if e == nil {
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
			return nil
		case <-ticker.C:
		}
	}
}
