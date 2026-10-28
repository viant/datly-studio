// Package studiohost composes Studio authoring, browser APIs and persisted
// namespace runtimes in one process. Private applications reuse this host.
package studiohost

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viant/authz"
	studio "github.com/viant/datly-studio"
	"github.com/viant/datly-studio/app/datlycmd"
	"github.com/viant/datly-studio/app/hostconfig"
	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/datly-studio/audit"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/runtime/accessprovider"
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	studiocatalog "github.com/viant/datly-studio/studio/catalog"
	authoring "github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/bootstrap/connector"
	nativeauth "github.com/viant/datly/runtime/auth"
	datlyconfig "github.com/viant/datly/standalone/config"
	"github.com/viant/mcp-protocol/authorization"
	oauthmeta "github.com/viant/mcp-protocol/oauth2/meta"
	"github.com/viant/scy"
)

// Endpoints contains actual bound addresses, including OS-allocated ports.
type Endpoints struct{ HTTP, AdminMCP, NamespaceMCP, NativeHTTP string }

// Options are trusted process bindings. Namespaces and application definitions
// are never accepted here from browser or agent requests.
type Options struct {
	// Brand is the linked application name; the separate Data Studio entry retains its own name.
	Brand   string
	Native  datlycmd.Options
	Browser studioapi.Options
	// ConfigureRuntime installs linked extensions before namespace startup.
	ConfigureRuntime func(*runtimehost.Config) error
	// ConfigureBrowser binds app extensions after internal listeners are ready,
	// before constructing or serving the browser API. Addresses are process-owned.
	ConfigureBrowser func(*studioapi.Options, Endpoints, *runtimehost.NamespaceGroup) error
	Ready            func(Endpoints)
	Diagnostics      io.Writer
}

// Serve owns every service until shutdown. It starts no child processes and
// creates no generated configuration files. Internal addresses and admin
// credentials are process-owned, rather than operator-supplied wiring.
// The caller must not mutate config or its nested values after startup begins.
func Serve(parent context.Context, config *hostconfig.Config, options Options) error {
	if parent == nil {
		return errors.New("Studio host context is required")
	}
	if err := parent.Err(); err != nil {
		return err
	}
	if err := config.ValidateAuth(); err != nil {
		return err
	}
	if err := config.ValidateListeners(); err != nil {
		return err
	}
	catalog, err := config.CatalogConnector()
	if err != nil {
		return err
	}
	if catalog.Driver != "sqlite" && catalog.Driver != "mysql" {
		return errors.New("Studio storage driver must be sqlite or mysql")
	}
	if config.JWTValidator == nil {
		return errors.New("Studio host requires JWTValidator")
	}
	issuer, audience := "", ""
	if config.JWTClaims != nil {
		issuer, audience = config.JWTClaims.Issuer, config.JWTClaims.Audience
	}
	policyTenant := config.Tenant
	if policyTenant == "" {
		policyTenant = "studio"
	}
	if options.Browser.Database != nil || options.Browser.Listener != nil || options.Browser.JWTValidator != nil {
		return errors.New("Studio host owns browser storage, listener and JWT configuration")
	}
	if options.Diagnostics == nil {
		options.Diagnostics = io.Discard
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	// Resolve the one storage secret through the same SCY contract as Datly.
	if catalog.Secret != nil {
		secret, loadErr := scy.New().Load(ctx, catalog.Secret)
		if loadErr != nil {
			return errors.New("Studio storage secret could not be loaded")
		}
		catalog.DSN = secret.Expand(catalog.DSN)
		catalog.Secret = nil
	}
	if catalog.Driver == "sqlite" {
		catalog.DSN, err = prepareSQLiteStorage(catalog.DSN)
		if err != nil {
			return err
		}
	}
	storage, err := connector.Open(ctx, []connector.Config{catalog}, catalog.Name)
	if err != nil {
		return err
	}
	defer storage.Close()
	db, err := storage.ResolveDB(ctx, catalog.Name)
	if err != nil {
		return err
	}
	if options.Browser.SchemaVerifier != nil {
		if err := options.Browser.SchemaVerifier(ctx, db); err != nil {
			return err
		}
	}

	authorityConfig := accessprovider.Config{Issuer: issuer, Audience: audience, JWTValidator: config.JWTValidator, Tenant: policyTenant}
	var provider authz.Provider
	if config.Identity != nil && options.Native.AccessProviderFactory == nil {
		providers, resolveErr := accessprovider.HTTPProviders(ctx, authorityConfig, *config.Identity, config.EntityEvaluation)
		provider, err = providers.Identity, resolveErr
	} else {
		provider, err = accessprovider.New(ctx, authorityConfig, options.Native.AccessProviderFactory)
	}

	if err != nil {
		return err
	}
	if provider == nil {
		return errors.New("Studio host identity provider is required")
	}
	viewerRole := "AUDIT_VIEWER"
	if config.Audit != nil && config.Audit.ViewerRole != "" {
		viewerRole = config.Audit.ViewerRole
	}
	if viewerRole == "*" || strings.TrimSpace(viewerRole) != viewerRole {
		return errors.New("Audit.ViewerRole must be an exact role name")
	}
	auditService := &audit.Service{DB: db, Provider: provider, ViewerRole: viewerRole}
	defer auditService.Close(context.Background())
	auditService.Logger = audit.New(audit.Config{Enabled: true, Sanitize: audit.RedactJSON, Identity: auditService.Identity, Policy: auditService.Policy,
		OnError: func(error) { fmt.Fprintln(options.Diagnostics, "MCP audit persistence failed") }, OnDrop: func() { fmt.Fprintln(options.Diagnostics, "MCP audit queue full") }}, &audit.SQLSink{DB: db})
	defer func() {
		drain, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = auditService.Logger.Close(drain)
	}()
	factory := func(context.Context, accessprovider.Config) (authz.Provider, error) { return provider, nil }
	options.Native.AccessProviderFactory = factory
	if options.Native.Authoring == nil {
		options.Native.Authoring = &authoring.AuthoringConfig{}
	}
	authoringConfig := *options.Native.Authoring
	authoringConfig.Const = config.Const
	if authoringConfig.ModulePath == "" {
		authoringConfig.ModulePath = authoring.DefaultAuthoringModule
	}
	if authoringConfig.ReaderCaches == nil {
		authoringConfig.ReaderCaches = config.Caches
	}
	options.Native.Authoring = &authoringConfig
	options.Browser.AuthoringModule = authoringConfig.ModulePath
	publicAddress, err := config.Endpoint.ListenAddress()
	if err != nil {
		return err
	}
	publicListener, err := net.Listen("tcp", publicAddress)
	if err != nil {
		return err
	}
	defer publicListener.Close()
	origin, err := browserOrigin(config.PublicURL, publicListener.Addr().String())
	if err != nil {
		return err
	}
	if err := studiocatalog.ConfigureManagementNavigation(nil, origin, "datastudio"); err != nil {
		return err
	}
	nativeConfig := config.Config
	nativeConfig.Endpoint = datlyconfig.Endpoint{Address: "127.0.0.1:0"}
	adminMCP := *config.MCP
	if adminMCP.Authorization == nil {
		// A protected policy also carries the verified bearer into native MCP input
		// binding. Reuse the browser SDK MCP mount as the stable resource URI.
		adminMCP.Authorization = &authorization.Policy{Global: &authorization.Authorization{
			ProtectedResourceMetadata: &oauthmeta.ProtectedResourceMetadata{Resource: origin + "/v1/studio/sdk-mcp/mcp", AuthorizationServers: optionalIssuer(issuer)},
		}}
	}
	nativeConfig.MCP = &adminMCP
	// Linked Studio contracts share this storage connection through standard aliases.
	catalog.Name = "studio"
	nativeConfig.Connector = "studio"
	nativeConfig.Connectors = []connector.Config{catalog, {Name: "authz", AliasOf: "studio"}}
	if nativeConfig.GoBootstrap == nil {
		packages, err := studio.LinkedComponentPackages()
		if err != nil {
			return err
		}
		nativeConfig.GoBootstrap = &datlyconfig.Packages{Packages: packages, LinkedOnly: true}
	}
	var processGroup atomic.Pointer[runtimehost.NamespaceGroup]
	control := inProcessControl(&processGroup)
	nativeMiddleware := options.Native.RequestMiddleware
	options.Native.RequestMiddleware = func(next http.Handler) http.Handler {
		if nativeMiddleware != nil {
			next = nativeMiddleware(next)
		}
		next = auditService.Logger.Wrap(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(audit.WithService(runtimeadmin.WithClient(r.Context(), control), auditService)))
		})
	}
	native, err := datlycmd.New(ctx, &nativeConfig, options.Native, options.Diagnostics)
	if err != nil {
		return err
	}
	nativeDone := make(chan struct{})
	var nativeErr error
	go func() { nativeErr = native.Serve(ctx, options.Diagnostics); close(nativeDone) }()
	defer func() {
		cancel()
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = native.Shutdown(shutdown)
		<-nativeDone
	}()
	ready, stop := context.WithTimeout(ctx, 30*time.Second)
	addresses, err := native.WaitReady(ready)
	stop()
	if err != nil {
		return err
	}
	if len(addresses) != 2 {
		return errors.New("Studio native host did not publish HTTP and admin MCP listeners")
	}
	namespaceAddress, err := config.NamespaceMCP.ListenAddress()
	if err != nil {
		return err
	}

	tenant := config.Tenant
	if tenant == "" {
		tenant = "studio"
	}
	// Declared reader JwtClaim inputs must use the same deployment-owned
	// verifier and claim policy as the management host. The access provider
	// authenticates transport access; it does not configure reader codecs.
	readerJWT, err := nativeauth.New(ctx, &nativeauth.Config{JWTValidator: config.JWTValidator, ClaimPolicy: config.JWTClaims})
	if err != nil {
		return fmt.Errorf("configure namespace reader JWT verification: %w", err)
	}
	runtimeConfig := runtimehost.Config{MCPToolNames: config.MCPToolNames, ToolCallAudit: auditService.Logger, Const: config.Const, InProcess: true, RootDir: config.BaseDir, ModulePath: authoringConfig.ModulePath,
		CodecFactory:     studiocodecs.New(readerJWT),
		NamespaceMCPMode: "shared", HTTP: runtimehost.Listener{Address: "127.0.0.1:0"}, MCP: runtimehost.Listener{Address: namespaceAddress},
		Studio:         runtimehost.Studio{Driver: catalog.Driver, DSN: catalog.DSN},
		Authentication: runtimehost.Authentication{DefaultMode: "required", Issuer: issuer, Audience: audience, ConnectionExport: config.NamespaceConnectionExport, Providers: config.NamespaceOAuthProviders},
		Access:         &runtimehost.ResourceAccessConfig{Tenant: tenant, Provider: provider}, ReaderCaches: config.Caches}
	// PublicURL already defines the trusted reverse-proxy origin for this host.
	// Local deployments without it retain the namespace listener's loopback URL.
	if config.PublicURL != "" {
		runtimeConfig.Authentication.PublicMCPURL = origin
	}
	if options.ConfigureRuntime != nil {
		if err := options.ConfigureRuntime(&runtimeConfig); err != nil {
			return err
		}
	}
	group, err := runtimehost.NewNamespaceGroup(ctx, runtimeConfig)
	if err != nil {
		return err
	}
	defer func() {
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = group.Close(shutdown)
	}()
	processGroup.Store(group)

	endpoints := Endpoints{HTTP: publicListener.Addr().String(), AdminMCP: addresses[1], NamespaceMCP: group.Listeners.SharedAddress(), NativeHTTP: addresses[0]}
	browser := options.Browser
	browser.Const = config.Const
	browser.RuntimeControl = control
	if config.Login != nil {
		login, err := config.Login.Resolve(ctx)
		if err != nil {
			return err
		}
		browser.Login = &studioapi.Login{AuthorizationURL: login.Endpoint.AuthURL, TokenURL: login.Endpoint.TokenURL, ClientID: login.ClientID, ClientSecret: login.ClientSecret, RedirectURL: origin + "/v1/studio/auth/callback", Scopes: login.Scopes}
	}

	if config.Auth != nil && config.Auth.Enabled {
		login, err := config.Auth.Resolve(ctx, origin)
		if err != nil {
			return err
		}
		browser.Login = &studioapi.Login{AuthorizationURL: login.Endpoint.AuthURL, TokenURL: login.Endpoint.TokenURL, ClientID: login.ClientID, ClientSecret: login.ClientSecret, RedirectURL: origin + "/v1/studio/auth/callback", Scopes: login.Scopes}
	}
	if options.ConfigureBrowser != nil {
		if err := options.ConfigureBrowser(&browser, endpoints, group); err != nil {
			return err
		}
	}
	sessionKey, err := browserSessionKey(ctx, config.SessionKey, browser.Login, origin)
	if err != nil {
		return err
	}

	if browser.SchemaVerifier == nil {
		// Database migrations are operator-owned; startup performs no schema inspection.
		browser.SchemaVerifier = func(context.Context, *sql.DB) error { return nil }
	}
	browser.Database = db
	browser.DatabaseDriver = catalog.Driver
	browser.Listener = publicListener
	priorHTTPMiddleware := browser.HTTPMiddleware
	browser.HTTPMiddleware = func(next http.Handler) http.Handler {
		settings := map[string]any{"mode": "bff"}
		if config.Auth != nil {
			settings["enabled"] = config.Auth.Enabled
			if config.Auth.Enabled {
				settings["label"] = config.Auth.OAuth.Label
				settings["redirectSameTab"] = true
			}
		}
		handler := browserConfigHandler(origin, options.Brand, next, settings)
		if priorHTTPMiddleware != nil {
			return priorHTTPMiddleware(handler)
		}
		return handler
	}
	browser.JWTValidator = config.JWTValidator
	browser.AccessProviderConfig = &authorityConfig
	browser.AccessProviderFactory = factory
	browserReady := make(chan struct{})
	priorReady := browser.Ready
	browser.Ready = func(address net.Addr) {
		if priorReady != nil {
			priorReady(address)
		}
		close(browserReady)
	}
	args := []string{"--address", publicListener.Addr().String(), "--driver", catalog.Driver, "--dsn", catalog.DSN, "--mode", "authenticated",
		"--jwt-issuer", issuer, "--jwt-audience", audience, "--session-key", sessionKey,
		"--allowed-origin", origin, "--access-tenant", tenant,
		"--dynamic-mcp-url", "http://" + group.Listeners.SharedAddress(),
		"--static-datly-url", "http://" + addresses[0], "--static-mcp-url", "http://" + addresses[1]}
	if config.Auth != nil && config.Auth.CookieName != "" {
		args = append(args, "--session-cookie", config.Auth.CookieName)
	}
	if config.StaticRoot != "" {
		args = append(args, "--static-root", config.StaticRoot)
	}
	browserDone := make(chan error, 1)
	go func() { browserDone <- studioapi.Serve(ctx, args, browser) }()
	select {
	case err := <-browserDone:
		return err
	case <-nativeDone:
		cancel()
		<-browserDone
		return errors.Join(errors.New("native Studio stopped before browser readiness"), nativeErr)
	case <-browserReady:
	case <-ctx.Done():
		cancel()
		return <-browserDone
	}
	// Private modules have bound their in-process dependencies at browser readiness.
	if changes, err := group.Reconcile(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		isolated := false
		for _, change := range changes {
			if change.Status == "failed" && change.NamespaceID != "" {
				isolated = true
				break
			}
		}
		// No per-namespace results means the persisted snapshot or global
		// discovery failed. Keep that terminal; only individual restores retry.
		if !isolated {
			return err
		}
		fmt.Fprintln(options.Diagnostics, "namespace reconciliation:", err)
	}
	namespaceDone := make(chan error, 1)
	go func() {
		previous := ""
		namespaceDone <- group.Run(ctx, time.Second, func(_ []namespacemcp.Change, err error) {
			message := ""
			if err != nil {
				message = err.Error()
			}
			if message != "" && message != previous {
				fmt.Fprintln(options.Diagnostics, "namespace reconciliation:", message)
			}
			previous = message
		})
	}()
	defer func() { cancel(); <-namespaceDone }()
	if options.Ready != nil {
		options.Ready(endpoints)
	}
	select {
	case err := <-browserDone:
		return err
	case <-nativeDone:
		cancel()
		<-browserDone
		return errors.Join(errors.New("native Studio stopped unexpectedly"), nativeErr)
	case <-ctx.Done():
		cancel()
		return <-browserDone
	}
}

func randomKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", errors.New("Studio host random key generation failed")
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
func browserOrigin(explicit, address string) (string, error) {
	if explicit == "" {
		hostname, port, err := net.SplitHostPort(address)
		if err != nil {
			return "", err
		}
		if ip := net.ParseIP(hostname); ip != nil && ip.IsUnspecified() {
			hostname = "localhost"
		}
		explicit = "http://" + net.JoinHostPort(hostname, port)
	}
	parsed, err := url.Parse(explicit)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return "", errors.New("Studio PublicURL must be an HTTP origin")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || net.ParseIP(parsed.Hostname()).IsLoopback())) {
		return "", fmt.Errorf("Studio PublicURL requires HTTPS or loopback HTTP")
	}
	return explicit, nil
}

// Browser configuration is derived from the application's one public origin.
// Operators do not maintain a separate client configuration file.
func browserConfigHandler(origin, brand string, next http.Handler, settings ...map[string]any) http.Handler {
	authentication := map[string]any{"mode": "bff"}
	if len(settings) > 0 {
		authentication = settings[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/studio-config.json" && r.URL.Path != "/datly-studio-config.json" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodHead {
			return
		}
		name := brand
		if name == "" || r.URL.Path == "/datly-studio-config.json" {
			name = "Data Studio"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"brand": name, "mode": "authenticated", "apiBaseURL": origin, "authentication": authentication})
	})
}

func optionalIssuer(issuer string) []string {
	if issuer == "" {
		return nil
	}
	return []string{issuer}
}
