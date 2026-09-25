package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readSkillRootByID(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, skillID string) (*stored.SnapshotSkill, error) {
	resources := resource.New()
	if err := resources.Register(stored.SkillDatlyResourceNamespace, stored.SkillDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.SkillComponent{}), "store_snapshot",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo, SkillId: skillID,
		Has: &stored.InputHas{ReportId: true, VersionNo: true, SkillId: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("skill root lookup returned %T", value)
	}
	if len(output.Skills) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Skills) != 1 || output.Skills[0] == nil || output.Skills[0].ReportId != reportID ||
		output.Skills[0].VersionNo != versionNo || output.Skills[0].SkillId != skillID {
		return nil, fmt.Errorf("skill root lookup returned ambiguous or mismatched rows")
	}
	return output.Skills[0], nil
}
