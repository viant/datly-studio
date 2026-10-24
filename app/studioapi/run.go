// Package studioapi starts the Studio SDK and browser host.
package studioapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	"github.com/viant/datly-studio/store/sql/accesscatalog"
	"github.com/viant/datly/constant"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
	accessstore "github.com/viant/authz/component/store/sql"
	_ "github.com/viant/bigquery"
	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly-studio/sdk/connectivity"
	"github.com/viant/datly-studio/sdk/httptransport"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/host"
	versionclone "github.com/viant/datly-studio/studio/report_versions/clone"
	"github.com/viant/scy/auth/jwt/verifier"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/pg"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

// nativeSDKPaths are exact BFF mounts for generated static Datly SDK routes.
// Every other SDK operation remains on the generic SDK gateway.
var nativeSDKPaths = []string{
	"/v1/authz/sdk/namedPolicies.manage",
	"/v1/authz/sdk/policies.create",
	"/v1/authz/sdk/policies.get",
	"/v1/authz/sdk/policies.context",
	"/v1/authz/sdk/policies.replace",
	"/v1/authz/sdk/authorization.check",
	"/v1/studio/sdk/versions.clone",
	"/v1/studio/sdk/access.list",
	"/v1/studio/sdk/access.context",
	"/v1/studio/sdk/access.get",
	"/v1/studio/sdk/access.replace",
	"/v1/studio/sdk/authorization_predicates.types",
	"/v1/studio/sdk/codecs.types",
	"/v1/studio/sdk/authorization_predicates.get",
	"/v1/studio/sdk/authorization_predicates.create",
	"/v1/studio/sdk/authorization_predicates.update",
	"/v1/studio/sdk/authorization_predicates.delete",
	"/v1/studio/sdk/authorization_predicates.list",
	"/v1/studio/sdk/acl.list",
	"/v1/studio/sdk/acl.delete",
	"/v1/studio/sdk/acl.upsert",
	"/v1/studio/sdk/connectors.get",
	"/v1/studio/sdk/connectors.create",
	"/v1/studio/sdk/connectors.schemas",
	"/v1/studio/sdk/connectors.discover",
	"/v1/studio/sdk/audit.status",
	"/v1/studio/sdk/audit.configure",
	"/v1/studio/sdk/audit.list",
	"/v1/studio/sdk/audit.export",
	"/v1/studio/sdk/connectors.tables",
	"/v1/studio/sdk/connectors.table",
	"/v1/studio/sdk/connectors.test",
	"/v1/studio/sdk/connectors.test_sql",
	"/v1/studio/sdk/preview.execute",
	"/v1/studio/sdk/versions.validate",
	"/v1/studio/sdk/versions.test_view",
	"/v1/studio/sdk/versions.test_relation",
	"/v1/studio/sdk/versions.test_compose",
	"/v1/studio/sdk/versions.warmup",
	"/v1/studio/sdk/versions.invalidate_cache",
	"/v1/studio/sdk/connectors.activate",
	"/v1/studio/sdk/connectors.delete",
	"/v1/studio/sdk/connectors.disable",
	"/v1/studio/sdk/connectors.update",
	"/v1/studio/sdk/connectors.list",
	"/v1/studio/sdk/namespaces.get",
	"/v1/studio/sdk/namespaces.create",
	"/v1/studio/sdk/namespaces.delete",
	"/v1/studio/sdk/namespaces.update",
	"/v1/studio/sdk/namespaces.list",
	"/v1/studio/sdk/namespace_skills.list",
	"/v1/studio/sdk/namespace_skills.get",
	"/v1/studio/sdk/namespace_skills.create",
	"/v1/studio/sdk/namespace_skills.update",
	"/v1/studio/sdk/namespace_skills.delete",
	"/v1/studio/sdk/namespace_skills.revisions",
	"/v1/studio/sdk/namespace_skills.publish",
	"/v1/studio/sdk/namespace_skills.unpublish",
	"/v1/studio/sdk/publications.get",
	"/v1/studio/sdk/publications.publish",
	"/v1/studio/sdk/publications.rollback",
	"/v1/studio/sdk/publications.unpublish",
	"/v1/studio/sdk/publications.events.list",
	"/v1/studio/sdk/runtime.status",
	"/v1/studio/sdk/components.get",
	"/v1/studio/sdk/components.create",
	"/v1/studio/sdk/components.update",
	"/v1/studio/sdk/components.list",
	"/v1/studio/sdk/versions.get",
	"/v1/studio/sdk/versions.create",
	"/v1/studio/sdk/versions.load_dql",
	"/v1/studio/sdk/versions.load_archive",
	"/v1/studio/sdk/versions.inspect",
	"/v1/studio/sdk/versions.apply",
	"/v1/studio/sdk/versions.builder",
	"/v1/studio/sdk/versions.list",
	"/v1/studio/sdk/versions.export_dql",
	"/v1/studio/sdk/versions.descriptor",
	"/v1/studio/sdk/versions.download",
	"/v1/studio/sdk/versions.warmup_get",
	"/v1/studio/sdk/versions.warmup_list",
	"/v1/studio/sdk/resources.get",
	"/v1/studio/sdk/resources.upsert_file",
	"/v1/studio/sdk/resources.delete_file",
	"/v1/studio/sdk/resources.upsert_folder",
	"/v1/studio/sdk/resources.delete_folder",
	"/v1/studio/sdk/resources.upsert_skill",
	"/v1/studio/sdk/resources.delete_skill",
}

// Run starts the Studio API with its standard command-line flags. Applications
// can link their own predicates and components before calling Run from main.
func Run() {
	RunWithOptions(Options{})
}

// Options supplies process-owned identity wiring for embedders. The factory
// is invoked once during startup and is never selected from request data.
type RuntimeControl interface {
	Reload(context.Context, int64) error
	ProbeRuntime(context.Context) (*sdk.RuntimeHost, error)
}

type Options struct {
	// AuthoringModule is selected by the embedding host, never a request.
	AuthoringModule string
	Const           *constant.Values
	// RuntimeControl avoids an internal administration listener in embedded hosts.
	RuntimeControl RuntimeControl

	Login *Login
	// StaticFS serves a compiled public UI (for example embed.FS) when
	// -static-root is unset. A disk build remains an explicit operator override.
	StaticFS fs.FS
	// Listener transfers an already-reserved HTTP listener to Serve. The host
	// closes it on startup failure and after HTTP drain, allowing one parent to
	// reserve its public socket before deriving callback and browser origins.
	Listener net.Listener
	// Ready is called after startup and listener acquisition, before serving.
	// It must return promptly. The parent owns aggregate host readiness.
	Ready func(net.Addr)
	// DatabaseDriver selects host-owned catalog storage; empty preserves SQLite.
	DatabaseDriver string
	// Database is an optional caller-owned catalog connection, opened from the
	// shared Datly connector configuration. Serve never closes this handle.
	// DatabaseDriver must accompany it; the caller closes it after all services stop.
	Database *sql.DB
	// JWTValidator reuses the shared Datly/SCY verifier configuration. When set,
	// certificate/key CLI overrides are rejected. Claim and session checks remain.
	JWTValidator *verifier.Config
	// HTTPMiddleware is an optional process-owned listener boundary. Unset
	// preserves the complete standalone routing and authentication behavior.
	HTTPMiddleware func(http.Handler) http.Handler
	// SchemaVerifier validates an existing database instead of the default schema initialization.
	// Only trusted embedding hosts supply this callback.
	SchemaVerifier        func(context.Context, *sql.DB) error
	AccessProviderFactory accessprovider.ProviderFactory
	AccessProviderConfig  *accessprovider.Config
	// ReleaseUse is injected by a trusted host after binding a verified account
	// scope and exact published component companion index. Browser payloads
	// cannot configure it.
	ReleaseUse                 func(context.Context, sqltransport.AuthorizationRequest) (bool, error)
	ReleaseNamespace           string
	ReleaseManagementAvailable bool
	// Modules adds process-owned, independently authenticated extension APIs.
	// Embedders supply credential protection and business authorization in each
	// handler. Modules never inherit development identities or browser facts.
	Modules              ModuleFactory
	AuthenticatedModules AuthenticatedModuleFactory
}

// RunWithOptions starts Studio with optional host-owned resource identity
// bootstrap while preserving the standard command-line configuration.
func RunWithOptions(options Options) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := Serve(ctx, os.Args[1:], options); err != nil && !errors.Is(err, flag.ErrHelp) {
		log.Fatal(err)
	}
}

// Serve runs the host with caller-owned cancellation and isolated arguments.
// It returns startup and serving failures so embedding applications can release
// their other services. It never mutates os.Args or the global flag registry.
func Serve(lifecycleCtx context.Context, args []string, options Options) (result error) {
	if lifecycleCtx == nil {
		return errors.New("Studio lifecycle context is required")
	}
	if err := lifecycleCtx.Err(); err != nil {
		return err
	}
	if options.Listener != nil {
		defer options.Listener.Close()
	}
	lifecycleCtx, stop := context.WithCancel(lifecycleCtx)
	defer stop()
	flags := flag.NewFlagSet("studio", flag.ContinueOnError)
	address := flags.String("address", "127.0.0.1:8080", "SDK HTTP listen address")
	dsn := flags.String("dsn", "file:.data/studio.db?cache=shared", "Studio catalog DSN")
	defaultDriver := options.DatabaseDriver
	if defaultDriver == "" {
		defaultDriver = "sqlite"
	}
	databaseDriver := flags.String("driver", defaultDriver, "Studio catalog driver: sqlite or mysql")
	subject := flags.String("subject", "awitas", "local development principal")
	mode := flags.String("mode", "development", "gateway mode: development or authenticated")
	cookieName := flags.String("session-cookie", bffauth.DefaultCookieName, "authenticated BFF session cookie name")
	jwtCertURL := flags.String("jwt-cert-url", os.Getenv("STUDIO_JWT_CERT_URL"), "authenticated JWT certificate or JWKS URL")
	jwtPublicKey := flags.String("jwt-public-key", os.Getenv("STUDIO_JWT_PUBLIC_KEY_FILE"), "authenticated RSA public-key SCY resource URL, optionally URL|key")
	jwtIssuer := flags.String("jwt-issuer", os.Getenv("STUDIO_JWT_ISSUER"), "authenticated JWT issuer")
	jwtAudience := flags.String("jwt-audience", os.Getenv("STUDIO_JWT_AUDIENCE"), "authenticated JWT audience")
	sessionKey := flags.String("session-key", os.Getenv("STUDIO_SESSION_KEY"), "base64-encoded 32-byte BFF session encryption key")
	dynamicHTTPURL := flags.String("dynamic-http-url", "http://127.0.0.1:8082", "dynamic Datly HTTP target")
	dynamicMCPURL := flags.String("dynamic-mcp-url", "http://127.0.0.1:8091", "dynamic Datly MCP target")
	staticDatlyURL := flags.String("static-datly-url", "http://127.0.0.1:8081", "static Studio Datly component target")
	staticMCPURL := flags.String("static-mcp-url", "http://127.0.0.1:8090", "static Studio SDK MCP target")
	extensionBackendURL := flags.String("extension-backend-url", "", "trusted extension backend origin (authenticated mode only)")
	extensionUpstreamPrefix := flags.String("extension-upstream-prefix", "", "allowlisted extension upstream path prefix, for example /api/widgets")
	dynamicAdminToken := flags.String("dynamic-admin-token", os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN"), "dynamic runtime reload token")
	allowedOrigin := flags.String("allowed-origin", os.Getenv("STUDIO_ALLOWED_ORIGIN"), "exact Studio browser origin")
	staticRoot := flags.String("static-root", "", "optional absolute directory of public built UI assets served on the Studio BFF origin")
	sessionPruneInterval := flags.Duration("session-prune-interval", time.Minute, "how often to prune bounded batches of expired BFF sessions; 0 disables background cleanup")
	loginAuthURL := flags.String("login-auth-url", "", "optional OAuth authorization endpoint for the BFF login route")
	loginTokenURL := flags.String("login-token-url", "", "OAuth token endpoint paired with -login-auth-url")
	loginClientID := flags.String("login-client-id", "", "registered OAuth client ID for BFF login")
	loginClientSecretFile := flags.String("login-client-secret-file", "", "optional file containing OAuth client secret; omit for a public PKCE client")
	loginRedirectURL := flags.String("login-redirect-url", "", "exact public callback URL ending in /v1/studio/auth/callback")
	loginScopes := flags.String("login-scopes", "openid,profile,email", "comma-separated OAuth scopes for BFF login, including openid")
	loginIDToken := flags.Bool("login-id-token", false, "use an auth-only session and return verified OIDC ID tokens to the browser for direct Datly calls")
	accessTenant := flags.String("access-tenant", os.Getenv("STUDIO_ACCESS_TENANT"), "server-owned tenant for scoped exact-version previews")
	accessIssuer := flags.String("access-issuer", os.Getenv("STUDIO_ACCESS_ISSUER"), "dedicated ACL token issuer")
	accessAudience := flags.String("access-audience", os.Getenv("STUDIO_ACCESS_AUDIENCE"), "dedicated ACL token audience")
	accessKey := flags.String("access-public-key", os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), "ACL issuer RSA public key PEM file")
	accessCertURL := flags.String("access-cert-url", os.Getenv("STUDIO_ACCESS_CERT_URL"), "ACL issuer JWKS endpoint for rotating keys")
	accessUserInfoURL := flags.String("access-user-info-url", os.Getenv("STUDIO_ACCESS_USER_INFO_URL"), "optional trusted user-info endpoint for ID-token ACL facts")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("Studio does not accept positional arguments")
	}
	if *sessionPruneInterval < 0 {
		return errors.New("-session-prune-interval must not be negative")
	}
	resolvedMode := strings.ToLower(strings.TrimSpace(*mode))
	if err := validateGatewayMode(resolvedMode); err != nil {
		return err
	}
	extensionConfig, err := resolveExtensionProxyConfig(resolvedMode, *extensionBackendURL, *extensionUpstreamPrefix)
	if err != nil {
		return err
	}
	var adminToken string
	if options.RuntimeControl == nil {
		adminToken, err = resolveRuntimeAdminToken(resolvedMode, *dynamicAdminToken)
		if err != nil {
			return err
		}
	}

	origin, err := resolveAllowedOrigin(resolvedMode, *allowedOrigin)
	if err != nil {
		return err
	}
	var loginSecret string
	if *loginClientSecretFile != "" {
		secretBytes, readErr := os.ReadFile(*loginClientSecretFile)
		if readErr != nil {
			return readErr
		}
		loginSecret = strings.TrimSpace(string(secretBytes))
		if loginSecret == "" {
			return errors.New("-login-client-secret-file is empty")
		}
	}
	if login := options.Login; login != nil {
		if *loginAuthURL != "" || *loginTokenURL != "" || *loginClientID != "" || *loginClientSecretFile != "" || *loginRedirectURL != "" {
			return errors.New("host-owned Login cannot be combined with login CLI overrides")
		}
		*loginAuthURL, *loginTokenURL, *loginClientID, *loginRedirectURL = login.AuthorizationURL, login.TokenURL, login.ClientID, login.RedirectURL
		loginSecret = login.ClientSecret
		if len(login.Scopes) > 0 {
			*loginScopes = strings.Join(login.Scopes, ",")
		}
	}
	loginOAuth, err := resolveLoginConfig(resolvedMode, origin, *loginAuthURL, *loginTokenURL, *loginClientID, loginSecret, *loginRedirectURL, *loginScopes)
	if err != nil {
		return err
	}
	if *loginIDToken && loginOAuth == nil {
		return errors.New("-login-id-token requires configured OAuth login endpoints")
	}
	if *loginIDToken && extensionConfig != nil {
		return errors.New("identity-token mode does not support the BFF extension proxy")
	}
	if *databaseDriver != "sqlite" && *databaseDriver != "mysql" {
		return errors.New("Studio catalog driver must be sqlite or mysql")
	}
	if options.DatabaseDriver != "" && *databaseDriver != options.DatabaseDriver {
		return errors.New("Studio catalog driver differs from host configuration")
	}
	db := options.Database
	if db != nil && options.DatabaseDriver == "" {
		return errors.New("caller-owned catalog requires an explicit database driver")
	}
	if db == nil {
		db, err = sql.Open(*databaseDriver, *dsn)
		if err != nil {
			return err
		}
		defer db.Close()
	}
	if err = ensureHostDatabaseSchema(lifecycleCtx, db, *databaseDriver, options.SchemaVerifier); err != nil {
		return err
	}
	definitionReader, err := preview.NewDefinitionReader(db)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := definitionReader.Close(closeCtx); err != nil {
			log.Printf("Studio preview definition reader close: %v", err)
		}
	}()
	connectorReader, err := preview.NewConnectorReader(db)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := connectorReader.Close(closeCtx); err != nil {
			log.Printf("Studio preview connector reader close: %v", err)
		}
	}()
	dynamicPreview := preview.Dynamic{Const: options.Const, ModulePath: options.AuthoringModule, StudioDB: db, RootDir: ".", Definitions: definitionReader, Connectors: connectorReader}
	predicates, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		return err
	}
	dynamicPreview.Types, err = predicates.RuntimeTypes()
	if err != nil {
		return err
	}
	// Validate the same server-owned authorization input types that the dynamic
	// runtime links before staging a published reader.
	if err = accesscontext.RegisterTypes(dynamicPreview.Types); err != nil {
		return err
	}
	runtimeAdmin := options.RuntimeControl
	if runtimeAdmin == nil {
		runtimeAdmin, err = runtimeadmin.New(*dynamicHTTPURL, adminToken, nil)
		if err != nil {
			return err
		}
	}
	cacheInvalidator, _ := runtimeAdmin.(sqltransport.CacheViewInvalidator)

	authorizer, err := authorization.NewSDKAuthorizer(db)
	if err != nil {
		return err
	}
	authorizer.ReleaseUse = options.ReleaseUse
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := authorizer.Close(closeCtx); err != nil {
			log.Printf("Studio authorizer reader close: %v", err)
		}
	}()
	transport := &sqltransport.Transport{AuthoringModule: options.AuthoringModule, DB: db, Authorizer: authorizer, ReleaseNamespace: options.ReleaseNamespace, ReleaseManagementAvailable: options.ReleaseManagementAvailable, Predicates: predicates, Probe: connectivity.SQLProbe{Const: options.Const}, Catalog: connectivity.SQLCatalog{Const: options.Const}, SQLTester: dynamicPreview, Preview: dynamicPreview, ViewTester: dynamicPreview, RelationTester: dynamicPreview, ComposeTester: dynamicPreview, Warmup: dynamicPreview, CacheInvalidator: cacheInvalidator, Validator: dynamicPreview, ContractInspector: dynamicPreview, Activator: runtimeAdmin, RuntimeProbe: runtimeAdmin}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := transport.Close(closeCtx); err != nil {
			log.Printf("Studio predicate reader close: %v", err)
		}
	}()
	var sdkTransport sdk.Transport = transport
	accessConfig := accessprovider.EnvironmentConfig()
	accessConfig.Issuer, accessConfig.Audience = strings.TrimSpace(*accessIssuer), strings.TrimSpace(*accessAudience)
	accessConfig.PublicKeyFile, accessConfig.CertURL = strings.TrimSpace(*accessKey), strings.TrimSpace(*accessCertURL)
	accessConfig.UserInfoURL = strings.TrimSpace(*accessUserInfoURL)
	if options.AccessProviderConfig != nil {
		accessConfig = *options.AccessProviderConfig
	}
	resourceProvider, providerErr := configuredAccessProvider(lifecycleCtx, options, accessConfig)
	if providerErr != nil {
		return providerErr
	}
	if resourceProvider != nil {
		if resolvedMode != string(httptransport.Authenticated) {
			return errors.New("resource ACL requires authenticated Studio mode")
		}
		accessService := &authz.Service{Store: &accessstore.Store{DB: db}, Provider: resourceProvider}
		namespaceReader, err := namespacepolicy.NewReader(db)
		if err != nil {
			return err
		}
		defer namespaceReader.Close(context.Background())
		executionAccess := *accessService
		executionAccess.Store = &namespacepolicy.Store{Store: accessService.Store, Namespace: namespaceReader.Namespace}
		dynamicPreview.Access, dynamicPreview.AccessTenant = &executionAccess, *accessTenant
		transport.Preview, transport.ViewTester, transport.RelationTester, transport.ComposeTester = dynamicPreview, dynamicPreview, dynamicPreview, dynamicPreview
		sdkTransport = &access.Transport{Next: transport, Service: accessService, Catalog: &access.Catalog{Service: accessService, Source: &accesscatalog.Store{DB: db}}}
	}
	mux := http.NewServeMux()
	var nativeBearerPaths []string
	if options.AuthenticatedModules != nil && resolvedMode != string(httptransport.Authenticated) {
		return errors.New("authenticated modules require authenticated mode")
	}
	var mountedModules []Module
	defer func() {
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		result = errors.Join(result, closeModules(shutdown, mountedModules))
	}()
	if options.Modules != nil {
		if resolvedMode != string(httptransport.Authenticated) || resourceProvider == nil {
			return errors.New("Studio extension modules require authenticated mode and a trusted identity provider")
		}
		modules, moduleErr := options.Modules(lifecycleCtx, db, resourceProvider)
		if moduleErr != nil {
			return moduleErr
		}
		mountedModules = append(mountedModules, modules...)
		if moduleErr = mountModules(mux, modules); moduleErr != nil {
			return moduleErr
		}
	}
	if *staticRoot != "" {
		assets, assetErr := newStaticAssets(*staticRoot)
		if assetErr != nil {
			return assetErr
		}
		defer assets.Close()
		mux.Handle("/", assets)
	} else if options.StaticFS != nil {
		assets, assetErr := newStaticFS(options.StaticFS)
		if assetErr != nil {
			return assetErr
		}
		mux.Handle("/", assets)
	}
	var extensionProxy http.Handler
	gatewayConfig := httptransport.Config{Mode: httptransport.Development, DevelopmentSubject: *subject, RequireNamespace: true}
	if resolvedMode == string(httptransport.Development) {
		devJWT, jwtErr := httptransport.NewDevelopmentJWT("studio-development", "studio-sdk")
		if jwtErr != nil {
			return jwtErr
		}
		gatewayConfig.DevelopmentCredential = devJWT.Credential
		publicKey, keyErr := devJWT.PublicKeyPEM()
		if keyErr != nil {
			return keyErr
		}
		cloneHost, cloneErr := versionclone.NewLocalHost(lifecycleCtx, ".", *dsn, publicKey)
		if cloneErr != nil {
			return cloneErr
		}
		defer cloneHost.Shutdown(context.Background())
		sdkTransport = &versionclone.Transport{Next: sdkTransport, Invoker: cloneHost}
		transport.SystemCredentialProvider = func(ctx context.Context) (sdk.VerifiedCredential, error) {
			return devJWT.Credential(ctx, sdk.SystemPrincipal().Subject)
		}
	}
	if resolvedMode == string(httptransport.Authenticated) {
		verification, authConfigErr := configuredJWTVerification(options.JWTValidator, *jwtCertURL, *jwtPublicKey)
		if authConfigErr != nil {
			return authConfigErr
		}
		if options.JWTValidator != nil && *jwtAudience == "" && loginOAuth != nil {
			*jwtAudience = loginOAuth.ClientID
		}
		issuer, audience, key, authConfigErr := resolveAuthenticatedIdentity(*jwtIssuer, *jwtAudience, *sessionKey, options.JWTValidator != nil)
		if authConfigErr != nil {
			return authConfigErr
		}
		if *loginIDToken && audience != loginOAuth.ClientID {
			return errors.New("identity-token login requires -jwt-audience to equal -login-client-id")
		}
		jwtVerifier := verifier.New(verification)
		if err = jwtVerifier.Init(lifecycleCtx); err != nil {
			return err
		}
		sessionStore, sessionErr := bffauth.NewSQLStore(db, key)
		if sessionErr != nil {
			return sessionErr
		}
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := sessionStore.Close(closeCtx); err != nil {
				log.Printf("Studio BFF session runtime close: %v", err)
			}
		}()
		if *sessionPruneInterval > 0 {
			prunerDone := startSessionPruner(lifecycleCtx, sessionStore, *sessionPruneInterval)
			defer func() { stop(); <-prunerDone }()
		}
		sessions, sessionErr := bffauth.New(bffauth.Config{CookieName: *cookieName, Secure: secureSessionCookies(origin), Issuer: issuer, Audience: audience, Store: sessionStore}, strictClaimsVerifier{jwtVerifier})
		if sessionErr != nil {
			return sessionErr
		}
		if !*loginIDToken {
			sessions.Register(mux)
			staticTarget, parseErr := url.Parse(*staticDatlyURL)
			if parseErr != nil {
				return parseErr
			}
			nativeSDK, proxyErr := sessions.ProxyWithResolver(staticTarget, "/", nativeSDKTarget(staticTarget))
			if proxyErr != nil {
				return proxyErr
			}
			for _, path := range nativeSDKPaths {
				mux.Handle(path, nativeSDK)
			}
		}
		if loginOAuth != nil {
			login, loginErr := bffauth.NewLogin(bffauth.LoginConfig{OAuth: *loginOAuth, CookieKey: key, Secure: secureSessionCookies(origin), UseIDToken: *loginIDToken, AllowedOrigin: origin}, sessions)
			if loginErr != nil {
				return loginErr
			}
			if *loginIDToken {
				browserAuth, authErr := login.DatlyHTTP()
				if authErr != nil {
					return authErr
				}
				for _, route := range []string{"GET /v1/studio/auth/login", "GET /v1/studio/auth/callback", "POST /v1/studio/auth/token", "DELETE /v1/studio/auth/session"} {
					mux.Handle(route, browserAuth)
				}
			} else {
				if err := login.Register(mux); err != nil {
					return err
				}
			}
		}
		if !*loginIDToken {
			for _, proxyConfig := range []struct{ mount, target string }{{"/v1/studio/runtime/", *dynamicHTTPURL}, {"/v1/studio/mcp/", *dynamicMCPURL}, {"/v1/studio/sdk-mcp/", *staticMCPURL}} {
				target, parseErr := url.Parse(proxyConfig.target)
				if parseErr != nil {
					return parseErr
				}
				var resolver bffauth.ProxyTargetResolver
				if proxyConfig.mount == "/v1/studio/mcp/" {
					resolver = namespaceMCPTarget(sdkTransport)
				}
				proxy, proxyErr := sessions.ProxyWithResolver(target, strings.TrimSuffix(proxyConfig.mount, "/"), resolver)
				if proxyConfig.mount == "/v1/studio/sdk-mcp/" {
					proxy, proxyErr = sessions.ProxyNativeMCP(target, strings.TrimSuffix(proxyConfig.mount, "/"))
					nativeBearerPaths = append(nativeBearerPaths, proxyConfig.mount+"mcp")
				}
				if proxyErr != nil {
					return proxyErr
				}
				mux.Handle(proxyConfig.mount, proxy)
			}
		}
		if extensionConfig != nil {
			var proxyErr error
			extensionProxy, proxyErr = newExtensionProxy(sessions, extensionConfig)
			if proxyErr != nil {
				return proxyErr
			}
		}
		if options.AuthenticatedModules != nil {
			if resourceProvider == nil {
				return errors.New("authenticated modules require resource identity")
			}
			modules, moduleErr := options.AuthenticatedModules(lifecycleCtx, db, resourceProvider, sessions)
			if moduleErr != nil {
				return moduleErr
			}
			mountedModules = append(mountedModules, modules...)
			if moduleErr = mountModules(mux, modules); moduleErr != nil {
				return moduleErr
			}
		}
		gatewayConfig = httptransport.Config{Mode: httptransport.Authenticated, Authenticator: sessions, RequireNamespace: true}
	} else {
		target, parseErr := url.Parse(*dynamicMCPURL)
		if parseErr != nil {
			return parseErr
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ModifyResponse = stripProxyCORS
		proxy.ErrorHandler = mcpProxyError
		original := proxy.Director
		proxy.Director = func(request *http.Request) {
			original(request)
			request.URL.Path = "/" + strings.TrimPrefix(strings.TrimPrefix(request.URL.Path, "/v1/studio/mcp"), "/")
		}
		mux.Handle("/v1/studio/mcp/", http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			path := "/" + strings.TrimPrefix(strings.TrimPrefix(request.URL.Path, "/v1/studio/mcp"), "/")
			if path == "/_studio" || strings.HasPrefix(path, "/_studio/") {
				http.NotFound(response, request)
				return
			}
			proxy.ServeHTTP(response, request)
		}))
	}
	// Authenticated SDK requests have only the selected Datly component routes
	// mounted above. Keep the generic gateway for explicit local development
	// until its static-host credential handoff is available.
	if resolvedMode == string(httptransport.Development) {
		gateway := httptransport.Gateway{Config: gatewayConfig, Transport: sdkTransport}
		mux.Handle(httptransport.PathPrefix, gateway)
	}
	log.Printf("Studio SDK %s host listening on http://%s", resolvedMode, *address)
	handler, err := configuredHTTPMiddleware(options.HTTPMiddleware, requestIDs(corsModules(origin, resolvedMode == string(httptransport.Authenticated), mountedModules, noStore(routeExtensionProxy(mux, extensionProxy)), nativeBearerPaths...)))
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: *address, Handler: explicitExecutionDeadline(handler),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 2 * time.Minute,
	}
	listener := options.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", *address)
		if err != nil {
			return err
		}
	}
	if options.Ready != nil {
		options.Ready(listener.Addr())
	}
	if err := serveUntilStopped(lifecycleCtx, server, listener); err != nil {
		return err
	}
	return nil
}

func configuredAccessProvider(ctx context.Context, options Options, config accessprovider.Config) (authz.Provider, error) {
	factory := options.AccessProviderFactory
	if factory == nil {
		factory = accessprovider.RegisteredEnvironmentFactory()
	}
	configured := factory != nil || config.Issuer != "" || config.Audience != "" || config.PublicKeyFile != "" || config.CertURL != "" || config.UserInfoURL != ""
	if !configured {
		return nil, nil
	}
	return accessprovider.New(ctx, config, factory)
}

func mcpProxyError(response http.ResponseWriter, request *http.Request, err error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(response).Encode(map[string]string{"code": "mcp_unavailable", "message": "The dedicated Datly MCP server is unavailable. Start or reconnect the dynamic runtime, then retry."})
}

// stripProxyCORS keeps CORS policy at the Studio boundary. Datly's own MCP
// transport headers are correct for direct access but must not be combined
// with the BFF headers on browser requests.
func stripProxyCORS(response *http.Response) error {
	for _, header := range []string{
		"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers", "Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers", "Access-Control-Max-Age",
	} {
		response.Header.Del(header)
	}
	return nil
}

func resolveAuthenticatedConfig(certURL, issuer, audience, encodedKey string, publicKeys ...string) (string, string, []byte, error) {
	publicKey := ""
	if len(publicKeys) > 0 {
		publicKey = publicKeys[0]
	}
	if _, err := jwtVerificationConfig(certURL, publicKey); err != nil {
		return "", "", nil, err
	}
	return resolveAuthenticatedIdentity(issuer, audience, encodedKey)
}

func resolveAuthenticatedIdentity(issuer, audience, encodedKey string, shared ...bool) (string, string, []byte, error) {
	optionalRestrictions := len(shared) > 0 && shared[0]
	issuer = strings.TrimSpace(issuer)
	audience = strings.TrimSpace(audience)
	if issuer == "" && !optionalRestrictions {
		return "", "", nil, errors.New("-jwt-issuer or STUDIO_JWT_ISSUER is required in authenticated mode")
	}
	if audience == "" && !optionalRestrictions {
		return "", "", nil, errors.New("-jwt-audience or STUDIO_JWT_AUDIENCE is required in authenticated mode")
	}
	encodedKey = strings.TrimSpace(encodedKey)
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		key, err = base64.RawURLEncoding.DecodeString(encodedKey)
	}
	if err != nil || len(key) != 32 {
		return "", "", nil, errors.New("-session-key or STUDIO_SESSION_KEY must be a base64-encoded 32-byte key")
	}
	return issuer, audience, key, nil
}

func ensureSchema(ctx context.Context, db *sql.DB) error { return schema.Initialize(ctx, db, "sqlite") }

const localRuntimeAdminToken = "datly-studio-local-runtime"

func validateGatewayMode(mode string) error {
	if mode != string(httptransport.Development) && mode != string(httptransport.Authenticated) {
		return fmt.Errorf("unsupported gateway mode %q", mode)
	}
	return nil
}

func resolveRuntimeAdminToken(mode, value string) (string, error) {
	value = strings.TrimSpace(value)
	if mode == string(httptransport.Development) && value == "" {
		return localRuntimeAdminToken, nil
	}
	if value == "" {
		return "", errors.New("-dynamic-admin-token or STUDIO_RUNTIME_ADMIN_TOKEN is required in authenticated mode")
	}
	if mode == string(httptransport.Authenticated) && value == localRuntimeAdminToken {
		return "", errors.New("authenticated mode rejects the known local runtime admin token")
	}
	return value, nil
}

func resolveAllowedOrigin(mode, value string) (string, error) {
	value = strings.TrimSpace(value)
	if mode == string(httptransport.Development) && value == "" {
		value = "http://127.0.0.1:5173"
	}
	if value == "" {
		return "", errors.New("-allowed-origin or STUDIO_ALLOWED_ORIGIN is required in authenticated mode")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid allowed origin %q", value)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func cors(allowedOrigin string, requireUnsafeOrigin bool, next http.Handler) http.Handler {
	return corsWithOriginException(allowedOrigin, requireUnsafeOrigin, nil, next)
}

func corsWithOriginException(allowedOrigin string, requireUnsafeOrigin bool, allowMissing func(*http.Request) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && origin != allowedOrigin {
			http.Error(w, "origin is not allowed", http.StatusForbidden)
			return
		}
		if requireUnsafeOrigin && origin == "" && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && (allowMissing == nil || !allowMissing(r)) {
			http.Error(w, "origin is required", http.StatusForbidden)
			return
		}
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Studio-Namespace, X-Studio-Development-Subject, X-Request-ID, Mcp-Protocol-Version, Mcp-Method, Mcp-Name, Mcp-Session-Id, Last-Event-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if r.Method == http.MethodOptions {
			if origin == "" {
				http.Error(w, "preflight origin is required", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func requestIDs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := make([]byte, 16)
		if _, err := rand.Read(value); err != nil {
			http.Error(w, "request correlation is unavailable", http.StatusInternalServerError)
			return
		}
		requestID := hex.EncodeToString(value)
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}
