package bffauth

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/store/sql/migrate"
	_ "modernc.org/sqlite"
)

func TestDatlySessionStorePrunesExpiredAcrossBatches(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	cutoff := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 260; i++ {
		_, err := db.ExecContext(ctx, `INSERT INTO bff_sessions(session_id_hash,subject_id,payload_ciphertext,expires_at_unix,created_at) VALUES(?,?,?,?,?)`,
			fmt.Sprintf("%064x", i), "expired", []byte("ciphertext"), cutoff.Add(-time.Minute).Unix(), cutoff.Add(-time.Hour))
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.ExecContext(ctx, `INSERT INTO bff_sessions(session_id_hash,subject_id,payload_ciphertext,expires_at_unix,created_at) VALUES(?,?,?,?,?)`,
		fmt.Sprintf("%064x", 300), "active", []byte("ciphertext"), cutoff.Add(time.Minute).Unix(), cutoff.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLStore(db, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	if err := store.DeleteExpired(ctx, cutoff); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteExpired(ctx, cutoff); err != nil {
		t.Fatalf("idempotent cleanup: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bff_sessions WHERE expires_at_unix<=?`, cutoff.Unix()).Scan(&count); err != nil || count != 0 {
		t.Fatalf("expired rows=%d err=%v", count, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bff_sessions WHERE subject_id='active'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("active rows=%d err=%v", count, err)
	}
}

func TestDatlySessionStoreConcurrentIsolatedSessions(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLStore(db, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	const workers = 8
	var group sync.WaitGroup
	errors := make(chan error, workers)
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			id, subject := fmt.Sprintf("session-%d", i), fmt.Sprintf("principal-%d", i)
			if err := store.Put(ctx, id, session{principal: sdk.Principal{Subject: subject}, token: subject + "-token", expiresAt: time.Now().Add(time.Hour)}); err != nil {
				errors <- err
				return
			}
			got, found, err := store.Get(ctx, id)
			if err != nil || !found || got.principal.Subject != subject || got.token != subject+"-token" {
				errors <- fmt.Errorf("%s: found=%v subject=%q token=%q err=%v", id, found, got.principal.Subject, got.token, err)
				return
			}
			if err := store.Delete(ctx, id); err != nil {
				errors <- err
			}
		}(i)
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestDatlySessionStoreRetainsPutReplacementAndIdempotentDelete(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLStore(db, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	expires := time.Now().Add(time.Hour)
	var originalCreated time.Time
	for _, name := range []string{"first", "replacement"} {
		if err := store.Put(ctx, "stable-cookie", session{principal: sdk.Principal{Subject: name}, token: name + "-token", expiresAt: expires}); err != nil {
			t.Fatalf("put %s: %v", name, err)
		}
		var created time.Time
		if err := db.QueryRowContext(ctx, `SELECT created_at FROM bff_sessions WHERE session_id_hash=?`, sessionHash("stable-cookie")).Scan(&created); err != nil {
			t.Fatal(err)
		}
		if name == "first" {
			originalCreated = created
		} else if !created.Equal(originalCreated) {
			t.Fatalf("replacement changed created_at: first=%v replacement=%v", originalCreated, created)
		}
	}
	got, found, err := store.Get(ctx, "stable-cookie")
	if err != nil || !found || got.principal.Subject != "replacement" || got.token != "replacement-token" {
		t.Fatalf("replacement session=%+v found=%v err=%v", got, found, err)
	}
	if err := store.Delete(ctx, "stable-cookie"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "stable-cookie"); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
	if _, found, err := store.Get(ctx, "stable-cookie"); err != nil || found {
		t.Fatalf("deleted session found=%v err=%v", found, err)
	}
}
