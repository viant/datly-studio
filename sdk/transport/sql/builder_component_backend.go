package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	versionedit "github.com/viant/datly-studio/studio/report_versions/store_edit"
	reportconfig "github.com/viant/datly-studio/studio/reports/store_config"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

func (t *Transport) builderWriteVersionEdit(ctx context.Context, tx *sql.Tx, row *versionedit.StoredVersion) error {
	if t.ComponentInvoker == nil {
		return t.writeVersionEdit(ctx, tx, row)
	}
	expected := *row.SourceRevision
	in := &versionedit.Input{}
	in.SetVersions([]*versionedit.StoredVersion{row})
	value, err := t.ComponentInvoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[versionedit.VersionComponent]().PkgPath(), Name: "version"},
		Route:     spec.RouteRef{Method: "PATCH", Path: "/_studio/report-version-store/edit"},
	}, Input: in})
	if err != nil {
		return err
	}
	result, ok := value.(*versionedit.Output)
	if !ok || result == nil || len(result.Data) != 1 || result.Data[0] == nil || result.Data[0].SourceRevision == nil || *result.Data[0].SourceRevision != expected+1 {
		return fmt.Errorf("native version edit returned %T without next source revision", value)
	}
	return nil
}

func (t *Transport) builderWriteReportConfig(ctx context.Context, tx *sql.Tx, row *reportconfig.StoredReport) error {
	if t.ComponentInvoker == nil {
		return t.writeReportConfig(ctx, tx, row)
	}
	expected := *row.Etag
	in := &reportconfig.Input{}
	in.SetReports([]*reportconfig.StoredReport{row})
	value, err := t.ComponentInvoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[reportconfig.ReportComponent]().PkgPath(), Name: "report"},
		Route:     spec.RouteRef{Method: "PATCH", Path: "/_studio/report-store/config"},
	}, Input: in})
	if err != nil {
		return err
	}
	result, ok := value.(*reportconfig.Output)
	if !ok || result == nil || len(result.Data) != 1 || result.Data[0] == nil || result.Data[0].Etag == nil || *result.Data[0].Etag != expected+1 {
		return fmt.Errorf("native report config returned %T without next etag", value)
	}
	return nil
}
