package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	list "github.com/viant/datly-studio/studio/report_acl/store_list"
	one "github.com/viant/datly-studio/studio/report_acl/store_one"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type aclStoreReader struct {
	runtime *druntime.Runtime
	list    dexec.ComponentTarget
	one     dexec.ComponentTarget
}

func (t *Transport) aclStoreReader() (*aclStoreReader, error) {
	t.aclReaderMu.Lock()
	defer t.aclReaderMu.Unlock()
	if t.aclReader != nil {
		return t.aclReader, nil
	}
	resources := resource.New()
	if err := resources.Register(list.AclDatlyResourceNamespace, list.AclDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(one.AclDatlyResourceNamespace, one.AclDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	listRegistration, listTarget, err := readercomponent.Compile(reflect.TypeOf(list.AclComponent{}), "store_list",
		reflect.TypeOf(list.Input{}), reflect.TypeOf(list.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	oneRegistration, oneTarget, err := readercomponent.Compile(reflect.TypeOf(one.AclComponent{}), "store_one",
		reflect.TypeOf(one.Input{}), reflect.TypeOf(one.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{listRegistration, oneRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	t.aclReader = &aclStoreReader{runtime: runtime, list: listTarget, one: oneTarget}
	return t.aclReader, nil
}

func (t *Transport) readACLList(ctx context.Context, reportID string) ([]*sdk.ReportACL, error) {
	reader, err := t.aclStoreReader()
	if err != nil {
		return nil, err
	}
	input := &list.Input{ReportId: reportID, Has: &list.InputHas{ReportId: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.list, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*list.Output)
	if !ok {
		return nil, fmt.Errorf("ACL list reader returned %T", value)
	}
	result := make([]*sdk.ReportACL, 0, len(output.Access))
	for _, row := range output.Access {
		if row == nil || row.ReportId != reportID {
			return nil, fmt.Errorf("ACL list reader returned a mismatched report")
		}
		result = append(result, aclFromValues(row.ReportId, row.SubjectType, row.SubjectId, row.CanView, row.CanRun,
			row.CanEdit, row.CanPublish, row.CanUseDql, row.Etag))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SubjectType != result[j].SubjectType {
			return result[i].SubjectType < result[j].SubjectType
		}
		return result[i].SubjectID < result[j].SubjectID
	})
	return result, nil
}

func (t *Transport) readACLOne(ctx context.Context, reportID, subjectType, subjectID string) (*sdk.ReportACL, error) {
	reader, err := t.aclStoreReader()
	if err != nil {
		return nil, err
	}
	input := &one.Input{ReportId: reportID, SubjectType: subjectType, SubjectId: subjectID,
		Has: &one.InputHas{ReportId: true, SubjectType: true, SubjectId: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.one, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*one.Output)
	if !ok {
		return nil, fmt.Errorf("ACL identity reader returned %T", value)
	}
	if len(output.Access) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Access) != 1 || output.Access[0] == nil || output.Access[0].ReportId != reportID ||
		output.Access[0].SubjectType != subjectType || output.Access[0].SubjectId != subjectID {
		return nil, fmt.Errorf("ACL identity reader returned an ambiguous or mismatched row")
	}
	row := output.Access[0]
	return aclFromValues(row.ReportId, row.SubjectType, row.SubjectId, row.CanView, row.CanRun,
		row.CanEdit, row.CanPublish, row.CanUseDql, row.Etag), nil
}

func aclFromValues(reportID, subjectType, subjectID string, canView, canRun, canEdit, canPublish, canUseDQL bool, etag int64) *sdk.ReportACL {
	return &sdk.ReportACL{ReportID: reportID, SubjectType: subjectType, SubjectID: subjectID,
		CanView: canView, CanRun: canRun, CanEdit: canEdit, CanPublish: canPublish, CanUseDQL: canUseDQL, ETag: etag}
}
