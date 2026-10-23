package bffauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

// Browsers share cookies across ports. Two native Studio login flows must retain
// separate PKCE state, and completing one must not clear the other's pending flow.
func TestDatlyLoginStateCookiesAreIsolatedAcrossApplications(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		if r.Form.Get("code") != "approved" || r.Form.Get("code_verifier") == "" {
			t.Error("invalid synthetic PKCE exchange")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-1", "id_token": "id-2", "refresh_token": "refresh-1", "token_type": "Bearer", "expires_in": 3600})
	}))
	defer provider.Close()
	jar, _ := cookiejar.New(nil)
	type flow struct {
		handler           http.Handler
		origin            *url.URL
		state, cookieName string
	}
	flows := make([]flow, 0, 2)
	for i, origin := range []string{"http://127.0.0.1:18192", "http://127.0.0.1:18195"} {
		name := []string{"first_studio_session", "second_studio_session"}[i]
		sessions, err := New(Config{Issuer: "https://identity.example", Audience: "studio-web", CookieName: name}, identityVerifier{})
		if err != nil {
			t.Fatal(err)
		}
		login, err := NewLogin(LoginConfig{OAuth: oauth2.Config{ClientID: "studio-web", RedirectURL: origin + callbackPath, Scopes: []string{"openid"}, Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}}, CookieKey: bytes.Repeat([]byte{byte(i + 1)}, 32), UseIDToken: true, AllowedOrigin: origin}, sessions)
		if err != nil {
			t.Fatal(err)
		}
		handler, err := login.DatlyHTTP()
		if err != nil {
			t.Fatal(err)
		}
		start := httptest.NewRecorder()
		handler.ServeHTTP(start, httptest.NewRequest(http.MethodGet, origin+loginPath, nil))
		if start.Code != http.StatusFound {
			t.Fatalf("start status=%d", start.Code)
		}
		cookies := start.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != name+"_login_state" {
			t.Fatalf("native login must use its application's state cookie; got names=%v", cookieNames(cookies))
		}
		originURL, _ := url.Parse(origin)
		jar.SetCookies(originURL, cookies)
		redirect, err := url.Parse(start.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		flows = append(flows, flow{handler, originURL, redirect.Query().Get("state"), name})
	}
	for _, flow := range flows {
		callbackURL, _ := url.Parse(flow.origin.String() + callbackPath + "?code=approved&state=" + url.QueryEscape(flow.state))
		request := httptest.NewRequest(http.MethodGet, callbackURL.String(), nil)
		for _, cookie := range jar.Cookies(callbackURL) {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		flow.handler.ServeHTTP(response, request)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("callback for %s failed: status=%d body=%s", flow.cookieName, response.Code, response.Body.String())
		}
		foundSession, clearedOwn := false, false
		for _, cookie := range response.Result().Cookies() {
			if cookie.Name == flow.cookieName {
				foundSession = true
			}
			if cookie.Name == flow.cookieName+"_login_state" && cookie.MaxAge < 0 {
				clearedOwn = true
			}
		}
		if !foundSession || !clearedOwn {
			t.Fatal("callback must create its session and clear only its own login state")
		}
		jar.SetCookies(callbackURL, response.Result().Cookies())
	}
}

func cookieNames(cookies []*http.Cookie) []string {
	result := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		result = append(result, cookie.Name)
	}
	return result
}
