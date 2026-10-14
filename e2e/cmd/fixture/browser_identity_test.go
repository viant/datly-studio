package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserFixtureIdentityIsAllowlistedAndServerSelected(t *testing.T) {
	dir := t.TempDir()
	for _, subject := range subjects {
		if err := os.WriteFile(filepath.Join(dir, subject+".token"), []byte("synthetic-"+subject), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var gotBearer, gotSubject string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBearer = r.Header.Get("Authorization")
		gotSubject = r.Header.Get("X-Studio-Development-Subject")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	mux := http.NewServeMux()
	if err := registerBrowserIdentity(mux, httputil.NewSingleHostReverseProxy(target), dir); err != nil {
		t.Fatal(err)
	}
	login := httptest.NewRequest(http.MethodPost, "/e2e/login", strings.NewReader("identity=namespace-admin"))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	result := httptest.NewRecorder()
	mux.ServeHTTP(result, login)
	cookies := result.Result().Cookies()
	if result.Code != http.StatusSeeOther || len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("login did not set protected fixture cookie")
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.list", strings.NewReader("{}"))
	req.AddCookie(cookies[0])
	req.Header.Set("Authorization", "Bearer forged")
	req.Header.Set("X-Studio-Development-Subject", "owner")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent || gotBearer != "Bearer synthetic-namespace-admin" || gotSubject != "" {
		t.Fatal("browser headers changed fixture identity")
	}
	config := httptest.NewRequest(http.MethodGet, "/studio-config.json", nil)
	config.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, config)
	data, _ := io.ReadAll(response.Result().Body)
	if !strings.Contains(string(data), `"subject":"namespace-admin"`) || strings.Contains(string(data), "synthetic-") {
		t.Fatal("configuration identity mismatch or credential exposure")
	}
	bad := httptest.NewRequest(http.MethodPost, "/e2e/login", strings.NewReader("identity=../../secret"))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, bad)
	if response.Code != http.StatusBadRequest || len(response.Result().Cookies()) != 0 {
		t.Fatal("unknown identity accepted")
	}
	bad = httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.list", nil)
	bad.AddCookie(&http.Cookie{Name: "studio_e2e_identity", Value: "unknown"})
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, bad)
	if response.Code != http.StatusUnauthorized {
		t.Fatal("unknown cookie fell back to owner")
	}
}
