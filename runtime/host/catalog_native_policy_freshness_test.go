package host

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/viant/authz"
	policyschema "github.com/viant/authz/component/schema"
	policystore "github.com/viant/authz/component/store/sql"
)

// Exercise the pinned policy store, including writes invisible to its process-
// local invalidation registry. No test replaces policy lookup or evaluation.
func TestNativePolicyFreshnessAcrossProcesses(t *testing.T) {
	ctx := context.Background()
	resource := authz.Resource{Kind: "component", ID: "fixture-reader", Version: "1", Tenant: "fixture"}
	if path := os.Getenv("STUDIO_POLICY_FRESHNESS_CHILD_DB"); path != "" {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		store := &policystore.Store{DB: db}
		defer store.Close(ctx)
		doc, err := store.Get(ctx, resource)
		if err != nil {
			t.Fatal(err)
		}
		doc.Policies["execute"] = authz.Policy{Mode: "deny"}
		if _, err = store.Replace(ctx, doc, doc.Revision, "fixture-child"); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "policies.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migration, err := policyschema.New("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if err = migration.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	reader := &policystore.Store{DB: db}
	defer reader.Close(ctx)
	if _, err = reader.Get(ctx, resource); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("absent: %v", err)
	}
	other, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	writer := &policystore.Store{DB: other}
	defer writer.Close(ctx)
	doc, err := writer.Provision(ctx, authz.Document{Resource: resource, Policies: map[string]authz.Policy{"execute": {Mode: "public"}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if loaded, err := reader.Get(ctx, resource); err != nil || loaded.Revision != doc.Revision {
		t.Fatalf("negative-to-created across pools: %+v %v", loaded, err)
	}
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativePolicyFreshnessAcrossProcesses$", "-test.count=1")
	command.Env = append(os.Environ(), "STUDIO_POLICY_FRESHNESS_CHILD_DB="+path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("child native replacement: %v %s", err, output)
	}
	loaded, err := reader.Get(ctx, resource)
	if err != nil || loaded.Revision != doc.Revision+1 || loaded.Policies["execute"].Mode != "deny" {
		t.Fatalf("cross-process revocation: %+v %v", loaded, err)
	}
	// A caller-owned unit must stay private until its actual commit and must not
	// publish an aborted replacement through a prepared reader.
	for _, commit := range []bool{false, true} {
		tx, err := other.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		transactional, err := policystore.NewTransactionStore(ctx, other, tx)
		if err != nil {
			t.Fatal(err)
		}
		draft := loaded
		draft.Policies = map[string]authz.Policy{"execute": {Mode: "public"}}
		_, err = transactional.Replace(ctx, draft, draft.Revision, "fixture-transaction")
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if err = transactional.Close(ctx); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if commit {
			err = tx.Commit()
		} else {
			err = tx.Rollback()
		}
		if err != nil {
			t.Fatal(err)
		}
		got, err := reader.Get(ctx, resource)
		if err != nil {
			t.Fatal(err)
		}
		want := loaded.Revision
		mode := "deny"
		if commit {
			want++
			mode = "public"
		}
		if got.Revision != want || got.Policies["execute"].Mode != mode {
			t.Fatalf("commit=%t: %+v", commit, got)
		}
	}
}
