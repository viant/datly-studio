package sqltransport

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/componentarchive"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_resource_files/store_download"
	budget "github.com/viant/datly-studio/studio/report_resource_files/store_download_budget"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) downloadComponent(ctx context.Context, input, output any) error {
	var identity versionIdentityRequest
	if err := decode(input, &identity); err != nil {
		return invalid(err)
	}
	version, err := t.getVersionValue(ctx, identity.ReportID, identity.VersionNo)
	if err != nil {
		return err
	}
	source := version.GeneratedDQL
	if source == "" {
		source = version.AuthoredDQL
	}
	if err := componentarchive.CheckSource(source); err != nil {
		return invalid(err)
	}
	rows, err := t.downloadResourceFiles(ctx, identity.ReportID, identity.VersionNo)
	if err != nil {
		if errors.Is(err, componentarchive.ErrInvalid) {
			return invalid(err)
		}
		return internal(err)
	}
	files := make([]componentarchive.File, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			return internal(fmt.Errorf("download resource reader returned a mismatched row"))
		}
		files = append(files, componentarchive.File{ReportID: row.ReportId, VersionNo: row.VersionNo, ResourcePath: row.ResourcePath, Content: row.Content})
	}
	download, err := componentarchive.Build(ctx, identity.ReportID, identity.VersionNo, source, files)
	if err != nil {
		if errors.Is(err, componentarchive.ErrInvalid) {
			return invalid(err)
		}
		return internal(err)
	}
	return assign(output, download)
}

func (t *Transport) downloadResourceFiles(ctx context.Context, reportID string, versionNo int) ([]*stored.DownloadResourceFile, error) {
	resources := resource.New()
	if err := resources.Register(stored.FileDatlyResourceNamespace, stored.FileDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(budget.BudgetDatlyResourceNamespace, budget.BudgetDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.FileComponent{}), "store_download",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	budgetRegistration, budgetTarget, err := readercomponent.Compile(reflect.TypeOf(budget.BudgetComponent{}), "store_download_budget",
		reflect.TypeOf(budget.Input{}), reflect.TypeOf(budget.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{budgetRegistration, registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	budgetInput := &budget.Input{}
	budgetInput.SetReportId(reportID)
	budgetInput.SetVersionNo(versionNo)
	budgetValue, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: budgetTarget, Input: budgetInput})
	if err != nil {
		return nil, err
	}
	budgetOutput, ok := budgetValue.(*budget.Output)
	if !ok || budgetOutput == nil || budgetOutput.Summary == nil {
		return nil, fmt.Errorf("download resource budget reader returned %T without totals", budgetValue)
	}
	if err := componentarchive.CheckBudget(budgetOutput.Summary.FileCount, budgetOutput.Summary.ContentBytes); err != nil {
		return nil, err
	}
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo,
		Has: &stored.InputHas{ReportId: true, VersionNo: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("download resource reader returned %T", value)
	}
	return output.Files, nil
}
