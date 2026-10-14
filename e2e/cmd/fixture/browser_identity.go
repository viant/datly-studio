package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
)

// This loopback-only e2e host serves synthetic identities. It is never linked
// into production mains, and reads credentials only for the fixed fixture list.
func registerBrowserIdentity(mux *http.ServeMux, proxy *httputil.ReverseProxy, dir string) error {
	tokens := make(map[string]string, len(subjects))
	for _, subject := range subjects {
		data, err := os.ReadFile(filepath.Join(dir, subject+".token"))
		if err != nil {
			return err
		}
		tokens[subject] = strings.TrimSpace(string(data))
	}
	selected := func(r *http.Request) (string, bool) {
		subject := "owner"
		if cookie, err := r.Cookie("studio_e2e_identity"); err == nil {
			subject = cookie.Value
		}
		_, ok := tokens[subject]
		return subject, ok
	}
	login := template.Must(template.New("login").Parse(`<!doctype html><html><head><title>Studio e2e identity</title></head><body><main><h1>Studio e2e identity</h1><form method="post"><label for="identity">Test identity</label><select id="identity" name="identity">{{range .}}<option value="{{.}}">{{.}}</option>{{end}}</select><button type="submit">Sign in</button></form></main></body></html>`))
	mux.HandleFunc("/e2e/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = login.Execute(w, subjects)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid identity", http.StatusBadRequest)
			return
		}
		subject := r.PostForm.Get("identity")
		if _, ok := tokens[subject]; !ok {
			http.Error(w, "unknown fixture identity", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "studio_e2e_identity", Value: subject, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		subject, ok := selected(r)
		if !ok {
			http.Error(w, "unknown fixture identity", http.StatusUnauthorized)
			return
		}
		r.Header.Set("Authorization", "Bearer "+tokens[subject])
		r.Header.Del("X-Studio-Development-Subject")
		proxy.ServeHTTP(w, r)
	})
	mux.HandleFunc("/studio-config.json", func(w http.ResponseWriter, r *http.Request) {
		subject, ok := selected(r)
		if !ok {
			http.Error(w, "unknown fixture identity", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(map[string]any{"mode": "development", "apiBaseURL": "http://127.0.0.1:18780", "development": map[string]string{"subject": subject}}); err != nil {
			fmt.Println("fixture configuration encoding failed")
		}
	})
	return nil
}
