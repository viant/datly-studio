// Package bffauth provides Studio's opaque, HttpOnly BFF session boundary.
package bffauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/scy/auth/jwt"
)

const DefaultCookieName = "studio_session"

type ClaimsVerifier interface {
	VerifyClaims(context.Context, string) (*jwt.Claims, error)
}

type Config struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
	Issuer     string
	Audience   string
	Store      SessionStore
}

type session struct {
	principal sdk.Principal
	token     string
	claims    *jwt.Claims
	expiresAt time.Time
}

// SessionStore keeps opaque BFF sessions outside the serving process so a
// session remains valid when requests move between instances.
type SessionStore interface {
	Put(context.Context, string, session) error
	Get(context.Context, string) (session, bool, error)
	Delete(context.Context, string) error
	DeleteExpired(context.Context, time.Time) error
}

// Service is both the auth HTTP surface and the SDK gateway authenticator.
// Tokens remain server-side; the cookie contains only a random session ID.
type Service struct {
	config   Config
	verifier ClaimsVerifier
	now      func() time.Time
	store    SessionStore
}

func New(config Config, verifier ClaimsVerifier) (*Service, error) {
	if verifier == nil {
		return nil, errors.New("BFF JWT verifier is required")
	}
	if strings.TrimSpace(config.CookieName) == "" {
		config.CookieName = DefaultCookieName
	}
	if config.TTL <= 0 {
		config.TTL = 12 * time.Hour
	}
	if config.Store == nil {
		config.Store = newMemoryStore()
	}
	return &Service{config: config, verifier: verifier, now: time.Now, store: config.Store}, nil
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/studio/auth/me", s.handleMe)
	mux.HandleFunc("POST /v1/studio/auth/session", s.handleSession)
	mux.HandleFunc("DELETE /v1/studio/auth/session", s.handleLogout)
}

// Proxy expands the opaque session cookie into a server-side bearer header.
// Browser cookies are stripped before the request reaches Datly.
func (s *Service) Proxy(target *url.URL, mount string) (http.Handler, error) {
	if target == nil || target.Scheme == "" || target.Host == "" {
		return nil, errors.New("dynamic Datly proxy target is required")
	}
	mount = "/" + strings.Trim(strings.TrimSpace(mount), "/")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = stripUpstreamCORS
	original := proxy.Director
	proxy.Director = func(request *http.Request) {
		original(request)
		request.URL.Path = "/" + strings.TrimPrefix(strings.TrimPrefix(request.URL.Path, mount), "/")
		request.Header.Del("Cookie")
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		path := "/" + strings.TrimPrefix(strings.TrimPrefix(request.URL.Path, mount), "/")
		if path == "/_studio" || strings.HasPrefix(path, "/_studio/") {
			writeJSON(response, http.StatusNotFound, map[string]string{"message": "dynamic administration is not exposed through the browser proxy"})
			return
		}
		current, err := s.sessionForRequest(request)
		if err != nil {
			writeJSON(response, http.StatusUnauthorized, map[string]string{"message": "Studio authentication is required"})
			return
		}
		forward := request.Clone(request.Context())
		forward.Header = make(http.Header)
		for _, name := range []string{"Accept", "Content-Type", "Mcp-Session-Id", "Mcp-Protocol-Version", "Mcp-Method", "Last-Event-ID", "X-Request-ID"} {
			for _, value := range request.Header.Values(name) {
				forward.Header.Add(name, value)
			}
		}
		forward.Header.Set("Authorization", "Bearer "+current.token)
		proxy.ServeHTTP(response, forward)
	}), nil
}

// stripUpstreamCORS leaves one browser-facing policy owner: Studio's BFF.
// Passing Datly's transport headers through would create duplicate
// Access-Control-Allow-Origin/Credentials values, which browsers reject.
func stripUpstreamCORS(response *http.Response) error {
	for _, header := range []string{
		"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers", "Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers", "Access-Control-Max-Age",
	} {
		response.Header.Del(header)
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, request *http.Request) (context.Context, error) {
	current, err := s.sessionForRequest(request)
	if err != nil {
		return nil, err
	}
	ctx = sdk.WithPrincipal(ctx, current.principal)
	return sdk.WithVerifiedCredential(ctx, sdk.VerifiedCredential{Bearer: current.token, Claims: current.claims}), nil
}

func (s *Service) handleMe(response http.ResponseWriter, request *http.Request) {
	principal, err := s.principal(request)
	if err != nil {
		writeJSON(response, http.StatusUnauthorized, map[string]any{"authenticated": false})
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"authenticated": true, "subject": principal.Subject})
}

func (s *Service) handleSession(response http.ResponseWriter, request *http.Request) {
	token, err := bearer(request.Header.Get("Authorization"))
	if err != nil {
		writeJSON(response, http.StatusUnauthorized, map[string]string{"message": "verified bearer token is required"})
		return
	}
	id, principal, expires, err := s.Exchange(request.Context(), token)
	if err != nil {
		writeJSON(response, http.StatusUnauthorized, map[string]string{"message": "token verification failed"})
		return
	}
	s.setSessionCookie(response, id, expires)
	writeJSON(response, http.StatusCreated, map[string]any{"authenticated": true, "subject": principal.Subject})
}

func (s *Service) setSessionCookie(response http.ResponseWriter, id string, expires time.Time) {
	maxAge := int(expires.Sub(s.now()).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(response, &http.Cookie{Name: s.config.CookieName, Value: id, Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: maxAge})
}

// Exchange verifies a JWT and pre-seeds an opaque session for server-side OOB
// login flows. The returned ID belongs only in an HttpOnly cookie.
func (s *Service) Exchange(ctx context.Context, token string) (string, sdk.Principal, time.Time, error) {
	claims, err := s.verifier.VerifyClaims(ctx, strings.TrimSpace(token))
	if err != nil {
		return "", sdk.Principal{}, time.Time{}, err
	}
	if err = validateTokenBinding(claims, s.config.Issuer, s.config.Audience); err != nil {
		return "", sdk.Principal{}, time.Time{}, err
	}
	if (strings.TrimSpace(s.config.Issuer) != "" || strings.TrimSpace(s.config.Audience) != "") && claims.ExpiresAt == nil {
		return "", sdk.Principal{}, time.Time{}, errors.New("bound BFF access token requires an expiry")
	}
	principal, err := principalFromClaims(claims)
	if err != nil {
		return "", sdk.Principal{}, time.Time{}, err
	}
	id, err := randomID()
	if err != nil {
		return "", sdk.Principal{}, time.Time{}, err
	}
	expires := s.now().Add(s.config.TTL)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(expires) {
		expires = claims.ExpiresAt.Time
	}
	if !s.now().Before(expires) {
		return "", sdk.Principal{}, time.Time{}, errors.New("verified JWT is expired")
	}
	if err = s.store.DeleteExpired(ctx, s.now()); err != nil {
		return "", sdk.Principal{}, time.Time{}, fmt.Errorf("remove expired BFF sessions: %w", err)
	}
	if err = s.store.Put(ctx, id, session{principal: principal, token: strings.TrimSpace(token), claims: claims, expiresAt: expires}); err != nil {
		return "", sdk.Principal{}, time.Time{}, fmt.Errorf("persist BFF session: %w", err)
	}
	return id, principal, expires, nil
}

func (s *Service) handleLogout(response http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie(s.config.CookieName); err == nil {
		_ = s.store.Delete(request.Context(), cookie.Value)
	}
	http.SetCookie(response, &http.Cookie{Name: s.config.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	response.WriteHeader(http.StatusNoContent)
}

func (s *Service) principal(request *http.Request) (sdk.Principal, error) {
	current, err := s.sessionForRequest(request)
	if err != nil {
		return sdk.Principal{}, err
	}
	return current.principal, nil
}

func (s *Service) sessionForRequest(request *http.Request) (session, error) {
	cookie, err := request.Cookie(s.config.CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return session{}, errors.New("Studio session is required")
	}
	current, ok, err := s.store.Get(request.Context(), cookie.Value)
	if err != nil {
		return session{}, fmt.Errorf("load Studio session: %w", err)
	}
	if !ok || !s.now().Before(current.expiresAt) {
		if ok {
			_ = s.store.Delete(request.Context(), cookie.Value)
		}
		return session{}, errors.New("Studio session is expired")
	}
	return current, nil
}

func validateTokenBinding(claims *jwt.Claims, issuer, audience string) error {
	issuer = strings.TrimSpace(issuer)
	audience = strings.TrimSpace(audience)
	if issuer != "" && (claims == nil || claims.Issuer != issuer) {
		return errors.New("JWT issuer does not match the configured issuer")
	}
	if audience != "" && (claims == nil || !claims.VerifyAudience(audience, true)) {
		return errors.New("JWT audience does not contain the configured audience")
	}
	return nil
}

func principalFromClaims(claims *jwt.Claims) (sdk.Principal, error) {
	if claims == nil {
		return sdk.Principal{}, errors.New("verified JWT claims are required")
	}
	if subject := strings.TrimSpace(claims.Subject); subject != "" {
		return sdk.Principal{Subject: subject}, nil
	}
	return sdk.Principal{}, errors.New("JWT subject is required")
}

func bearer(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Bearer token is required")
	}
	return parts[1], nil
}

func randomID() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("random session ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
