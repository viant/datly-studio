package studioapi

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
)

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
