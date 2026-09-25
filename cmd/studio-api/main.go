// studio-api starts the local Studio SDK host used by the Forge UI.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	jwtlib "github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	_ "github.com/viant/bigquery"
	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/access"
	accessoauth "github.com/viant/datly-studio/sdk/access/oauth"
	"github.com/viant/datly-studio/sdk/connectivity"
	"github.com/viant/datly-studio/sdk/httptransport"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	accessstore "github.com/viant/datly-studio/store/sql/access"
	"github.com/viant/datly-studio/store/sql/migrate"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/scy/auth/jwt/verifier"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/pg"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

func main() {
	address := flag.String("address", "127.0.0.1:8080", "SDK HTTP listen address")
	dsn := flag.String("dsn", "file:.data/studio.db?cache=shared", "Studio SQLite DSN")
	subject := flag.String("subject", "awitas", "local development principal")
	mode := flag.String("mode", "development", "gateway mode: development or authenticated")
	cookieName := flag.String("session-cookie", bffauth.DefaultCookieName, "authenticated BFF session cookie name")
	jwtCertURL := flag.String("jwt-cert-url", "", "authenticated JWT certificate URL")
	jwtIssuer := flag.String("jwt-issuer", os.Getenv("STUDIO_JWT_ISSUER"), "authenticated JWT issuer")
	jwtAudience := flag.String("jwt-audience", os.Getenv("STUDIO_JWT_AUDIENCE"), "authenticated JWT audience")
	sessionKey := flag.String("session-key", os.Getenv("STUDIO_SESSION_KEY"), "base64-encoded 32-byte BFF session encryption key")
	dynamicHTTPURL := flag.String("dynamic-http-url", "http://127.0.0.1:8082", "dynamic Datly HTTP target")
	dynamicMCPURL := flag.String("dynamic-mcp-url", "http://127.0.0.1:8091", "dynamic Datly MCP target")
	extensionBackendURL := flag.String("extension-backend-url", "", "trusted extension backend origin (authenticated mode only)")
	extensionUpstreamPrefix := flag.String("extension-upstream-prefix", "", "allowlisted extension upstream path prefix, for example /api/widgets")
	dynamicAdminToken := flag.String("dynamic-admin-token", os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN"), "dynamic runtime reload token")
	allowedOrigin := flag.String("allowed-origin", os.Getenv("STUDIO_ALLOWED_ORIGIN"), "exact Studio browser origin")
	staticRoot := flag.String("static-root", "", "optional absolute directory of public built UI assets served on the Studio BFF origin")
	sessionPruneInterval := flag.Duration("session-prune-interval", time.Minute, "how often to prune bounded batches of expired BFF sessions; 0 disables background cleanup")
	loginAuthURL := flag.String("login-auth-url", "", "optional OAuth authorization endpoint for the BFF login route")
	loginTokenURL := flag.String("login-token-url", "", "OAuth token endpoint paired with -login-auth-url")
	loginClientID := flag.String("login-client-id", "", "registered OAuth client ID for BFF login")
	loginClientSecretFile := flag.String("login-client-secret-file", "", "optional file containing OAuth client secret; omit for a public PKCE client")
	loginRedirectURL := flag.String("login-redirect-url", "", "exact public callback URL ending in /v1/studio/auth/callback")
	loginScopes := flag.String("login-scopes", "openid,profile,email", "comma-separated OAuth scopes for BFF login, including openid")
	accessIssuer := flag.String("access-issuer", "", "dedicated ACL token issuer")
	accessAudience := flag.String("access-audience", "", "dedicated ACL token audience")
	accessKey := flag.String("access-public-key", "", "ACL issuer RSA public key PEM")
	flag.Parse()
	if *sessionPruneInterval < 0 {
		log.Fatal("-session-prune-interval must not be negative")
	}
	lifecycleCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	resolvedMode := strings.ToLower(strings.TrimSpace(*mode))
	if err := validateGatewayMode(resolvedMode); err != nil {
		log.Fatal(err)
	}
	extensionConfig, err := resolveExtensionProxyConfig(resolvedMode, *extensionBackendURL, *extensionUpstreamPrefix)
	if err != nil {
		log.Fatal(err)
	}
	adminToken, err := resolveRuntimeAdminToken(resolvedMode, *dynamicAdminToken)
	if err != nil {
		log.Fatal(err)
	}
	origin, err := resolveAllowedOrigin(resolvedMode, *allowedOrigin)
	if err != nil {
		log.Fatal(err)
	}
	var loginSecret string
	if *loginClientSecretFile != "" {
		secretBytes, readErr := os.ReadFile(*loginClientSecretFile)
		if readErr != nil {
			log.Fatal(readErr)
		}
		loginSecret = strings.TrimSpace(string(secretBytes))
		if loginSecret == "" {
			log.Fatal("-login-client-secret-file is empty")
		}
	}
	loginOAuth, err := resolveLoginConfig(resolvedMode, origin, *loginAuthURL, *loginTokenURL, *loginClientID, loginSecret, *loginRedirectURL, *loginScopes)
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = ensureSchema(context.Background(), db); err != nil {
		log.Fatal(err)
	}
	definitionReader, err := preview.NewDefinitionReader(db)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := connectorReader.Close(closeCtx); err != nil {
			log.Printf("Studio preview connector reader close: %v", err)
		}
	}()
	dynamicPreview := preview.Dynamic{StudioDB: db, RootDir: ".", Definitions: definitionReader, Connectors: connectorReader}
	predicates, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		log.Fatal(err)
	}
	dynamicPreview.Types, err = predicates.RuntimeTypes()
	if err != nil {
		log.Fatal(err)
	}
	activateRuntime := sqltransport.RuntimeActivatorFunc(func(ctx context.Context, generation int64) error {
		payload, err := json.Marshal(map[string]int64{"generation": generation})
		if err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(*dynamicHTTPURL, "/")+"/_studio/reload", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Studio-Runtime-Token", adminToken)
		response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("dynamic runtime reload returned %s", response.Status)
		}
		return nil
	})
	authorizer, err := authorization.NewSDKAuthorizer(db)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := authorizer.Close(closeCtx); err != nil {
			log.Printf("Studio authorizer reader close: %v", err)
		}
	}()
	transport := &sqltransport.Transport{DB: db, Authorizer: authorizer, Predicates: predicates, Probe: connectivity.SQLProbe{}, Catalog: connectivity.SQLCatalog{}, SQLTester: dynamicPreview, Preview: dynamicPreview, ViewTester: dynamicPreview, RelationTester: dynamicPreview, ComposeTester: dynamicPreview, Warmup: dynamicPreview, Validator: dynamicPreview, Activator: activateRuntime, RuntimeProbe: runtimeProbe{url: strings.TrimSuffix(*dynamicHTTPURL, "/") + "/_studio/status", token: adminToken, client: &http.Client{Timeout: 2 * time.Second}}}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := transport.Close(closeCtx); err != nil {
			log.Printf("Studio predicate reader close: %v", err)
		}
	}()
	var sdkTransport sdk.Transport = transport
	if *accessIssuer != "" || *accessAudience != "" || *accessKey != "" {
		if resolvedMode != string(httptransport.Authenticated) {
			log.Fatal("resource ACL requires authenticated Studio mode")
		}
		pem, keyErr := os.ReadFile(*accessKey)
		if keyErr != nil {
			log.Fatal(keyErr)
		}
		key, keyErr := jwtlib.ParseRSAPublicKeyFromPEM(pem)
		if keyErr != nil {
			log.Fatal(keyErr)
		}
		provider, providerErr := accessoauth.New(accessoauth.Config{Issuer: *accessIssuer, Audience: *accessAudience, Algorithms: []string{"RS256"}, Keyfunc: func(*jwtlib.Token) (any, error) { return key, nil }})
		if providerErr != nil {
			log.Fatal(providerErr)
		}
		sdkTransport = &access.Transport{Next: transport, Service: &access.Service{Store: &accessstore.Store{DB: db}, Provider: provider}}
	}
	mux := http.NewServeMux()
	if *staticRoot != "" {
		assets, assetErr := newStaticAssets(*staticRoot)
		if assetErr != nil {
			log.Fatal(assetErr)
		}
		defer assets.Close()
		mux.Handle("/", assets)
	}
	var extensionProxy http.Handler
	gatewayConfig := httptransport.Config{Mode: httptransport.Development, DevelopmentSubject: *subject}
	if resolvedMode == string(httptransport.Development) {
		devJWT, jwtErr := httptransport.NewDevelopmentJWT("studio-development", "studio-sdk")
		if jwtErr != nil {
			log.Fatal(jwtErr)
		}
		gatewayConfig.DevelopmentCredential = devJWT.Credential
		transport.SystemCredentialProvider = func(ctx context.Context) (sdk.VerifiedCredential, error) {
			return devJWT.Credential(ctx, sdk.SystemPrincipal().Subject)
		}
	}
	if resolvedMode == string(httptransport.Authenticated) {
		issuer, audience, key, authConfigErr := resolveAuthenticatedConfig(*jwtCertURL, *jwtIssuer, *jwtAudience, *sessionKey)
		if authConfigErr != nil {
			log.Fatal(authConfigErr)
		}
		jwtVerifier := verifier.New(&verifier.Config{CertURL: strings.TrimSpace(*jwtCertURL)})
		if err = jwtVerifier.Init(context.Background()); err != nil {
			log.Fatal(err)
		}
		sessionStore, sessionErr := bffauth.NewSQLStore(db, key)
		if sessionErr != nil {
			log.Fatal(sessionErr)
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
		sessions, sessionErr := bffauth.New(bffauth.Config{CookieName: *cookieName, Secure: true, Issuer: issuer, Audience: audience, Store: sessionStore}, jwtVerifier)
		if sessionErr != nil {
			log.Fatal(sessionErr)
		}
		sessions.Register(mux)
		if loginOAuth != nil {
			login, loginErr := bffauth.NewLogin(bffauth.LoginConfig{OAuth: *loginOAuth, CookieKey: key, Secure: true}, sessions)
			if loginErr != nil {
				log.Fatal(loginErr)
			}
			login.Register(mux)
		}
		for _, proxyConfig := range []struct{ mount, target string }{{"/v1/studio/runtime/", *dynamicHTTPURL}, {"/v1/studio/mcp/", *dynamicMCPURL}} {
			target, parseErr := url.Parse(proxyConfig.target)
			if parseErr != nil {
				log.Fatal(parseErr)
			}
			proxy, proxyErr := sessions.Proxy(target, strings.TrimSuffix(proxyConfig.mount, "/"))
			if proxyErr != nil {
				log.Fatal(proxyErr)
			}
			mux.Handle(proxyConfig.mount, proxy)
		}
		if extensionConfig != nil {
			var proxyErr error
			extensionProxy, proxyErr = newExtensionProxy(sessions, extensionConfig)
			if proxyErr != nil {
				log.Fatal(proxyErr)
			}
		}
		gatewayConfig = httptransport.Config{Mode: httptransport.Authenticated, Authenticator: sessions}
	} else {
		target, parseErr := url.Parse(*dynamicMCPURL)
		if parseErr != nil {
			log.Fatal(parseErr)
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
	gateway := httptransport.Gateway{Config: gatewayConfig, Transport: sdkTransport}
	mux.Handle(httptransport.PathPrefix, gateway)
	log.Printf("Studio SDK development host listening on http://%s", *address)
	server := &http.Server{
		Addr: *address, Handler: requestIDs(cors(origin, resolvedMode == string(httptransport.Authenticated), noStore(routeExtensionProxy(mux, extensionProxy)))),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 2 * time.Minute,
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	if err := serveUntilStopped(lifecycleCtx, server, listener); err != nil {
		log.Fatal(err)
	}
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

func resolveAuthenticatedConfig(certURL, issuer, audience, encodedKey string) (string, string, []byte, error) {
	if strings.TrimSpace(certURL) == "" {
		return "", "", nil, errors.New("-jwt-cert-url is required in authenticated mode")
	}
	issuer = strings.TrimSpace(issuer)
	audience = strings.TrimSpace(audience)
	if issuer == "" {
		return "", "", nil, errors.New("-jwt-issuer or STUDIO_JWT_ISSUER is required in authenticated mode")
	}
	if audience == "" {
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

type runtimeProbe struct {
	url, token string
	client     *http.Client
}

func (p runtimeProbe) ProbeRuntime(ctx context.Context) (*sdk.RuntimeHost, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Studio-Runtime-Token", p.token)
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("dynamic runtime status returned %s", response.Status)
	}
	var payload struct {
		AuthenticationMode string `json:"authenticationMode"`
		Status             string `json:"status"`
		Revision           int64  `json:"revision"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Status != "ready" {
		return nil, fmt.Errorf("dynamic runtime is not ready")
	}
	return &sdk.RuntimeHost{AuthenticationMode: payload.AuthenticationMode, Status: payload.Status, Revision: payload.Revision, CheckedAt: time.Now().UTC()}, nil
}

func ensureSchema(ctx context.Context, db *sql.DB) error {
	service, err := migrate.New()
	if err != nil {
		return err
	}
	return service.Up(ctx, db)
}

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && origin != allowedOrigin {
			http.Error(w, "origin is not allowed", http.StatusForbidden)
			return
		}
		if requireUnsafeOrigin && origin == "" && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			http.Error(w, "origin is required", http.StatusForbidden)
			return
		}
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Studio-Development-Subject, X-Request-ID, Mcp-Protocol-Version, Mcp-Method, Mcp-Session-Id, Last-Event-ID")
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
