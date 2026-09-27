package bffauth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/xdatly"
	xresponse "github.com/viant/xdatly/response"
	"golang.org/x/oauth2"
)

// These auth components are mounted only on Studio's auth HTTP host. They have
// no MCP declarations and are not registered in the static or dynamic MCP host.
type browserAuthComponents struct {
	Start    xdatly.Component[authStartInput, authHTTPOutput]    `component:"start,path=/v1/studio/auth/login,method=GET"`
	Callback xdatly.Component[authCallbackInput, authHTTPOutput] `component:"callback,path=/v1/studio/auth/callback,method=GET"`
	Token    xdatly.Component[authTokenInput, authHTTPOutput]    `component:"token,path=/v1/studio/auth/token,method=POST"`
	Logout   xdatly.Component[authLogoutInput, authHTTPOutput]   `component:"logout,path=/v1/studio/auth/session,method=DELETE"`
}

type authStartInput struct{}
type authCallbackInput struct {
	Codes      []string `parameter:"Codes,kind=query,in=code"`
	States     []string `parameter:"States,kind=query,in=state"`
	Error      string   `parameter:"Error,kind=query,in=error"`
	FlowCookie string   `parameter:"FlowCookie,kind=cookie,in=studio_login_state"`
}
type authTokenInput struct {
	Origin       string `parameter:"Origin,kind=header,in=Origin"`
	CookieHeader string `parameter:"CookieHeader,kind=header,in=Cookie"`
}
type authLogoutInput struct {
	Origin       string `parameter:"Origin,kind=header,in=Origin"`
	CookieHeader string `parameter:"CookieHeader,kind=header,in=Cookie"`
}
type authHTTPOutput struct{ *xresponse.Buffered }

// DatlyHTTP returns a dedicated HTTP-only Datly runtime for browser auth.
// The Login value is trusted backend configuration captured at construction;
// request inputs cannot replace it. Session persistence remains encrypted in
// the existing private Datly reader/writer components.
func (l *Login) DatlyHTTP() (http.Handler, error) {
	if l == nil || !l.useIDToken {
		return nil, errors.New("identity-token login is required for Datly auth routes")
	}
	registered := make([]*registry.RegisteredComponent, 0, 4)
	for _, component := range []struct {
		field   string
		input   reflect.Type
		handler rhandler.TypedHandler
	}{
		{"Start", reflect.TypeFor[authStartInput](), custom.NewFunc[authStartInput, authHTTPOutput](l.datlyStart)},
		{"Callback", reflect.TypeFor[authCallbackInput](), custom.NewFunc[authCallbackInput, authHTTPOutput](l.datlyCallback)},
		{"Token", reflect.TypeFor[authTokenInput](), custom.NewFunc[authTokenInput, authHTTPOutput](l.datlyToken)},
		{"Logout", reflect.TypeFor[authLogoutInput](), custom.NewFunc[authLogoutInput, authHTTPOutput](l.datlyLogout)},
	} {
		holder := reflect.TypeFor[browserAuthComponents]()
		field, ok := holder.FieldByName(component.field)
		if !ok {
			return nil, fmt.Errorf("browser auth component %s is missing", component.field)
		}
		metadata, present, err := dtag.ParseComponent(field.Tag)
		if err != nil || !present {
			return nil, fmt.Errorf("browser auth component %s metadata: %w", component.field, err)
		}
		contract, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
			PackageName: "browser_auth", PackagePath: holder.PkgPath(), Tag: metadata,
			InputType: component.input.Name(), OutputType: reflect.TypeFor[authHTTPOutput]().Name(),
		}).Resolve(component.input, reflect.TypeFor[authHTTPOutput]())
		if err != nil {
			return nil, fmt.Errorf("browser auth component %s: %w", component.field, err)
		}
		if len(contract.Routes) != 1 || len(contract.Routes[0].MCP) != 0 {
			return nil, fmt.Errorf("browser auth component %s must be HTTP-only", component.field)
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract, InputType: component.input,
			OutputType: reflect.TypeFor[authHTTPOutput](), Handler: component.handler, HandlerOwnedOutput: true})
		if err != nil {
			return nil, fmt.Errorf("build browser auth component %s: %w", component.field, err)
		}
		registration, err := artifact.Registration(registry.RegisteredComponent{})
		if err != nil {
			return nil, err
		}
		registered = append(registered, registration)
	}
	runtime, err := druntime.NewRuntime(registered, druntime.WithExposedPackages([]string{reflect.TypeFor[browserAuthComponents]().PkgPath()}, nil))
	if err != nil {
		return nil, err
	}
	return gateway.NewHandler(runtime, nil, "studio-browser-auth"), nil
}

func authResponse(status int, body string, options ...xresponse.Option) *authHTTPOutput {
	all := []xresponse.Option{xresponse.WithStatusCode(status), xresponse.WithHeader("Cache-Control", "no-store"),
		xresponse.WithHeader("Referrer-Policy", "no-referrer")}
	if body != "" {
		all = append(all, xresponse.WithHeader("Content-Type", "text/plain; charset=utf-8"), xresponse.WithBytes([]byte(body+"\n")))
	}
	all = append(all, options...)
	return &authHTTPOutput{Buffered: xresponse.NewBuffered(all...)}
}

func (l *Login) datlyStart(_ context.Context, _ *authStartInput) (*authHTTPOutput, error) {
	state, err := randomID()
	if err != nil {
		return authResponse(http.StatusServiceUnavailable, "login unavailable"), nil
	}
	verifier := oauth2.GenerateVerifier()
	encoded, err := l.seal(loginState{State: state, Verifier: verifier, Expires: l.now().Add(stateTTL).Unix()})
	if err != nil {
		return authResponse(http.StatusServiceUnavailable, "login unavailable"), nil
	}
	cookie := (&http.Cookie{Name: stateCookie, Value: encoded, Path: callbackPath, HttpOnly: true,
		Secure: l.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(stateTTL.Seconds())}).String()
	return authResponse(http.StatusFound, "", xresponse.WithHeader("Set-Cookie", cookie),
		xresponse.WithHeader("Location", l.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)))), nil
}

func (l *Login) datlyCallback(ctx context.Context, in *authCallbackInput) (*authHTTPOutput, error) {
	clear := xresponse.WithHeader("Set-Cookie", (&http.Cookie{Name: stateCookie, Value: "", Path: callbackPath,
		HttpOnly: true, Secure: l.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1}).String())
	if in == nil || in.Error != "" || len(in.Codes) != 1 || len(in.States) != 1 || in.Codes[0] == "" || in.States[0] == "" {
		return authResponse(http.StatusUnauthorized, "login was not completed", clear), nil
	}
	if in.FlowCookie == "" {
		return authResponse(http.StatusUnauthorized, "login state is unavailable", clear), nil
	}
	pending, err := l.open(in.FlowCookie)
	if err != nil || l.now().Unix() >= pending.Expires || subtle.ConstantTimeCompare([]byte(in.States[0]), []byte(pending.State)) != 1 {
		return authResponse(http.StatusUnauthorized, "login state is invalid", clear), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, l.client)
	token, err := l.config.Exchange(ctx, in.Codes[0], oauth2.VerifierOption(pending.Verifier))
	if err != nil || token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return authResponse(http.StatusUnauthorized, "login token exchange failed", clear), nil
	}
	idToken, _ := token.Extra("id_token").(string)
	id, _, expires, err := l.sessions.ExchangeIDToken(ctx, idToken, token.RefreshToken)
	if err != nil {
		return authResponse(http.StatusUnauthorized, "login token verification failed", clear), nil
	}
	return authResponse(http.StatusSeeOther, "", clear,
		xresponse.WithHeader("Set-Cookie", l.sessions.sessionCookie(id, expires).String()),
		xresponse.WithHeader("Location", "/")), nil
}

func (l *Login) datlyToken(ctx context.Context, in *authTokenInput) (*authHTTPOutput, error) {
	if in == nil || in.Origin != l.allowedOrigin {
		return authResponse(http.StatusForbidden, "invalid request origin"), nil
	}
	id := cookieValue(in.CookieHeader, l.sessions.config.CookieName)
	if id == "" {
		return authResponse(http.StatusUnauthorized, "sign in required"), nil
	}
	coordinator, distributed := l.sessions.store.(refreshLeaseStore)
	if !distributed {
		l.refreshMu.Lock()
		defer l.refreshMu.Unlock()
	}
	current, err := l.sessions.sessionForID(ctx, id)
	if err != nil || current.refreshToken == "" {
		return authResponse(http.StatusUnauthorized, "sign in required"), nil
	}
	if !l.now().Add(time.Minute).Before(current.idTokenExpiresAt) {
		owner := ""
		if distributed {
			owner, err = randomID()
			if err != nil {
				return authResponse(http.StatusServiceUnavailable, "identity refresh unavailable"), nil
			}
			acquired, leaseErr := coordinator.AcquireRefreshLease(ctx, id, owner, l.now(), 30*time.Second)
			if leaseErr != nil {
				return authResponse(http.StatusServiceUnavailable, "identity refresh unavailable"), nil
			}
			if !acquired {
				current, err = l.awaitRotatedSession(ctx, id, current.refreshToken)
				if err != nil {
					return authResponse(http.StatusServiceUnavailable, "identity refresh is in progress"), nil
				}
				return identityTokenResponse(current), nil
			}
			defer func() {
				releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				_ = coordinator.ReleaseRefreshLease(releaseCtx, id, owner, l.now())
			}()
			current, err = l.sessions.sessionForID(ctx, id)
			if err != nil || current.refreshToken == "" {
				return authResponse(http.StatusUnauthorized, "sign in required"), nil
			}
			if l.now().Add(time.Minute).Before(current.idTokenExpiresAt) {
				return identityTokenResponse(current), nil
			}
		}
		refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		refreshCtx = context.WithValue(refreshCtx, oauth2.HTTPClient, l.client)
		base := &oauth2.Token{RefreshToken: current.refreshToken, Expiry: l.now().Add(-time.Second)}
		refreshed, refreshErr := l.config.TokenSource(refreshCtx, base).Token()
		if refreshErr != nil || refreshed == nil {
			return authResponse(http.StatusServiceUnavailable, "identity token refresh failed"), nil
		}
		idToken, _ := refreshed.Extra("id_token").(string)
		claims, verifyErr := l.sessions.verifier.VerifyClaims(refreshCtx, idToken)
		if verifyErr != nil || claims == nil || validateTokenBinding(claims, l.sessions.config.Issuer, l.sessions.config.Audience) != nil ||
			claims.ExpiresAt == nil || !l.now().Before(claims.ExpiresAt.Time) ||
			claims.ExpiresAt.Time.After(l.now().Add(maximumBrowserIDTokenTTL)) || claims.Subject != current.principal.Subject {
			return authResponse(http.StatusUnauthorized, "refreshed identity token was rejected"), nil
		}
		current.token, current.claims, current.idTokenExpiresAt = idToken, claims, claims.ExpiresAt.Time
		if refreshed.RefreshToken != "" {
			current.refreshToken = refreshed.RefreshToken
		}
		commitCtx, commitCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer commitCancel()
		if distributed {
			err = coordinator.CompleteRefreshLease(commitCtx, id, owner, l.now(), current)
		} else {
			err = l.sessions.store.Put(commitCtx, id, current)
		}
		if err != nil {
			return authResponse(http.StatusServiceUnavailable, "identity session update failed"), nil
		}
	}
	return identityTokenResponse(current), nil
}

type refreshLeaseStore interface {
	AcquireRefreshLease(context.Context, string, string, time.Time, time.Duration) (bool, error)
	CompleteRefreshLease(context.Context, string, string, time.Time, session) error
	ReleaseRefreshLease(context.Context, string, string, time.Time) error
}

func (l *Login) awaitRotatedSession(ctx context.Context, id, oldRefreshToken string) (session, error) {
	deadline := time.NewTimer(12 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := l.sessions.sessionForID(ctx, id)
		// A concurrent SQLite writer can temporarily prevent a cross-instance
		// read. Retry until the bounded refresh wait ends; never call the IdP
		// with the same rotating refresh token from this follower.
		if err == nil && current.refreshToken != oldRefreshToken && l.now().Add(time.Minute).Before(current.idTokenExpiresAt) {
			return current, nil
		}
		select {
		case <-ctx.Done():
			return session{}, ctx.Err()
		case <-deadline.C:
			return session{}, errors.New("refresh lease did not complete")
		case <-ticker.C:
		}
	}
}

func identityTokenResponse(current session) *authHTTPOutput {
	body, err := json.Marshal(map[string]any{"id_token": current.token, "subject": current.principal.Subject})
	if err != nil {
		return authResponse(http.StatusInternalServerError, "identity response failed")
	}
	return authResponse(http.StatusOK, "", xresponse.WithHeader("Content-Type", "application/json"), xresponse.WithBytes(append(body, '\n')))
}

func (l *Login) datlyLogout(ctx context.Context, in *authLogoutInput) (*authHTTPOutput, error) {
	if in == nil || in.Origin != l.allowedOrigin {
		return authResponse(http.StatusForbidden, "invalid request origin"), nil
	}
	if id := cookieValue(in.CookieHeader, l.sessions.config.CookieName); id != "" {
		if err := l.sessions.store.Delete(ctx, id); err != nil {
			return authResponse(http.StatusServiceUnavailable, "logout unavailable"), nil
		}
	}
	clear := (&http.Cookie{Name: l.sessions.config.CookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: l.sessions.config.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1}).String()
	return authResponse(http.StatusNoContent, "", xresponse.WithHeader("Set-Cookie", clear)), nil
}

func cookieValue(header, name string) string {
	if header == "" || name == "" {
		return ""
	}
	request := &http.Request{Header: http.Header{"Cookie": {header}}}
	cookie, err := request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
