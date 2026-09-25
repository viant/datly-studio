package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readResourceFileByID(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, resourceID string) (*stored.SnapshotFile, error) {
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo, ResourceId: resourceID,
		Has: &stored.InputHas{ReportId: true, VersionNo: true, ResourceId: true}}
	rows, err := t.readResourceFiles(ctx, tx, input)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(rows) != 1 || rows[0].ResourceId != resourceID {
		return nil, fmt.Errorf("resource file lookup returned ambiguous or mismatched rows")
	}
	return rows[0], nil
}

func (t *Transport) readResourceFilesByPath(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, namespace, resourcePath string) ([]*stored.SnapshotFile, error) {
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo, Namespace: namespace, ResourcePath: resourcePath,
		Has: &stored.InputHas{ReportId: true, VersionNo: true, Namespace: true, ResourcePath: true}}
	rows, err := t.readResourceFiles(ctx, tx, input)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Namespace != namespace || row.ResourcePath != resourcePath {
			return nil, fmt.Errorf("resource file path lookup returned a mismatched row")
		}
	}
	return rows, nil
}

func (t *Transport) readResourceFiles(ctx context.Context, tx *sql.Tx, input *stored.Input) ([]*stored.SnapshotFile, error) {
	resources := resource.New()
	if err := resources.Register(stored.FileDatlyResourceNamespace, stored.FileDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.FileComponent{}), "store_snapshot",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("resource file lookup returned %T", value)
	}
	for _, row := range output.Files {
		if row == nil || row.ReportId != input.ReportId || row.VersionNo != input.VersionNo {
			return nil, fmt.Errorf("resource file lookup returned a mismatched row")
		}
	}
	return output.Files, nil
}
