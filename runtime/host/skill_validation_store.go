package host

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	storedcontent "github.com/viant/datly-studio/studio/report_resource_files/store_skill_content"
	storedroots "github.com/viant/datly-studio/studio/report_skill_roots/store_validation"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// skillValidationStore is an internal Datly runtime. Its routes are dispatch
// metadata and are never installed on the Studio HTTP or MCP gateways.
type skillValidationStore struct {
	runtime *druntime.Runtime
	roots   dexec.ComponentTarget
	content dexec.ComponentTarget
}

func newSkillValidationStore(db *sql.DB) (*skillValidationStore, error) {
	resources := resource.New()
	if err := resources.Register(storedroots.SkillDatlyResourceNamespace, storedroots.SkillDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(storedcontent.FileDatlyResourceNamespace, storedcontent.FileDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	rootRegistration, rootTarget, err := readercomponent.Compile(
		reflect.TypeOf(storedroots.SkillComponent{}), "store_validation",
		reflect.TypeOf(storedroots.Input{}), reflect.TypeOf(storedroots.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	contentRegistration, contentTarget, err := readercomponent.Compile(
		reflect.TypeOf(storedcontent.FileComponent{}), "store_skill_content",
		reflect.TypeOf(storedcontent.Input{}), reflect.TypeOf(storedcontent.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{rootRegistration, contentRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &skillValidationStore{runtime: runtime, roots: rootTarget, content: contentTarget}, nil
}

func (s *skillValidationStore) Close(ctx context.Context) error {
	return s.runtime.Shutdown(ctx)
}

func (s *skillValidationStore) Skills(ctx context.Context, reportID string, versionNo int) ([]*storedroots.ValidationSkill, error) {
	input := &storedroots.Input{ReportId: reportID, VersionNo: versionNo,
		Has: &storedroots.InputHas{ReportId: true, VersionNo: true}}
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.roots, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedroots.Output)
	if !ok {
		return nil, fmt.Errorf("skill validation roots returned %T", value)
	}
	for _, skill := range output.Skills {
		if skill == nil || skill.ReportId != reportID || skill.VersionNo != versionNo || skill.SkillId == "" || skill.Namespace == "" {
			return nil, fmt.Errorf("skill validation roots returned an incomplete or mismatched identity")
		}
	}
	return output.Skills, nil
}

func (s *skillValidationStore) Content(ctx context.Context, reportID string, versionNo int, namespace, resourcePath string) ([]byte, error) {
	input := &storedcontent.Input{ReportId: reportID, VersionNo: versionNo, Namespace: namespace, ResourcePath: resourcePath,
		Has: &storedcontent.InputHas{ReportId: true, VersionNo: true, Namespace: true, ResourcePath: true}}
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.content, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedcontent.Output)
	if !ok {
		return nil, fmt.Errorf("skill validation content returned %T", value)
	}
	if len(output.Files) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Files) != 1 || output.Files[0] == nil {
		return nil, fmt.Errorf("skill validation content returned an ambiguous identity")
	}
	file := output.Files[0]
	if file.ReportId != reportID || file.VersionNo != versionNo || file.Namespace != namespace || file.ResourcePath != resourcePath {
		return nil, fmt.Errorf("skill validation content returned a mismatched identity")
	}
	return file.Content, nil
}
