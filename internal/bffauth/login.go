package bffauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const (
	loginPath    = "/v1/studio/auth/login"
	callbackPath = "/v1/studio/auth/callback"
	tokenPath    = "/v1/studio/auth/token"
	stateCookie  = "studio_login_state"
	stateTTL     = 5 * time.Minute
)

type LoginConfig struct {
	OAuth         oauth2.Config
	CookieKey     []byte // the deployment's shared 32-byte BFF session key
	Secure        bool
	UseIDToken    bool
	AllowedOrigin string
}

// Login owns authorization-code + S256 PKCE and the provider token lifecycle.
// Identity-token deployments mount DatlyHTTP. Register keeps credentials on
// the server and exposes only the opaque browser session cookie.
type Login struct {
	config          oauth2.Config
	sessions        *Service
	aead            cipher.AEAD
	secure          bool
	now             func() time.Time
	client          *http.Client
	useIDToken      bool
	allowedOrigin   string
	stateCookieName string
	refreshMu       sync.Mutex
}

type loginState struct {
	ReturnTo string `json:"returnTo,omitempty"`
	State    string `json:"state"`
	Verifier string `json:"verifier"`
	Expires  int64  `json:"expires"`
}

func NewLogin(config LoginConfig, sessions *Service) (*Login, error) {
	if sessions == nil {
		return nil, errors.New("BFF login requires a session service")
	}
	if len(config.CookieKey) != 32 {
		return nil, errors.New("BFF login requires a 32-byte state encryption key")
	}
	if strings.TrimSpace(config.OAuth.ClientID) == "" || !slices.Contains(config.OAuth.Scopes, "openid") {
		return nil, errors.New("BFF login requires an OAuth client ID and openid scope")
	}
	if err := validateLoginURL(config.OAuth.Endpoint.AuthURL, "authorization endpoint"); err != nil {
		return nil, err
	}
	if err := validateLoginURL(config.OAuth.Endpoint.TokenURL, "token endpoint"); err != nil {
		return nil, err
	}
	redirect, err := url.Parse(config.OAuth.RedirectURL)
	if err != nil || redirect == nil || redirect.Path != callbackPath || redirect.RawQuery != "" || redirect.Fragment != "" {
		return nil, errors.New("BFF login redirect URI must use the exact Studio callback path")
	}
	if err := validateLoginURL(config.OAuth.RedirectURL, "redirect URI"); err != nil {
		return nil, err
	}
	if config.UseIDToken && strings.TrimSpace(config.AllowedOrigin) == "" {
		return nil, errors.New("browser identity-token flow requires an allowed origin")
	}
	key := hmac.New(sha256.New, config.CookieKey)
	key.Write([]byte("datly-studio/bff-login-state/v1"))
	block, err := aes.NewCipher(key.Sum(nil))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	stateCookieName := stateCookie
	if sessions.config.CookieName != DefaultCookieName {
		stateCookieName = sessions.config.CookieName + "_login_state"
	}
	login := &Login{stateCookieName: stateCookieName, config: config.OAuth, sessions: sessions, aead: aead, secure: config.Secure,
		useIDToken: config.UseIDToken, allowedOrigin: strings.TrimRight(config.AllowedOrigin, "/"),
		now: time.Now, client: &http.Client{Timeout: 10 * time.Second}}
	sessions.refreshSession = login.freshIdentitySession
	return login, nil
}

func validateLoginURL(value, name string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.User != nil || parsed.Opaque != "" || parsed.Hostname() == "" || parsed.Fragment != "" {
		return fmt.Errorf("BFF login %s must be an absolute HTTPS URL", name)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || net.ParseIP(parsed.Hostname()) != nil && net.ParseIP(parsed.Hostname()).IsLoopback()) {
		return nil
	}
	return fmt.Errorf("BFF login %s must be an absolute HTTPS URL", name)
}

func (l *Login) Register(mux *http.ServeMux) error {
	if l.useIDToken {
		return errors.New("identity-token login must use DatlyHTTP")
	}
	mux.HandleFunc("GET "+loginPath, l.handleStart)
	mux.HandleFunc("GET "+callbackPath, l.handleCallback)
	return nil
}

func (l *Login) handleStart(w http.ResponseWriter, r *http.Request) {
	state, err := randomID()
	if err != nil {
		http.Error(w, "login unavailable", http.StatusServiceUnavailable)
		return
	}
	verifier := oauth2.GenerateVerifier()
	pending := loginState{State: state, Verifier: verifier, Expires: l.now().Add(stateTTL).Unix(), ReturnTo: loginReturnTo(r.URL.Query()["returnTo"])}
	encoded, err := l.seal(pending)
	if err != nil {
		http.Error(w, "login unavailable", http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: l.stateCookieName, Value: encoded, Path: callbackPath, HttpOnly: true,
		Secure: l.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(stateTTL.Seconds())})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, l.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func (l *Login) handleCallback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: l.stateCookieName, Value: "", Path: callbackPath, HttpOnly: true,
		Secure: l.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	values := r.URL.Query()
	if values.Has("error") || len(values["state"]) != 1 || len(values["code"]) != 1 || values.Get("state") == "" || values.Get("code") == "" {
		http.Error(w, "login was not completed", http.StatusUnauthorized)
		return
	}
	cookie, err := r.Cookie(l.stateCookieName)
	if err != nil {
		http.Error(w, "login state is unavailable", http.StatusUnauthorized)
		return
	}
	pending, err := l.open(cookie.Value)
	if err != nil || l.now().Unix() >= pending.Expires || subtle.ConstantTimeCompare([]byte(values.Get("state")), []byte(pending.State)) != 1 {
		http.Error(w, "login state is invalid", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, l.client)
	token, err := l.config.Exchange(ctx, values.Get("code"), oauth2.VerifierOption(pending.Verifier))
	if err != nil || token == nil || strings.TrimSpace(token.AccessToken) == "" {
		http.Error(w, "login token exchange failed", http.StatusUnauthorized)
		return
	}
	// OIDC identifies the user with its ID token. An OAuth access token can be
	// opaque or carry a different resource audience; it is not a substitute for
	// those identity claims. OAuth-only providers retain verified JWT support.
	credential := token.AccessToken
	if identityToken, ok := token.Extra("id_token").(string); ok && strings.TrimSpace(identityToken) != "" {
		credential = identityToken
	}
	var id string
	var expires time.Time
	if identityToken, ok := token.Extra("id_token").(string); ok && strings.TrimSpace(identityToken) != "" && token.RefreshToken != "" {
		id, _, expires, err = l.sessions.ExchangeIDToken(ctx, identityToken, token.RefreshToken)
	} else {
		id, _, expires, err = l.sessions.Exchange(ctx, credential)
	}
	if err != nil {
		http.Error(w, "login token verification failed", http.StatusUnauthorized)
		return
	}
	l.sessions.setSessionCookie(w, id, expires)
	http.Redirect(w, r, loginReturnTo([]string{pending.ReturnTo}), http.StatusSeeOther)
}

func (l *Login) seal(state loginState) (string, error) {
	plain, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, l.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := l.aead.Seal(nonce, nonce, plain, []byte(callbackPath))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (l *Login) open(encoded string) (loginState, error) {
	var state loginState
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) < l.aead.NonceSize() {
		return state, errors.New("invalid login state")
	}
	plain, err := l.aead.Open(nil, data[:l.aead.NonceSize()], data[l.aead.NonceSize():], []byte(callbackPath))
	if err != nil || json.Unmarshal(plain, &state) != nil || state.State == "" || state.Verifier == "" {
		return loginState{}, errors.New("invalid login state")
	}
	return state, nil
}
