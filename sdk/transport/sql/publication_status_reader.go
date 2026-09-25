package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_publications/store_status"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readPublicationRow(ctx context.Context, tx *sql.Tx, reportID string) (*stored.StoredPublication, error) {
	resources := resource.New()
	if err := resources.Register(stored.PublicationDatlyResourceNamespace, stored.PublicationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.PublicationComponent{}), "store_status",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{ReportId: reportID, Has: &stored.InputHas{ReportId: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("publication status reader returned %T", value)
	}
	if len(output.Publications) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Publications) != 1 || output.Publications[0] == nil || output.Publications[0].ReportId != reportID {
		return nil, fmt.Errorf("publication status reader returned an ambiguous or mismatched row")
	}
	return output.Publications[0], nil
}

func (t *Transport) readPublicationStatus(ctx context.Context, reportID string) (*sdk.Publication, error) {
	row, err := t.readPublicationRow(ctx, nil, reportID)
	if err != nil {
		return nil, err
	}
	result := &sdk.Publication{ReportID: row.ReportId, ActiveVersionNo: row.ActiveVersionNo,
		DesiredVersionNo: row.DesiredVersionNo, DesiredGeneration: row.DesiredGeneration,
		ActiveGeneration: row.ActiveGeneration, Status: row.PublicationStatus,
		SpecHash: row.SpecHash, PublishedAt: row.PublishedAt}
	if row.RuntimeRevision != nil {
		result.RuntimeRevision = *row.RuntimeRevision
	}
	return result, nil
}
