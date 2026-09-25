package host

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/reports/store_run_access"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// runAccessStore preserves legacy owner/report_acl checks when generic
// resource policies are not configured. The reader is invoked in-process only.
type runAccessStore struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func newRunAccessStore(db *sql.DB) (*runAccessStore, error) {
	resources := resource.New()
	if err := resources.Register(stored.AccessDatlyResourceNamespace, stored.AccessDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.AccessComponent{}), "store_run_access",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &runAccessStore{runtime: runtime, target: target}, nil
}

func (s *runAccessStore) Allowed(ctx context.Context, reportID, subject string) (bool, error) {
	if s == nil || s.runtime == nil || reportID == "" || subject == "" {
		return false, fmt.Errorf("run access reader requires report ID and verified subject")
	}
	input := &stored.Input{ReportId: reportID, Subject: subject, Has: &stored.InputHas{ReportId: true, Subject: true}}
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.target, Input: input})
	if err != nil {
		return false, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return false, fmt.Errorf("run access reader returned %T", value)
	}
	if len(output.Allowed) == 0 {
		return false, nil
	}
	if len(output.Allowed) != 1 || output.Allowed[0] == nil || output.Allowed[0].ReportId != reportID {
		return false, fmt.Errorf("run access reader returned an ambiguous or mismatched report")
	}
	return true, nil
}

func (s *runAccessStore) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Shutdown(ctx)
}
