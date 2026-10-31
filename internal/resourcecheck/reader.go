// Package resourcecheck applies namespace limits to native, bound reader queries.
package resourcecheck

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/viant/datly/observability"
	"reflect"

	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/sqlx"
	xhandler "github.com/viant/xdatly/handler"
	api "google.golang.org/api/bigquery/v2"
)

// DryRunner is an optional driver capability, discovered without a local module overlay.
type DryRunner interface {
	DryRun(context.Context, string, []driver.NamedValue) (*api.Job, error)
}

// CheckShape fails closed where the native dry-run API cannot cover all queries.
func CheckShape(component *spec.Component) error {
	if component == nil || component.RootView == nil {
		return errors.New("resource checks require a reader")
	}
	root := component.RootView
	if len(root.Relations) > 0 || root.Partitioning != nil || root.SelfReference != nil || len(component.Views) > 0 {
		return errors.New("resource checks require whole-query planning; this Datly version exposes only the root query")
	}
	if component.Settings != nil && component.Settings.Report != nil && component.Settings.Report.Enabled {
		return errors.New("resource checks for cube and composition require whole-query planning")
	}
	return nil
}

// Wrap preserves Datly's input binding and query planning; it never rebuilds SQL.
func Wrap(reader exec.Reader, db *sql.DB, component *spec.Component, limits *settings.ResourceLimits) (exec.Reader, error) {
	if limits == nil {
		return nil, errors.New("resource checks are enabled but namespace resource limits are not configured")
	}
	var planner exec.ReadPlanner
	native, hasGuard := reader.(interface {
		WithQueryGuard(func(context.Context, *sql.DB, string, []any) error) exec.Reader
	})
	if hasGuard {
		reader = native.WithQueryGuard(func(ctx context.Context, source *sql.DB, query string, args []any) error {
			return Check(ctx, source, &exec.ReadPlan{SQL: query, Args: args}, limits)
		})
	} else {
		if err := CheckShape(component); err != nil {
			return nil, err
		}
		var ok bool
		planner, ok = reader.(exec.ReadPlanner)
		if !ok {
			return nil, errors.New("native reader query admission is unavailable")
		}
	}
	field := ""
	for _, p := range spec.EffectiveParameters(component.Parameters) {
		if p != nil && p.Source.Kind == "output" && p.Source.Name == "view" {
			field = p.Name
			break
		}
	}
	return &checkedReader{reader: reader, planner: planner, db: db, limits: *limits, field: field}, nil
}

type checkedReader struct {
	reader  exec.Reader
	planner exec.ReadPlanner
	db      *sql.DB
	limits  settings.ResourceLimits
	field   string
}

func (r *checkedReader) Read(ctx context.Context, input any, binder xhandler.Binder, resolver sqlx.ParameterResolver) (any, error) {
	if r.planner != nil {
		plan, err := r.planner.PlanRead(ctx, input, binder, resolver)
		if err != nil {
			return nil, err
		}
		if err = Check(ctx, r.db, plan, &r.limits); err != nil {
			return nil, err
		}
	}
	value, err := r.reader.Read(ctx, input, binder, resolver)
	if err != nil {
		return nil, err
	}
	if r.limits.MaxReturnedRows != nil {
		if err = checkRows(value, r.field, *r.limits.MaxReturnedRows); err != nil {
			return nil, err
		}
	}

	return value, nil
}

// Check exposes no SQL, argument or credential details in its returned errors.
func Check(ctx context.Context, db *sql.DB, plan *exec.ReadPlan, limits *settings.ResourceLimits) error {
	if db == nil || plan == nil || limits == nil {
		return errors.New("query resource check configuration is unavailable")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return errors.New("query resource check connector is unavailable")
	}
	defer conn.Close()
	var job *api.Job
	err = conn.Raw(func(raw any) error {
		capability, ok := raw.(DryRunner)
		if !ok {
			return errors.New("the configured driver does not expose BigQuery dry-run statistics")
		}
		args := make([]driver.NamedValue, len(plan.Args))
		for i, value := range plan.Args {
			args[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
		}
		var dryErr error
		job, dryErr = capability.DryRun(ctx, plan.SQL, args)
		return dryErr
	})
	if err != nil {
		return errors.New("BigQuery resource check failed or dry-run capability is unavailable")
	}
	return CheckStatistics(job, limits)
}

func CheckStatistics(job *api.Job, limits *settings.ResourceLimits) error {
	if job == nil || job.Statistics == nil || job.Statistics.Query == nil {
		return errors.New("BigQuery dry run did not provide query statistics")
	}
	stats := job.Statistics.Query
	if limits.MaxBytesProcessed != nil {
		switch stats.TotalBytesProcessedAccuracy {
		case "PRECISE", "UPPER_BOUND":
		default:
			return errors.New("BigQuery did not provide a safe upper estimate for bytes processed")
		}
		if stats.TotalBytesProcessed > *limits.MaxBytesProcessed {
			return fmt.Errorf("query bytes processed exceed namespace threshold (%d > %d)", stats.TotalBytesProcessed, *limits.MaxBytesProcessed)
		}
	}
	if limits.MaxReferencedTables != nil {
		// Missing lists are unknown, including an empty result for complex/external SQL.
		if len(stats.ReferencedTables) == 0 && (stats.StatementType != "SELECT" || stats.Schema == nil) {
			return errors.New("BigQuery did not provide referenced table statistics")
		}

		distinct := map[string]bool{}
		for _, table := range stats.ReferencedTables {
			if table != nil {
				distinct[table.ProjectId+"."+table.DatasetId+"."+table.TableId] = true
			}
		}
		if len(distinct) > *limits.MaxReferencedTables {
			return fmt.Errorf("query referenced tables exceed namespace threshold (%d > %d)", len(distinct), *limits.MaxReferencedTables)
		}
	}
	return nil
}

// Warmup stays in Datly; each source read is admitted by the native guard.
func (r *checkedReader) Warmup(ctx context.Context, in exec.ReaderWarmupInvocation) (int, error) {
	if r.planner != nil {
		return 0, errors.New("cache warmup resource checks require native per-query admission")
	}
	warmer, ok := r.reader.(exec.ReaderWarmer)
	if !ok {
		return 0, errors.New("native cache warmup is unavailable")
	}
	return warmer.Warmup(ctx, in)
}
func (r *checkedReader) WarmupTargets() []exec.ReaderWarmupTarget {
	if target, ok := r.reader.(exec.ReaderWarmupTargeter); ok {
		return target.WarmupTargets()
	}
	return nil
}
func (r *checkedReader) WithRecorder(recorder *observability.Recorder) exec.Reader {
	copy := *r
	if native, ok := r.reader.(interface {
		WithRecorder(*observability.Recorder) exec.Reader
	}); ok {
		copy.reader = native.WithRecorder(recorder)
		if r.planner != nil {
			copy.planner, _ = copy.reader.(exec.ReadPlanner)
		}
	}
	return &copy
}
func (r *checkedReader) PrepareQuery(ctx context.Context, input any, binder xhandler.Binder, resolver sqlx.ParameterResolver) (*exec.PreparedQuery, error) {
	if r.planner != nil {
		return nil, errors.New("projection resource checks require native per-query admission")
	}
	preparer, ok := r.reader.(exec.QueryPreparer)
	if !ok {
		return nil, errors.New("native query preparation is unavailable")
	}
	prepared, err := preparer.PrepareQuery(ctx, input, binder, resolver)
	if err != nil {
		return nil, err
	}
	if prepared != nil && prepared.Projection != nil && r.limits.MaxReturnedRows != nil {
		copy := *prepared
		copy.Projection = limitedProjection{reader: prepared.Projection, maximum: *r.limits.MaxReturnedRows}
		prepared = &copy
	}
	return prepared, nil
}

func checkRows(value any, field string, maximum int64) error {
	rows := reflect.ValueOf(value)
	for rows.IsValid() && (rows.Kind() == reflect.Pointer || rows.Kind() == reflect.Interface) {
		if rows.IsNil() {
			return nil
		}
		rows = rows.Elem()
	}
	if rows.IsValid() && rows.Kind() == reflect.Struct && field != "" {
		rows = rows.FieldByName(field)
		for rows.IsValid() && rows.Kind() == reflect.Pointer {
			if rows.IsNil() {
				return nil
			}
			rows = rows.Elem()
		}
	}
	if !rows.IsValid() || rows.Kind() != reflect.Slice && rows.Kind() != reflect.Array {
		return errors.New("native result row count is unavailable")
	}
	if int64(rows.Len()) > maximum {
		return errors.New("query returned more rows than the namespace threshold")
	}
	return nil
}

type limitedProjection struct {
	reader  exec.ProjectionReader
	maximum int64
}

func (p limitedProjection) ReadProjection(ctx context.Context, request exec.ProjectionRequest) (any, error) {
	value, err := p.reader.ReadProjection(ctx, request)
	if err != nil {
		return nil, err
	}
	if err = checkRows(value, "", p.maximum); err != nil {
		return nil, err
	}
	return value, nil
}
