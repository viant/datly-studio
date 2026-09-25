package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/reports/store_capabilities"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type capabilityReader struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func (t *Transport) readReportCapability(ctx context.Context, reportID, subject string) (*stored.ReportCapability, error) {
	reader, err := t.reportCapabilityReader()
	if err != nil {
		return nil, err
	}
	input := &stored.Input{ReportId: reportID, Subject: subject,
		Has: &stored.InputHas{ReportId: true, Subject: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("report capability reader returned %T", value)
	}
	if len(output.Capabilities) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Capabilities) != 1 || output.Capabilities[0] == nil || output.Capabilities[0].ReportId != reportID {
		return nil, fmt.Errorf("report capability reader returned an ambiguous or mismatched report")
	}
	return output.Capabilities[0], nil
}

func (t *Transport) reportCapabilityReader() (*capabilityReader, error) {
	t.capabilityReaderMu.Lock()
	defer t.capabilityReaderMu.Unlock()
	if t.capabilityReader != nil {
		return t.capabilityReader, nil
	}
	resources := resource.New()
	if err := resources.Register(stored.CapabilityDatlyResourceNamespace, stored.CapabilityDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.CapabilityComponent{}), "store_capabilities",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	t.capabilityReader = &capabilityReader{runtime: runtime, target: target}
	return t.capabilityReader, nil
}
