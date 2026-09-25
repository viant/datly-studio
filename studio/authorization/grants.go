package authorization

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/authorization_grants/store_read"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/xdatly/connector"
)

// readGrants resolves only the explicit verified-user grant selection. The
// generated Datly reader applies every selector as a typed SQL predicate.
func readGrants(ctx context.Context, provider connector.Provider, subject, reportID, source string, publishOnly bool) ([]*stored.Grant, error) {
	db, err := provider.Connector(ctx, "studio")
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, fmt.Errorf("authorization connector is unavailable")
	}
	resources := resource.New()
	if err := resources.Register(stored.GrantDatlyResourceNamespace, stored.GrantDatlyResources); err != nil {
		return nil, err
	}
	sql := &dsql.SQLComponent{DB: db}
	if err := sql.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.GrantComponent{}), "store_read",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, sql)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{SubjectId: subject, SubjectType: "user", IsLive: true,
		Has: &stored.InputHas{SubjectId: true, SubjectType: true, IsLive: true}}
	if reportID != "" {
		input.SetReportId(reportID)
	}
	if source != "" {
		input.SetGrantSource(source)
	}
	if publishOnly {
		input.SetCanPublish(true)
	}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("authorization grants returned %T", value)
	}
	if len(output.Grants) > 2 {
		return nil, fmt.Errorf("authorization grants exceeded requested limit")
	}
	for _, grant := range output.Grants {
		if grant == nil || grant.ReportId == "" || grant.SubjectId != subject || grant.SubjectType != "user" ||
			!grant.IsLive || reportID != "" && grant.ReportId != reportID ||
			source != "" && grant.GrantSource != source || publishOnly && !grant.CanPublish {
			return nil, fmt.Errorf("authorization grants returned a mismatched row")
		}
	}
	return output.Grants, nil
}
