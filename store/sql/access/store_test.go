package access

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acl "github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly-studio/store/sql/migrate"
	_ "modernc.org/sqlite"
)

func TestPolicyPersistenceAndAtomicHistory(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "policies.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m, _ := migrate.New()
	if err = m.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := &Store{DB: db}
	r := acl.Resource{Kind: "component", ID: "shared-id", Version: "1", Tenant: "one"}
	d := acl.Document{Resource: r, Policies: map[string]acl.Policy{"describe": {Mode: "public"}}}
	d, err = s.Provision(ctx, d, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Provision(ctx, d, "admin"); !errors.Is(err, acl.ErrConflict) {
		t.Fatalf("bootstrap overwrote: %v", err)
	}
	d.Policies["execute"] = acl.Policy{Mode: "public"}
	next, err := s.Replace(ctx, d, 1, "publisher")
	if err != nil || next.Revision != 2 {
		t.Fatalf("replace: %+v %v", next, err)
	}
	if _, err = s.Replace(ctx, d, 1, "stale"); !errors.Is(err, acl.ErrConflict) {
		t.Fatalf("stale writer: %v", err)
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM resource_policy_revisions`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history count=%d err=%v", count, err)
	}
	for _, other := range []acl.Resource{
		{Kind: "skill", ID: r.ID, Version: r.Version, Tenant: r.Tenant},
		{Kind: r.Kind, ID: r.ID, Version: r.Version, Tenant: "two"},
		{Kind: r.Kind, ID: r.ID, Version: "2", Tenant: r.Tenant},
	} {
		if _, err = s.Get(ctx, other); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("resource boundary crossed")
		}
	}
	// Force revision recording to fail after CAS; the head must roll back too.
	if _, err = db.Exec(`CREATE TRIGGER reject_policy_history BEFORE INSERT ON resource_policy_revisions BEGIN SELECT RAISE(ABORT,'history unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Replace(ctx, next, 2, "publisher"); err == nil {
		t.Fatal("expected failed audit write")
	}
	loaded, err := s.Get(ctx, r)
	if err != nil || loaded.Revision != 2 {
		t.Fatalf("CAS escaped transaction: %+v %v", loaded, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM resource_policy_revisions`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("rolled back activation left history: count=%d err=%v", count, err)
	}
	if _, err = db.Exec(`DROP TRIGGER reject_policy_history`); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

// TestPolicyActivationRunsAsGeneratedComponents proves the store's behavior is
// produced by the Datly resource policy components and their lifecycle rules
// rather than by adapter SQL: audit fields, denial before mutation, concurrent
// stale writers from one base revision, and replacing a missing resource.
func TestPolicyActivationRunsAsGeneratedComponents(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "policies.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m, _ := migrate.New()
	if err = m.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := &Store{DB: db}
	t.Cleanup(func() { _ = s.Close(ctx) })
	r := acl.Resource{Kind: "component", ID: "records", Version: "3", Tenant: "one"}
	public := map[string]acl.Policy{"execute": {Mode: "public"}}

	t.Run("replace before provisioning conflicts and writes nothing", func(t *testing.T) {
		if _, err := s.Replace(ctx, acl.Document{Resource: r, Revision: 1, Policies: public}, 1, "eager"); !errors.Is(err, acl.ErrConflict) {
			t.Fatalf("replace of missing resource: %v", err)
		}
		var heads, history int
		if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM resource_policy_heads),(SELECT COUNT(*) FROM resource_policy_revisions)`).Scan(&heads, &history); err != nil || heads != 0 || history != 0 {
			t.Fatalf("rows heads=%d history=%d err=%v", heads, history, err)
		}
	})
	t.Run("invalid policy is denied before any row exists", func(t *testing.T) {
		for name, doc := range map[string]acl.Document{
			"public management":  {Resource: r, Policies: map[string]acl.Policy{"manageAccess": {Mode: "public"}}},
			"unknown mode":       {Resource: r, Policies: map[string]acl.Policy{"execute": {Mode: "open"}}},
			"protected no rule":  {Resource: r, Policies: map[string]acl.Policy{"execute": {Mode: "protected"}}},
			"empty policies":     {Resource: r},
			"missing tenant":     {Resource: acl.Resource{Kind: "component", ID: "records", Version: "3"}, Policies: public},
			"missing kind":       {Resource: acl.Resource{ID: "records", Version: "3", Tenant: "one"}, Policies: public},
			"missing version":    {Resource: acl.Resource{Kind: "component", ID: "records", Tenant: "one"}, Policies: public},
			"missing identifier": {Resource: acl.Resource{Kind: "component", Version: "3", Tenant: "one"}, Policies: public},
		} {
			if _, err := s.Provision(ctx, doc, "bootstrap"); err == nil {
				t.Fatalf("%s accepted", name)
			}
		}
		if _, err := s.Provision(ctx, acl.Document{Resource: r, Policies: public}, ""); !errors.Is(err, acl.ErrDenied) {
			t.Fatalf("anonymous actor: %v", err)
		}
		var heads int
		if err := db.QueryRow(`SELECT COUNT(*) FROM resource_policy_heads`).Scan(&heads); err != nil || heads != 0 {
			t.Fatalf("denied activation wrote heads=%d err=%v", heads, err)
		}
	})
	t.Run("history records actor and time per revision", func(t *testing.T) {
		before := time.Now().UTC().Add(-time.Second)
		if _, err := s.Provision(ctx, acl.Document{Resource: r, Policies: public}, "bootstrap"); err != nil {
			t.Fatal(err)
		}
		next, err := s.Replace(ctx, acl.Document{Resource: r, Revision: 1, Policies: map[string]acl.Policy{"execute": {Mode: "protected", Rule: &acl.Rule{Kind: "role", Value: "reader"}}}}, 1, "publisher")
		if err != nil || next.Revision != 2 {
			t.Fatalf("replace: %+v %v", next, err)
		}
		rows, err := db.Query(`SELECT revision,actor_id,occurred_at,policies_json FROM resource_policy_revisions WHERE tenant_id='one' AND resource_kind='component' AND resource_id='records' AND resource_version='3' ORDER BY revision`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var seen []int64
		for rows.Next() {
			var revision int64
			var actor, occurred, policies string
			if err := rows.Scan(&revision, &actor, &occurred, &policies); err != nil {
				t.Fatal(err)
			}
			at, parseErr := time.Parse(time.RFC3339Nano, strings.Replace(strings.Replace(occurred, " ", "T", 1), "+00:00", "Z", 1))
			if parseErr != nil {
				at, parseErr = time.Parse("2006-01-02 15:04:05.999999999", occurred)
			}
			if parseErr != nil || at.Before(before) {
				t.Fatalf("occurred_at %q: %v", occurred, parseErr)
			}
			want := map[int64]string{1: "bootstrap", 2: "publisher"}[revision]
			if actor != want || !strings.Contains(policies, `"mode"`) {
				t.Fatalf("revision %d actor=%q policies=%s", revision, actor, policies)
			}
			seen = append(seen, revision)
		}
		if len(seen) != 2 || seen[0] != 1 || seen[1] != 2 {
			t.Fatalf("history revisions=%v", seen)
		}
		loaded, err := s.Get(ctx, r)
		if err != nil || loaded.Revision != 2 || loaded.Policies["execute"].Mode != "protected" || loaded.Policies["execute"].Rule == nil || loaded.Policies["execute"].Rule.Value != "reader" {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})
	t.Run("concurrent writers from one base revision", func(t *testing.T) {
		base, err := s.Get(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		first := base
		first.Policies = map[string]acl.Policy{"execute": {Mode: "public"}}
		second := base
		second.Policies = map[string]acl.Policy{"describe": {Mode: "public"}}
		winner, err := s.Replace(ctx, first, base.Revision, "first")
		if err != nil || winner.Revision != base.Revision+1 {
			t.Fatalf("first writer: %+v %v", winner, err)
		}
		if _, err = s.Replace(ctx, second, base.Revision, "second"); !errors.Is(err, acl.ErrConflict) {
			t.Fatalf("second writer from the same base must conflict: %v", err)
		}
		var count int
		if err = db.QueryRow(`SELECT COUNT(*) FROM resource_policy_revisions WHERE actor_id='second'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("stale writer left history: count=%d err=%v", count, err)
		}
		if err = db.QueryRow(`SELECT COUNT(*) FROM resource_policy_revisions`).Scan(&count); err != nil || count != 3 {
			t.Fatalf("history count=%d err=%v", count, err)
		}
		loaded, err := s.Get(ctx, r)
		if err != nil || loaded.Revision != 3 || loaded.Policies["execute"].Mode != "public" || len(loaded.Policies) != 1 {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})
	t.Run("returned document is detached from persisted state", func(t *testing.T) {
		loaded, err := s.Get(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		loaded.Policies["execute"] = acl.Policy{Mode: "protected", Rule: &acl.Rule{Kind: "subject", Value: "mallory"}}
		again, err := s.Get(ctx, r)
		if err != nil || again.Policies["execute"].Mode != "public" {
			t.Fatalf("caller mutation leaked into store: %+v %v", again, err)
		}
	})
}
