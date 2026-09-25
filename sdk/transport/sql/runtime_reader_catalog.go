package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/runtime_generations/catalogpredicate"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_readers"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readRuntimeReaderCatalog(ctx context.Context, generation int64) ([]sdk.RuntimeReader, error) {
	resources := resource.New()
	if err := resources.Register(stored.ReaderDatlyResourceNamespace, stored.ReaderDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/runtime_generations/catalogpredicate",
		Types: []reflect.Type{reflect.TypeFor[catalogpredicate.ReaderScope]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.ReaderComponent{}), "store_readers",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	principal, scoped := sdk.PrincipalFromContext(ctx)
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Generation: generation, Subject: principal.Subject, Scoped: scoped,
			Has: &stored.InputHas{Generation: true, Subject: true, Scoped: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("runtime reader catalog returned %T", value)
	}
	result := make([]sdk.RuntimeReader, 0, len(output.Readers))
	for _, row := range output.Readers {
		if row == nil || row.ReportId == "" || row.VersionNo <= 0 {
			return nil, fmt.Errorf("runtime reader catalog returned an invalid root")
		}
		item := sdk.RuntimeReader{ReportID: row.ReportId, Title: row.Title, Namespace: row.Namespace,
			OwnerPackage: sdk.OwnerPackageSegment(row.OwnerId), ConnectorName: row.DefaultConnectorName,
			ComponentName: row.ComponentName, VersionNo: row.VersionNo, Status: row.PublicationStatus,
			RuntimeRevision: versionOptionalString(row.RuntimeRevision), ActivatedAt: row.ActivatedAt}
		for _, exposure := range row.Exposure {
			if exposure == nil || exposure.ReportId != row.ReportId || exposure.VersionNo != row.VersionNo {
				return nil, fmt.Errorf("runtime reader catalog returned a mismatched exposure")
			}
			item.MCPExposures = append(item.MCPExposures, sdk.RuntimeMCPExposure{Kind: exposure.Kind,
				Name: exposure.Name, Component: row.ComponentName, Method: exposure.RouteMethod,
				Path: exposure.RoutePath, Description: versionOptionalString(exposure.Description),
				MIMEType: versionOptionalString(exposure.MimeType), Enabled: exposure.Enabled})
		}
		for _, folder := range row.Folder {
			if folder == nil || folder.ReportId != row.ReportId || folder.VersionNo != row.VersionNo {
				return nil, fmt.Errorf("runtime reader catalog returned a mismatched folder")
			}
			item.MCPResources = append(item.MCPResources, sdk.RuntimeMCPResource{Namespace: folder.Namespace,
				RootPath: folder.RootPath, URIPrefix: folder.UriPrefix})
		}
		for _, skill := range row.Skill {
			if skill == nil || skill.ReportId != row.ReportId || skill.VersionNo != row.VersionNo {
				return nil, fmt.Errorf("runtime reader catalog returned a mismatched skill")
			}
			item.Skills = append(item.Skills, sdk.RuntimeSkill{SkillID: skill.SkillId,
				SkillRoot: skill.SkillRoot, URIPrefix: skill.UriPrefix})
		}
		result = append(result, item)
	}
	return result, nil
}
