package compiledartifact

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/resourceartifact"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk/executionguard"
	"github.com/viant/sqlx/testutil/sqlfault"
	_ "modernc.org/sqlite"
)

func contentDigest(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

// MySQL must be empty and owned, with the existing ANSI_QUOTES deployment mode.
// It must retain default changed-row reporting; no pool-wide workaround is used.
func TestArtifactCompareAndSwap(t *testing.T) {
	for _, driver := range []string{"sqlite", "mysql"} {
		t.Run(driver, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			dsn := os.Getenv("STUDIO_PREVIEW_ARTIFACT_MYSQL_DSN")
			if driver == "sqlite" {
				dsn = "file:" + filepath.Join(t.TempDir(), "metadata.db") + "?_txlock=immediate&_pragma=busy_timeout(5000)"
			} else {
				if dsn == "" {
					t.Skip("requires an empty owned STUDIO_PREVIEW_ARTIFACT_MYSQL_DSN")
				}
				parsed, err := mysql.ParseDSN(dsn)
				if err != nil {
					t.Fatal(err)
				}
				if parsed.ClientFoundRows {
					t.Fatal("requires default MySQL changed-row semantics")
				}
			}
			db, err := sql.Open(driver, dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if driver == "mysql" {
				var count int
				if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&count); err != nil || count != 0 {
					t.Fatalf("fixture is not empty: %d %v", count, err)
				}
			}
			if err = schema.Initialize(ctx, db, driver); err != nil {
				t.Fatal(err)
			}
			for _, statement := range []string{
				`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('main','sqlite','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','general','General','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES('reader','reader','Reader','owner','draft','main','example.com/proof/reader','reader',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',1,'draft','dql','{}','1','hash','{}','pending','test','test',1,'owner',CURRENT_TIMESTAMP)`,
			} {
				if _, err = db.ExecContext(ctx, statement); err != nil {
					t.Fatal(err)
				}
			}
			owner := namespaceaccess.ID("owner", "general")
			store, err := NewStore(db)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close(context.WithoutCancel(ctx))
			identity := func(name string) resourceartifact.Identity {
				t.Helper()
				id, err := resourceartifact.New("preview://proof/reader@1", contentDigest([]byte(name)))
				if err != nil {
					t.Fatal(err)
				}
				return id
			}
			for _, absent := range []bool{false, true} {
				t.Run(fmt.Sprintf("race/absent=%v", absent), func(t *testing.T) {
					id := identity(t.Name())
					var expected *string
					if !absent {
						if err := store.CompareAndSwap(ctx, owner, "reader", 1, id, []byte("original"), nil); err != nil {
							t.Fatal(err)
						}
						hash := contentDigest([]byte("original"))
						expected = &hash
					}
					arrived, release := make(chan struct{}, 2), make(chan struct{})
					results := make(chan error, 2)
					var predicates atomic.Int32
					for i := 0; i < 2; i++ {
						observed := sql.OpenDB(&sqlfault.Connector{Base: db.Driver(), DSN: dsn, Before: func(ctx context.Context, call sqlfault.Call) error {
							if call.Phase == "prepare" && strings.HasPrefix(strings.ToLower(call.SQL), "update ") {
								where := strings.Index(strings.ToLower(call.SQL), "where")
								if where < 0 || !strings.Contains(strings.ToLower(call.SQL[where:]), "content_sha256") {
									return errors.New("digest missing from database predicate")
								}
								predicates.Add(1)
							}
							if driver == "mysql" && call.Phase == "next" && strings.Contains(call.SQL, "component_resource_files") && ((!absent && call.Row == 1) || (absent && call.Row == 0)) {
								arrived <- struct{}{}
								select {
								case <-release:
								case <-ctx.Done():
									return ctx.Err()
								}
							}
							return nil
						}})
						t.Cleanup(func() { observed.Close() })
						independent, err := NewStore(observed)
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() { independent.Close(context.WithoutCancel(ctx)) })
						go func(i int) {
							results <- independent.CompareAndSwap(ctx, owner, "reader", 1, id, []byte(fmt.Sprintf("replacement-%d", i)), expected)
						}(i)
					}
					if driver == "mysql" {
						for i := 0; i < 2; i++ {
							select {
							case <-arrived:
							case <-ctx.Done():
								close(release)
								t.Fatal("race did not reach both prerequisite reads")
							}
						}
					}
					close(release)
					successes, failures := 0, 0
					for i := 0; i < 2; i++ {
						if err := <-results; err == nil {
							successes++
						} else {
							failures++
						}
					}
					if successes != 1 || failures != 1 {
						t.Fatalf("success/failure=%d/%d", successes, failures)
					}
					if driver == "mysql" && !absent && predicates.Load() != 2 {
						t.Fatalf("guarded SQL attempts=%d", predicates.Load())
					}
					payload, err := store.Get(ctx, "reader", 1, id)
					if err != nil || !strings.HasPrefix(string(payload), "replacement-") {
						t.Fatalf("persisted winner: %s %v", payload, err)
					}
				})
			}
			t.Run("expected-state", func(t *testing.T) {
				id := identity(t.Name())
				if err := store.CompareAndSwap(ctx, owner, "reader", 1, id, []byte("original"), nil); err != nil {
					t.Fatal(err)
				}
				wrong := contentDigest([]byte("stale"))
				correct := contentDigest([]byte("original"))
				for _, expected := range []*string{nil, &wrong} {
					if err := store.CompareAndSwap(ctx, owner, "reader", 1, id, []byte("bad"), expected); err == nil {
						t.Fatal("stale/absent expectation accepted")
					}
				}
				if err := store.CompareAndSwap(ctx, owner, "reader", 1, identity("missing"), []byte("bad"), &correct); err == nil {
					t.Fatal("expected prior digest inserted a missing row")
				}
			})
			for _, scenario := range []string{"revoked", "storage-failure", "canceled"} {
				t.Run(scenario, func(t *testing.T) {
					id := identity(t.Name())
					if err := store.CompareAndSwap(ctx, owner, "reader", 1, id, []byte("original"), nil); err != nil {
						t.Fatal(err)
					}
					var revoked atomic.Bool
					observed := sql.OpenDB(&sqlfault.Connector{Base: db.Driver(), DSN: dsn, Before: func(ctx context.Context, call sqlfault.Call) error {
						if call.Phase == "prepare" && strings.HasPrefix(strings.ToLower(call.SQL), "update ") {
							if scenario == "revoked" {
								revoked.Store(true)
							}
							if scenario == "storage-failure" {
								return errors.New("owned storage failure")
							}
						}
						return nil
					}})
					defer observed.Close()
					independent, err := NewStore(observed)
					if err != nil {
						t.Fatal(err)
					}
					defer independent.Close(context.WithoutCancel(ctx))
					callCtx := executionguard.With(ctx, func(context.Context) error {
						if revoked.Load() {
							return errors.New("owned revocation")
						}
						return nil
					})
					if scenario == "canceled" {
						var cancel context.CancelFunc
						callCtx, cancel = context.WithCancel(callCtx)
						cancel()
					}
					hash := contentDigest([]byte("original"))
					if err := independent.CompareAndSwap(callCtx, owner, "reader", 1, id, []byte("replacement"), &hash); err == nil {
						t.Fatal("failure succeeded")
					}
					payload, err := store.Get(ctx, "reader", 1, id)
					if err != nil || string(payload) != "original" {
						t.Fatalf("failure changed artifact: %s %v", payload, err)
					}
					if observed.Stats().InUse != 0 {
						t.Fatal("failed write leaked a connection")
					}
				})
			}
		})
	}
}
