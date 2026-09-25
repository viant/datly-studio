package resources

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func ReadFolderByID(ctx context.Context, db *sql.DB, tx *sql.Tx, reportID string, versionNo int, folderID string) (*stored.SnapshotFolder, error) {
	resources := resource.New()
	if err := resources.Register(stored.FolderDatlyResourceNamespace, stored.FolderDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db, Tx: tx}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.FolderComponent{}), "store_snapshot",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo, FolderId: folderID,
		Has: &stored.InputHas{ReportId: true, VersionNo: true, FolderId: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("resource folder lookup returned %T", value)
	}
	if len(output.Folders) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Folders) != 1 || output.Folders[0] == nil || output.Folders[0].ReportId != reportID ||
		output.Folders[0].VersionNo != versionNo || output.Folders[0].FolderId != folderID {
		return nil, fmt.Errorf("resource folder lookup returned ambiguous or mismatched rows")
	}
	return output.Folders[0], nil
}
