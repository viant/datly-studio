// fixture hosts real Studio components with local synthetic credentials.
// Endly owns consumer fixtures, tested MCP calls, browser actions and assertions.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/app/datlycmd"
	"github.com/viant/datly-studio/e2e/predicates"
	_ "github.com/viant/datly-studio/examples/predicates"
	"github.com/viant/datly-studio/runtime/accessprovider"
	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/store/sql/migrate"
	studiohost "github.com/viant/datly-studio/studio/host"
	"go.yaml.in/yaml/v3"
	"log"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const predicatePackage = "github.com/viant/datly-studio/e2e/predicates"

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func write(name string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	check(err)
	check(os.WriteFile(name, data, 0600))
}
func main() {
	mode := flag.String("mode", "serve", "prepare or serve")
	root := flag.String("root", "", "Studio repository root")
	dir := flag.String("dir", "", "dedicated e2e working directory")
	flag.Parse()
	if !filepath.IsAbs(*root) || !filepath.IsAbs(*dir) || !strings.HasPrefix(*dir, filepath.Join(*root, ".data", "e2e")+string(os.PathSeparator)) {
		log.Fatal("root and isolated .data/e2e/<run> directory required")
	}
	check(os.MkdirAll(*dir, 0700))
	if *mode == "wait" {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			ready := true
			for _, port := range []string{"18780", "18790"} {
				c, e := net.DialTimeout("tcp", "127.0.0.1:"+port, 200*time.Millisecond)
				if e != nil {
					ready = false
				} else {
					c.Close()
				}
			}
			if ready {
				fmt.Println("Studio HTTP and MCP ready")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		log.Fatal("Studio HTTP/MCP startup deadline exceeded")
	}
	if *mode == "prepare" {
		prepare(*root, *dir)
		return
	}
	if *mode == "prepare-idp" {
		prepareIDP(*dir)
		return
	}
	if *mode != "serve" {
		log.Fatal("invalid mode")
	}
	serve(*root, *dir)
}
func studioDSN(dir string) string {
	return "file:" + filepath.Join(dir, "studio.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

func prepare(root, dir string) {
	// Refuse an existing catalog; Endly chooses a fresh run directory.
	if _, err := os.Stat(filepath.Join(dir, "studio.db")); err == nil {
		log.Fatal("catalog exists; use a new e2e run directory")
	}
	db, err := sql.Open("sqlite", studioDSN(dir))
	check(err)
	migration, err := migrate.New()
	check(err)
	check(migration.Up(context.Background(), db))
	check(db.Close())
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	check(err)
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	check(err)
	check(os.WriteFile(filepath.Join(dir, "public.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}), 0600))
	check(os.WriteFile(filepath.Join(dir, "private.pem"), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600))
	check(os.MkdirAll(filepath.Join(dir, "claims"), 0700))
	for index, subject := range subjects {
		claims := map[string]any{"sub": subject, "user_id": index + 1, "account_id": 42, "iss": "studio-e2e", "aud": "studio-e2e", "tenant": "studio-e2e"}
		if subject == "wrong-issuer" {
			claims["iss"] = "other-e2e"
		}
		encoded, err := json.Marshal(claims)
		check(err)
		check(os.WriteFile(filepath.Join(dir, "claims", subject+".json"), encoded, 0600))
	}
	configBytes, err := os.ReadFile(filepath.Join(root, "datly.yaml"))
	check(err)
	config := map[string]any{}
	check(yaml.Unmarshal(configBytes, &config))
	config["BaseDir"] = root
	config["Endpoint"] = map[string]any{"Address": "127.0.0.1:18781"}
	config["MCP"] = map[string]any{"Address": "127.0.0.1:18790", "Authorization": map[string]any{"Global": map[string]any{"ProtectedResourceMetadata": map[string]any{"Resource": "http://127.0.0.1:18790/mcp"}}}}
	config["Connectors"] = []any{map[string]any{"Name": "studio", "Driver": "sqlite", "DSN": studioDSN(dir)}, map[string]any{"Name": "authz", "AliasOf": "studio"}}
	config["JWTValidator"] = map[string]any{"RSA": []any{map[string]any{"URL": filepath.Join(dir, "public.pem")}}}
	config["JWTClaims"] = map[string]any{"Issuer": "studio-e2e", "Audience": "studio-e2e", "RequireSubject": true}
	config["CORS"] = map[string]any{"AllowOrigins": []string{"http://127.0.0.1:18780"}, "AllowMethods": []string{"GET", "POST", "OPTIONS"}, "AllowHeaders": []string{"Content-Type", "Authorization", "X-Studio-Namespace"}}
	write(filepath.Join(dir, "native.json"), config)
	write(filepath.Join(dir, "studio-config.json"), map[string]any{"mode": "development", "apiBaseURL": "http://127.0.0.1:18780", "development": map[string]string{"subject": "owner"}})
	fmt.Println("prepared isolated Studio catalog and synthetic identities")
}
func serve(root, dir string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	check(os.Setenv("STUDIO_ACCESS_POLICY_EDITOR_ROLES", "ROLE_ADMIN"))
	check(os.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN"))
	identity, err := newIDP(dir)
	check(err)
	check(accessprovider.RegisterEnvironmentFactory(func(context.Context, accessprovider.Config) (authz.Provider, error) { return identity, nil }))
	_ = predicates.LinkedType
	check(os.Setenv("STUDIO_PREDICATE_PACKAGES", predicatePackage))
	catalog, err := (studiohost.Config{}).PredicateCatalog()
	check(err)
	if !catalog.Contains(predicatePackage, "CustomerScope") {
		log.Fatal("package-string predicate discovery failed")
	}
	check(os.Setenv("STUDIO_DYNAMIC_HTTP_URL", "http://127.0.0.1:18782"))
	check(os.Setenv("STUDIO_RUNTIME_ADMIN_TOKEN", "studio-e2e-admin"))
	group, err := runtimehost.NewNamespaceGroup(ctx, runtimehost.Config{HTTP: runtimehost.Listener{Address: "127.0.0.1:18782"}, MCP: runtimehost.Listener{Address: "127.0.0.1:0"}, Authentication: runtimehost.Authentication{DefaultMode: "required", Issuer: "studio-e2e", Audience: "studio-e2e"}, Access: &runtimehost.ResourceAccessConfig{Provider: identity, Tenant: "studio-e2e", Issuer: "studio-e2e", Audience: "studio-e2e", PublicKeyFile: filepath.Join(dir, "public.pem")}, Studio: runtimehost.Studio{Driver: "sqlite", DSN: studioDSN(dir)}, Admin: runtimehost.Admin{Token: "studio-e2e-admin"}, RootDir: root})
	check(err)
	check(group.StartAdmin("127.0.0.1:18782", "studio-e2e-admin"))
	defer group.Close(context.Background())
	go func() { check(group.Run(ctx, 100*time.Millisecond, nil)) }()
	go func() {
		if code := datlycmd.Run(ctx, []string{"run", "-conf", filepath.Join(dir, "native.json")}, os.Stdout, os.Stderr); code != 0 {
			cancel()
		}
	}()
	native, _ := url.Parse("http://127.0.0.1:18781")
	proxy := httputil.NewSingleHostReverseProxy(native)
	mux := http.NewServeMux()
	check(registerBrowserIdentity(mux, proxy, dir))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ready","predicatePackage":"`+predicatePackage+`"}`)
	})
	mux.Handle("/", http.FileServer(http.Dir(filepath.Join(root, "ui", "dist"))))
	server := &http.Server{Addr: "127.0.0.1:18780", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { <-ctx.Done(); server.Shutdown(context.Background()) }()
	fmt.Println("Studio e2e fixture listening on 18780, 18781, 18782, 18790")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		check(err)
	}
}
