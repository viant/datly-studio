package studioapi

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/schema"
)

type databaseModuleCloser struct {
	db     *sql.DB
	closed int
}

func (m *databaseModuleCloser) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (m *databaseModuleCloser) Close(ctx context.Context) error {
	m.closed++
	return m.db.PingContext(ctx)
}

func TestServeClosesModuleOnMountFailureBeforeBorrowedDatabase(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/module.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	module := &databaseModuleCloser{db: db}
	err = Serve(ctx, []string{"--mode", "authenticated", "--allowed-origin", "http://localhost:8095", "--dynamic-admin-token", "test-token", "--jwt-issuer", "https://issuer.example", "--jwt-audience", "studio"}, Options{
		Database: db, DatabaseDriver: "sqlite",
		SchemaVerifier: func(ctx context.Context, actual *sql.DB) error { return actual.PingContext(ctx) },
		AccessProviderFactory: func(context.Context, accessprovider.Config) (authz.Provider, error) {
			return appFactoryProvider{subject: "editor"}, nil
		},
		Modules: func(context.Context, *sql.DB, authz.Provider) ([]Module, error) {
			return []Module{{Path: "/mcp", Handler: module}}, nil
		},
	})
	if err == nil || module.closed != 1 {
		t.Fatalf("startup error=%v module close count=%d", err, module.closed)
	}
	if err = db.PingContext(ctx); err != nil {
		t.Fatalf("borrowed DB closed before module cleanup: %v", err)
	}
}

func TestServeIsolatesArgumentsAndReturnsFailures(t *testing.T) {
	originalArgs := append([]string(nil), os.Args...)
	originalFlags := flag.CommandLine
	for i := 0; i < 2; i++ {
		if err := Serve(context.Background(), []string{"--help"}, Options{}); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("help: %v", err)
		}
		if err := Serve(context.Background(), []string{"--mode", "invalid"}, Options{}); err == nil {
			t.Fatal("invalid mode accepted")
		}
		if err := Serve(context.Background(), []string{"unexpected"}, Options{}); err == nil {
			t.Fatal("positional argument accepted")
		}
	}
	if !reflect.DeepEqual(os.Args, originalArgs) || flag.CommandLine != originalFlags {
		t.Fatal("Serve modified global CLI state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, nil, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestServeClosesDatabaseOnStartupFailure(t *testing.T) {
	rejected := errors.New("schema rejected")
	var opened *sql.DB
	err := Serve(context.Background(), []string{"--dsn", "file:" + t.TempDir() + "/studio.db"}, Options{
		SchemaVerifier: func(ctx context.Context, db *sql.DB) error {
			opened = db
			if err := db.PingContext(ctx); err != nil {
				return err
			}
			return rejected
		},
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("startup: %v", err)
	}
	if opened == nil {
		t.Fatal("schema verifier not called")
	}
	if err := opened.Ping(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("database was not closed: %v", err)
	}
}

func TestServePreservesCallerOwnedDatabaseOnFailure(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/shared.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rejected := errors.New("shared schema rejected")
	err = Serve(context.Background(), nil, Options{Database: db, DatabaseDriver: "sqlite",
		SchemaVerifier: func(ctx context.Context, actual *sql.DB) error {
			if actual != db {
				t.Fatal("host opened a second catalog connection")
			}
			return rejected
		},
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("startup: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("caller-owned connection closed: %v", err)
	}
}
