package host

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	_ "modernc.org/sqlite"
)

func TestCompiledReaderRestartHitChangedSourceAndFreshBusinessRows(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runtime\n\ngo 1.25.0\n"), 0600))
	sourceDSN := "file:" + filepath.Join(root, "source.db")
	source, err := sql.Open("sqlite", sourceDSN)
	require.NoError(t, err)
	defer source.Close()
	_, err = source.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT); INSERT INTO records VALUES(1,'first')`)
	require.NoError(t, err)
	studioDSN := "file:" + filepath.Join(root, "studio.db")
	studio, err := sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	require.NoError(t, schema.ApplySQLite(ctx, studio, "studio"))
	now := time.Now().UTC()
	_, err = studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active','{}',1,?,?)`, sourceDSN, now, now)
	require.NoError(t, err)
	for _, item := range []struct{ id, path, name string }{{"records", "/records", "Record"}, {"other", "/other", "OtherRecord"}} {
		_, err = studio.Exec(`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,'owner','active','source',?, ?,1,?,?)`, item.id, item.id, item.id, "example.com/runtime/"+item.id, item.id, now, now)
		require.NoError(t, err)
		dql := fmt.Sprintf(`#package('example.com/runtime/%s')
#setting($_ = $connector('source'))
#setting($_ = $route('%s','GET'))
#define($_ = $Records<[]*%s>(output/view))
SELECT records.*, type(records,'%s') FROM (SELECT id,name FROM records) records`, item.id, item.path, item.name, item.name)
		_, err = studio.Exec(`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,1,'published','dql',?,?,'{}','studio.v1','hash','{}','valid','v1','studio.v1',1,'owner',?)`, item.id, dql, dql, now)
		require.NoError(t, err)
		_, err = studio.Exec(`INSERT INTO component_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES(?,1,1,1,'active','fixture','hash','owner',?,?)`, item.id, now, now)
		require.NoError(t, err)
	}
	_, err = studio.Exec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'fixture','active',2,'{}','owner',?,?)`, now, now)
	require.NoError(t, err)
	require.NoError(t, studio.Close())
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: studioDSN}, Admin: Admin{Token: "test-token"}, RootDir: root}
	start := func() (*Service, *atomic.Int32) {
		service, err := New(ctx, config)
		require.NoError(t, err)
		count := &atomic.Int32{}
		service.compileRuntimeContract = func(ctx context.Context, source *transcribe.Source) (*transcribe.RuntimeContract, error) {
			count.Add(1)
			return transcribe.NewCompiler().RuntimeContracts(ctx, root, source)
		}
		require.NoError(t, service.Start(ctx))
		return service, count
	}
	closeService := func(service *Service) { require.NoError(t, service.Close(ctx)) }
	first, count := start()
	require.EqualValues(t, 2, count.Load(), "first publication compiles both readers")
	closeService(first)
	_, err = source.Exec(`UPDATE records SET name='fresh' WHERE id=1`)
	require.NoError(t, err)
	studio, err = sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	_, err = studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('unrelated','sqlite',?,'owner','active','{}',1,?,?)`, sourceDSN, now, now)
	require.NoError(t, err)
	require.NoError(t, studio.Close())
	second, count := start()
	require.Zero(t, count.Load(), "restart and an unrelated connector must hydrate both readers without database schema discovery")
	address, _ := second.Addresses()
	response, err := http.Get("http://" + address + "/records")
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), "fresh", "business rows must execute against current DB data")
	closeService(second)
	studio, err = sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	_, err = studio.Exec(`UPDATE component_versions SET generated_dql=generated_dql || char(10) || '-- changed source' WHERE report_id='other' AND version_no=1`)
	require.NoError(t, err)
	require.NoError(t, studio.Close())
	third, count := start()
	require.EqualValues(t, 1, count.Load(), "changing one reader must compile only that reader")
	closeService(third)
	studio, err = sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	malformed := []byte(`{"version":`)
	malformedSHA := sha256.Sum256(malformed)
	_, err = studio.Exec(`UPDATE component_resource_files SET content=?,content_size=?,content_sha256=? WHERE namespace='__datly_compiled' AND report_id='records'`, malformed, len(malformed), hex.EncodeToString(malformedSHA[:]))
	require.NoError(t, err)
	require.NoError(t, studio.Close())
	fourth, count := start()
	require.EqualValues(t, 1, count.Load(), "malformed but checksum-valid artifact must be recompiled")
	closeService(fourth)
	fifth, count := start()
	require.Zero(t, count.Load(), "repaired artifact must hit on the next restart")
	before := fifth.manager.Revision()
	studio, err = sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	_, err = studio.Exec(`UPDATE component_versions SET generated_dql='not a valid DQL component' WHERE report_id='records' AND version_no=1`)
	require.NoError(t, err)
	require.Error(t, fifth.Reload(ctx, 0), "failed candidate compilation must not replace active readers")
	require.Equal(t, before, fifth.manager.Revision())
	address, _ = fifth.Addresses()
	response, err = http.Get("http://" + address + "/records")
	require.NoError(t, err)
	body, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), "fresh")
	closeService(fifth)
	defer studio.Close()
	var cached int
	require.NoError(t, studio.QueryRow(`SELECT COUNT(*) FROM component_resource_files WHERE namespace='__datly_compiled'`).Scan(&cached))
	require.GreaterOrEqual(t, cached, 3)
}

func TestPublishingOneAdditionalReaderKeepsExistingCompiledArtifacts(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runtime\n\ngo 1.25.0\n"), 0600))
	sourceDSN := "file:" + filepath.Join(root, "source.db")
	source, err := sql.Open("sqlite", sourceDSN)
	require.NoError(t, err)
	defer source.Close()
	_, err = source.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT); INSERT INTO records VALUES(1,'fresh')`)
	require.NoError(t, err)
	studioDSN := "file:" + filepath.Join(root, "studio.db")
	studio, err := sql.Open("sqlite", studioDSN)
	require.NoError(t, err)
	require.NoError(t, schema.ApplySQLite(ctx, studio, "studio"))
	now := time.Now().UTC()
	_, err = studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active','{}',1,?,?)`, sourceDSN, now, now)
	require.NoError(t, err)
	addReader := func(id, name string) {
		t.Helper()
		_, insertErr := studio.Exec(`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,'owner','active','source',?,?,1,?,?)`, id, id, id, "example.com/runtime/"+id, id, now, now)
		require.NoError(t, insertErr)
		dql := fmt.Sprintf(`#package('example.com/runtime/%s')
#setting($_ = $connector('source'))
#setting($_ = $route('/%s','GET'))
#define($_ = $Records<[]*%s>(output/view))
SELECT records.*, type(records,'%s') FROM (SELECT id,name FROM records) records`, id, id, name, name)
		_, insertErr = studio.Exec(`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,1,'published','dql',?,?,'{}','studio.v1','hash','{}','valid','v1','studio.v1',1,'owner',?)`, id, dql, dql, now)
		require.NoError(t, insertErr)
		_, insertErr = studio.Exec(`INSERT INTO component_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES(?,1,1,1,'active','fixture','hash','owner',?,?)`, id, now, now)
		require.NoError(t, insertErr)
	}
	addReader("first", "FirstRecord")
	addReader("second", "SecondRecord")
	_, err = studio.Exec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'fixture','active',2,'{}','owner',?,?)`, now, now)
	require.NoError(t, err)
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: studioDSN}, Admin: Admin{Token: "test-token"}, RootDir: root}
	activate := func() int32 {
		t.Helper()
		service, newErr := New(ctx, config)
		require.NoError(t, newErr)
		var live atomic.Int32
		service.compileRuntimeContract = func(ctx context.Context, source *transcribe.Source) (*transcribe.RuntimeContract, error) {
			live.Add(1)
			return transcribe.NewCompiler().RuntimeContracts(ctx, root, source)
		}
		require.NoError(t, service.Reload(ctx, 0))
		require.NoError(t, service.Close(ctx))
		return live.Load()
	}
	require.EqualValues(t, 2, activate())
	addReader("third", "ThirdRecord")
	_, err = studio.Exec(`UPDATE runtime_generations SET report_count=3 WHERE generation_no=1`)
	require.NoError(t, err)
	require.EqualValues(t, 1, activate(), "adding a reader must not rediscover the existing two")
	var stored int
	require.NoError(t, studio.QueryRow(`SELECT COUNT(*) FROM component_resource_files WHERE namespace='__datly_compiled'`).Scan(&stored))
	require.Equal(t, 3, stored)
	require.NoError(t, studio.Close())
}
