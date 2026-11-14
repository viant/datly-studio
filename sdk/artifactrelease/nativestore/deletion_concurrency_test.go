package nativestore_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/componentremoval/removaltest"
	"github.com/viant/datly-studio/internal/datatest"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// The DSN must identify an empty database in an exclusively owned MySQL 8.4
// instance with ANSI_QUOTES and performance_schema enabled. SQL below is only
// fixture setup, deterministic scheduling, and independent postcondition reads.
func TestNativeReleaseStageDeletionRaceMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_RELEASE_DELETION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires an empty task-owned STUDIO_RELEASE_DELETION_MYSQL_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	f := removaltest.NewMySQL(ctx, t, dsn)
	for _, stageFirst := range []bool{true, false} {
		name := "delete-before-stage"
		if stageFirst {
			name = "stage-before-delete"
		}
		t.Run(name, func(t *testing.T) {
			f.T = t
			id := name
			f.Exec(ctx, `INSERT INTO components(id,namespace_id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,current_draft_version,last_version_no,created_at,updated_at) VALUES(?,?,'alpha',?,?,'alice','active','main',?,?,1,1,?,?)`, id, f.Namespace, id, id, "reader/"+id, id, removaltest.Now, removaltest.Now)
			f.Exec(ctx, `INSERT INTO component_versions(namespace_id,report_id,version_no,state,authoring_mode,component_spec_json,type_manifest_json,spec_format_version,spec_hash,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,?,1,'draft','dql','{}','{}','1',?,'valid','v1','v1',7,'alice',?)`, f.Namespace, id, strings.Repeat("d", 64), removaltest.Now)
			scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "alpha"}
			store, err := datatest.NativeReleaseStore(t, f.DB, scope, "alice")
			if err != nil {
				t.Fatal(err)
			}
			current, err := contract.NewChannel(contract.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: name}, "Race", []authz.NamedReference{{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}})
			if err != nil {
				t.Fatal(err)
			}
			root := artifactcohort.RootRef{URI: "window://alpha/" + name, Kind: "window", Version: 1, Hash: strings.Repeat("a", 64)}
			bundle := contract.Bundle{Namespace: "alpha", Roots: []artifactcohort.RootRef{root}}
			bundle.Hash = contract.BundleHash(bundle)
			if err = store.PutBundle(ctx, bundle); err != nil {
				t.Fatal(err)
			}
			if err = store.CreateChannel(ctx, current); err != nil {
				t.Fatal(err)
			}
			next, err := contract.PreparePromotion(current, 0, bundle)
			if err != nil {
				t.Fatal(err)
			}
			binding := releasevalue.ComponentBinding{Root: root, DataSourceID: "rows", ComponentKind: "dynamic", ComponentID: id, ComponentRevision: "1", Method: http.MethodGet, Route: "/rows", SchemaFingerprint: strings.Repeat("b", 64), ContentFingerprint: strings.Repeat("c", 64)}
			gateName := "release-removal-" + name
			gate, err := f.DB.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer gate.Close()
			var acquired int
			if err = gate.QueryRowContext(ctx, "SELECT GET_LOCK(?,0)", gateName).Scan(&acquired); err != nil || acquired != 1 {
				t.Fatalf("acquire fixture gate: %d %v", acquired, err)
			}
			released := false
			release := func() {
				t.Helper()
				if released {
					return
				}
				if err := gate.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", gateName).Scan(&acquired); err != nil || acquired != 1 {
					t.Fatalf("release fixture gate: %d %v", acquired, err)
				}
				released = true
			}
			defer release()
			timing, table, condition := "BEFORE DELETE", "component_versions", "OLD.report_id='"+id+"'"
			if stageFirst {
				timing, table, condition = "BEFORE INSERT", "artifact_release_bundle_bindings", "NEW.bundle_hash='"+bundle.Hash+"'"
			}
			f.Exec(ctx, fmt.Sprintf(`CREATE TRIGGER pause_release_removal %s ON %s FOR EACH ROW BEGIN IF %s THEN IF GET_LOCK('%s',20) <> 1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='fixture gate timed out'; END IF; DO RELEASE_LOCK('%s'); END IF; END`, timing, table, condition, gateName, gateName))
			defer func() { release(); f.Exec(ctx, "DROP TRIGGER pause_release_removal") }()
			staged := make(chan error, 1)
			finished := make(chan string, 2)
			deleted := make(chan raceDeleteResponse, 1)
			stage := func() {
				_, err := store.StageRevisionWithBindings(ctx, current, next, bundle, []releasevalue.ComponentBinding{binding})
				staged <- err
				finished <- fmt.Sprintf("stage finished before fixture release: %v", err)
			}
			remove := func() {
				response := raceDelete(ctx, f, id)
				deleted <- response
				finished <- fmt.Sprintf("delete finished before fixture release: status=%d body=%s err=%v", response.code, response.body, response.err)
			}
			if stageFirst {
				go stage()
			} else {
				go remove()
			}
			waitRaceCondition(ctx, t, f.DB, finished, "first native operation reached its fixture gate", `SELECT COUNT(*) FROM performance_schema.metadata_locks WHERE OBJECT_TYPE='USER LEVEL LOCK' AND OBJECT_NAME=? AND LOCK_STATUS='PENDING'`, gateName)
			if stageFirst {
				go remove()
			} else {
				go stage()
			}
			// Observe a real row-lock wait, not elapsed wall time or goroutine order.
			waitRaceCondition(ctx, t, f.DB, finished, "competing native operation waits for the same physical parent", `SELECT COUNT(*) FROM performance_schema.data_lock_waits w JOIN performance_schema.data_locks l ON l.ENGINE_LOCK_ID=w.BLOCKING_ENGINE_LOCK_ID AND l.ENGINE=w.ENGINE WHERE l.OBJECT_SCHEMA=DATABASE() AND l.OBJECT_NAME='components' AND l.INDEX_NAME='PRIMARY' AND l.LOCK_DATA=?`, "'"+id+"'")
			t.Log("observed both native operations contending on the physical component parent")
			select {
			case err := <-staged:
				t.Fatalf("stage finished before gate release: %v", err)
			default:
			}
			select {
			case response := <-deleted:
				t.Fatalf("delete finished before gate release: %+v", response)
			default:
			}
			release()
			var stageErr error
			select {
			case stageErr = <-staged:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var deletion raceDeleteResponse
			select {
			case deletion = <-deleted:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if deletion.err != nil {
				t.Fatal(deletion.err)
			}
			if stageFirst {
				if stageErr != nil {
					t.Fatalf("stage winner failed: %v", stageErr)
				}
				if deletion.code != 409 || !strings.Contains(strings.ToLower(string(deletion.body)), "release") {
					t.Fatalf("delete did not reject retained JSON: %d %s", deletion.code, deletion.body)
				}
				stored, err := store.BundleBindings(ctx, bundle)
				if err != nil || len(stored) != 1 || stored[0].ComponentID != id || stored[0].ComponentRevision != "1" {
					t.Fatalf("winning binding=%+v err=%v", stored, err)
				}
				f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=1", 1, id)
				f.Count(ctx, "SELECT COUNT(*) FROM artifact_release_operations WHERE scope_id=? AND channel_name=? AND status='pending'", 1, scope.ID(), name)
			} else {
				if deletion.code != 200 {
					t.Fatalf("delete winner failed: %d %s", deletion.code, deletion.body)
				}
				f.Result(deletion.body, id, 1, 1)
				if !errors.Is(stageErr, contract.ErrConflict) {
					t.Fatalf("stage accepted deleted source or failed outside its lifecycle: %v", stageErr)
				}
				f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id=?", 0, id)
				f.Count(ctx, "SELECT COUNT(*) FROM artifact_release_bundle_bindings WHERE scope_id=? AND bundle_hash=?", 0, scope.ID(), bundle.Hash)
				f.Count(ctx, "SELECT COUNT(*) FROM artifact_release_operations WHERE scope_id=? AND channel_name=?", 0, scope.ID(), name)
				f.Count(ctx, "SELECT COUNT(*) FROM artifact_release_channels WHERE scope_id=? AND name=? AND revision=0 AND pending_operation_id IS NULL", 1, scope.ID(), name)
			}
			// Check the JSON contents against actual surviving versions, including the
			// binding retained by the earlier stage-first subtest.
			rows, err := f.DB.QueryContext(ctx, "SELECT bindings_json FROM artifact_release_bundle_bindings")
			if err != nil {
				t.Fatal(err)
			}
			var references []releasevalue.ComponentBinding
			for rows.Next() {
				var raw []byte
				if err = rows.Scan(&raw); err != nil {
					t.Fatal(err)
				}
				var bindings []releasevalue.ComponentBinding
				if err = json.Unmarshal(raw, &bindings); err != nil {
					t.Fatal(err)
				}
				references = append(references, bindings...)
			}
			if err = rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
			for _, reference := range references {
				if reference.ComponentKind == "dynamic" {
					f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id=? AND version_no=?", 1, reference.ComponentID, reference.ComponentRevision)
				}
			}
		})
	}
}

type raceDeleteResponse struct {
	code int
	body []byte
	err  error
}

func raceDelete(ctx context.Context, f *removaltest.Fixture, id string) raceDeleteResponse {
	body := fmt.Sprintf(`{"reportId":%q,"versionNo":1,"expectedSourceRevision":7}`, id)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+f.Addresses[0]+"/v1/studio/sdk/versions.delete", strings.NewReader(body))
	if err != nil {
		return raceDeleteResponse{err: err}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", f.Token("alice"))
	request.Header.Set("X-Studio-Namespace", f.Namespace)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return raceDeleteResponse{err: err}
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	return raceDeleteResponse{code: response.StatusCode, body: raw, err: err}
}
func waitRaceCondition(ctx context.Context, t *testing.T, db *sql.DB, finished <-chan string, description, query string, args ...any) {
	t.Helper()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case result := <-finished:
			t.Fatalf("%s: %s", description, result)
		default:
		}
		var count int
		if err := db.QueryRowContext(timeout, query, args...).Scan(&count); err != nil {
			t.Fatalf("%s: %v", description, err)
		}
		if count > 0 {
			return
		}
		select {
		case result := <-finished:
			t.Fatalf("%s: %s", description, result)
		case <-timeout.Done():
			t.Fatalf("%s: %v", description, timeout.Err())
		case <-ticker.C:
		}
	}
}
