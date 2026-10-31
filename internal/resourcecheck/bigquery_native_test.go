package resourcecheck_test

import (
	"context"
	"database/sql"
	"encoding/json"
	xreader "github.com/viant/xdatly/reader"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	bq "github.com/viant/bigquery"
	"github.com/viant/datly-studio/internal/resourcecheck"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/exec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	api "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

// Run with the three local modules to qualify their new capabilities together.
// Published dependencies without native admission retain the legacy tests above.
func TestNativeBigQueryResourceAdmission(t *testing.T) {
	ctx := context.Background()
	schemaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer schemaDB.Close()
	contract, err := transcribe.NewCompiler().RuntimeContractsInModule(ctx, "example.com/test", &transcribe.Source{Scope: "example.com/test/reader", Name: "reader", Connector: "consumer", Text: `#package('example.com/test/reader')
#setting($_ = $connector('consumer'))
#setting($_ = $route('/records','GET'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT r.*, type(r,'Record') FROM (SELECT 'one' AS id) r`, ColumnRefiner: column.New(column.Connections{"consumer": schemaDB})})
	require.NoError(t, err)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources})
	require.NoError(t, err)
	probe, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: schemaDB}})
	require.NoError(t, err)
	if _, ok := any(probe).(interface {
		WithQueryGuard(xreader.QueryGuard) exec.Reader
	}); !ok {
		t.Skip("published Datly pin predates native per-query admission")
	}
	var dry, executed atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			var job api.Job
			require.NoError(t, json.NewDecoder(r.Body).Decode(&job))
			if job.Configuration.DryRun {
				dry.Add(1)
				_, _ = w.Write([]byte(`{"status":{"state":"DONE"},"statistics":{"query":{"totalBytesProcessed":"100","totalBytesProcessedAccuracy":"PRECISE","referencedTables":[{"projectId":"p","datasetId":"d","tableId":"t"}]}}}`))
				return
			}
			executed.Add(1)
			_, _ = w.Write([]byte(`{"jobReference":{"projectId":"p","jobId":"j"},"status":{"state":"DONE"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jobComplete":true,"totalRows":"1","schema":{"fields":[{"name":"id","type":"STRING","mode":"NULLABLE"}]},"rows":[{"f":[{"v":"one"}]}]}`))
	}))
	defer server.Close()
	bq.SetOptions(option.WithHTTPClient(server.Client()))
	defer bq.SetOptions()
	cfg, err := bq.ParseDSN("bigquery://p/d?endpoint=" + url.QueryEscape(server.URL+"/"))
	require.NoError(t, err)
	driverConnector, err := bq.NewConnector(cfg)
	require.NoError(t, err)
	db := sql.OpenDB(driverConnector)
	defer db.Close()
	connector := &dsql.SQLComponent{DB: db}
	require.NoError(t, connector.RegisterConnector("consumer", db))
	reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	require.NoError(t, err)
	// No component opt-in means the native reader executes directly, with no
	// extra metadata job even though the driver supports dry-run statistics.
	_, err = reader.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, dry.Load())
	require.EqualValues(t, 1, executed.Load())
	executed.Store(0)
	bytes := int64(99)
	guarded, err := resourcecheck.Wrap(reader, db, contract.Component, &settings.ResourceLimits{MaxBytesProcessed: &bytes})
	require.NoError(t, err)
	_, err = guarded.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.ErrorContains(t, err, "bytes processed exceed")
	require.EqualValues(t, 0, executed.Load())
	require.EqualValues(t, 1, dry.Load())
	bytes = 101
	guarded, err = resourcecheck.Wrap(reader, db, contract.Component, &settings.ResourceLimits{MaxBytesProcessed: &bytes})
	require.NoError(t, err)
	_, err = guarded.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, executed.Load())
	require.EqualValues(t, 2, dry.Load())
}
