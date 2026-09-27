package resources

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/internal/resourcesnapshot"
	"github.com/viant/datly-studio/sdk"
	files "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	folders "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	skills "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type resourceSnapshotReader struct {
	runtime                *druntime.Runtime
	files, folders, skills dexec.ComponentTarget
}

// The runtime is scoped to one snapshot and shut down by its caller.
func newResourceSnapshotReader(db *sql.DB) (*resourceSnapshotReader, error) {
	resources := resource.New()
	for _, item := range []struct {
		namespace string
		data      fs.FS
	}{
		{files.FileDatlyResourceNamespace, files.FileDatlyResources},
		{folders.FolderDatlyResourceNamespace, folders.FolderDatlyResources},
		{skills.SkillDatlyResourceNamespace, skills.SkillDatlyResources},
	} {
		if err := resources.Register(item.namespace, item.data); err != nil {
			return nil, err
		}
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	var registrations []*registry.RegisteredComponent
	var targets [3]dexec.ComponentTarget
	for i, item := range []struct{ component, input, output reflect.Type }{
		{reflect.TypeOf(files.FileComponent{}), reflect.TypeOf(files.Input{}), reflect.TypeOf(files.Output{})},
		{reflect.TypeOf(folders.FolderComponent{}), reflect.TypeOf(folders.Input{}), reflect.TypeOf(folders.Output{})},
		{reflect.TypeOf(skills.SkillComponent{}), reflect.TypeOf(skills.Input{}), reflect.TypeOf(skills.Output{})},
	} {
		registration, target, err := readercomponent.Compile(item.component, "store_snapshot", item.input, item.output, resources, connector)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, registration)
		targets[i] = target
	}
	runtime, err := druntime.NewRuntime(registrations, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &resourceSnapshotReader{runtime: runtime, files: targets[0], folders: targets[1], skills: targets[2]}, nil
}

func ReadSnapshot(ctx context.Context, db *sql.DB, reportID string, versionNo int, result *sdk.ResourceSnapshot) (err error) {
	reader, err := newResourceSnapshotReader(db)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.runtime.Shutdown(context.Background())) }()
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.files, Input: &files.Input{
		ReportId: reportID, VersionNo: versionNo, Has: &files.InputHas{ReportId: true, VersionNo: true},
	}})
	if err != nil {
		return err
	}
	fileOutput, ok := value.(*files.Output)
	if !ok {
		return fmt.Errorf("resource file reader returned %T", value)
	}
	result.Files, err = resourcesnapshot.Files(reportID, versionNo, fileOutput.Files)
	if err != nil {
		return err
	}
	value, err = reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.folders, Input: &folders.Input{
		ReportId: reportID, VersionNo: versionNo, Has: &folders.InputHas{ReportId: true, VersionNo: true},
	}})
	if err != nil {
		return err
	}
	folderOutput, ok := value.(*folders.Output)
	if !ok {
		return fmt.Errorf("resource folder reader returned %T", value)
	}
	result.Folders, err = resourcesnapshot.Folders(reportID, versionNo, folderOutput.Folders)
	if err != nil {
		return err
	}
	value, err = reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.skills, Input: &skills.Input{
		ReportId: reportID, VersionNo: versionNo, Has: &skills.InputHas{ReportId: true, VersionNo: true},
	}})
	if err != nil {
		return err
	}
	skillOutput, ok := value.(*skills.Output)
	if !ok {
		return fmt.Errorf("skill root reader returned %T", value)
	}
	result.Skills, err = resourcesnapshot.Skills(reportID, versionNo, skillOutput.Skills)
	if err != nil {
		return err
	}
	return nil
}
