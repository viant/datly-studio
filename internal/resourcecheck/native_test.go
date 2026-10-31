package resourcecheck_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/resourcecheck"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/exec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	api "google.golang.org/api/bigquery/v2"
)

type dryDriver struct{}
type dryConnection struct{}

func (dryDriver) Open(string) (driver.Conn, error) { return dryConnection{}, nil }
func (dryConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("dry-run connection must not execute a query")
}
func (dryConnection) Close() error              { return nil }
func (dryConnection) Begin() (driver.Tx, error) { return nil, errors.New("no transactions") }
func (dryConnection) DryRun(context.Context, string, []driver.NamedValue) (*api.Job, error) {
	return &api.Job{Statistics: &api.JobStatistics{Query: &api.JobStatistics2{TotalBytesProcessed: 100, TotalBytesProcessedAccuracy: "PRECISE"}}}, nil
}

// Keep legacy compatibility coverage even when this test uses a newer Datly checkout.
type legacyPlanner struct {
	exec.Reader
	exec.ReadPlanner
}

var registerDry sync.Once

func TestNativeReaderUsesBoundDryRunAndCountsReturnedRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	contract, err := transcribe.NewCompiler().RuntimeContractsInModule(ctx, "example.com/test", &transcribe.Source{Scope: "example.com/test/reader", Name: "reader", Connector: "consumer", Text: `#package('example.com/test/reader')
#setting($_ = $connector('consumer'))
#setting($_ = $route('/records','GET'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT r.*, type(r,'Record') FROM (SELECT 1 AS id) r`, ColumnRefiner: column.New(column.Connections{"consumer": db})})
	require.NoError(t, err)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources})
	require.NoError(t, err)
	connector := &dsql.SQLComponent{DB: db}
	require.NoError(t, connector.RegisterConnector("consumer", db))
	reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	require.NoError(t, err)
	registerDry.Do(func() { sql.Register("studio-resource-dry-test", dryDriver{}) })
	dryDB, err := sql.Open("studio-resource-dry-test", "")
	require.NoError(t, err)
	defer dryDB.Close()
	bytes := int64(99)
	limits := &settings.ResourceLimits{MaxBytesProcessed: &bytes}
	checked, err := resourcecheck.Wrap(legacyPlanner{reader, any(reader).(exec.ReadPlanner)}, dryDB, contract.Component, limits)
	require.NoError(t, err)
	_, err = checked.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.ErrorContains(t, err, "bytes processed exceed")
	bytes = 101
	checked, err = resourcecheck.Wrap(legacyPlanner{reader, any(reader).(exec.ReadPlanner)}, dryDB, contract.Component, limits)
	require.NoError(t, err)
	result, err := checked.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	maxRows := int64(0)
	limits.MaxReturnedRows = &maxRows
	checked, err = resourcecheck.Wrap(legacyPlanner{reader, any(reader).(exec.ReadPlanner)}, dryDB, contract.Component, limits)
	require.NoError(t, err)
	_, err = checked.Read(ctx, reflect.New(contract.InputType).Interface(), nil, nil)
	require.ErrorContains(t, err, "more rows")
}
