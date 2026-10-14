package host

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/namespaceskill"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/namespace_skills/store_published"
	dexec "github.com/viant/datly/exec"
	mcpresource "github.com/viant/datly/mcp/resource"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// namespaceSkillFolders loads only immutable published revisions belonging to
// this listener. A root runtime never aggregates other namespaces' skills.
func (s *Service) namespaceSkillFolders(ctx context.Context) ([]mcpresource.Folder, map[string]bool, error) {
	if s.config.NamespaceID == "" {
		return nil, nil, nil
	}
	resources := resource.New()
	if err := resources.Register(stored.SkillDatlyResourceNamespace, stored.SkillDatlyResources); err != nil {
		return nil, nil, err
	}
	connector := &dsql.SQLComponent{DB: s.studio}
	if err := connector.RegisterConnector("studio", s.studio); err != nil {
		return nil, nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeFor[stored.SkillComponent](), "store_published", reflect.TypeFor[stored.Input](), reflect.TypeFor[stored.Output](), resources, connector)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, nil, err
	}
	defer runtime.Shutdown(context.Background())
	var folders []mcpresource.Folder
	uris := map[string]bool{}
	for offset := 0; ; offset += 100 {
		input := &stored.Input{}
		input.SetNamespaceId(s.config.NamespaceID)
		input.SetPageLimit(100)
		input.SetPageOffset(offset)
		value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
		if err != nil {
			return nil, nil, err
		}
		output, ok := value.(*stored.Output)
		if !ok || output == nil {
			return nil, nil, fmt.Errorf("namespace skill reader returned %T", value)
		}
		for _, row := range output.Skills {
			if row == nil || row.NamespaceId != s.config.NamespaceID {
				return nil, nil, fmt.Errorf("namespace skill reader returned mismatched identity")
			}
			snapshot, err := namespaceskill.Restore(ctx, row.NamespaceId, row.SkillId, row.Revision, row.BundleJson, row.ContentSha256)
			if err != nil {
				return nil, nil, fmt.Errorf("published namespace skill %s revision %d: %w", row.SkillId, row.Revision, err)
			}
			if snapshot.Name != row.Name {
				return nil, nil, fmt.Errorf("published namespace skill name does not match immutable identity")
			}
			folders = append(folders, snapshot.Folder)
			for _, plan := range snapshot.Plans {
				if uris[plan.URI()] {
					return nil, nil, fmt.Errorf("duplicate namespace skill resource URI")
				}
				uris[plan.URI()] = true
			}
			if len(uris) > 4096 {
				return nil, nil, fmt.Errorf("namespace skills exceed MCP resource inventory limit")
			}
		}
		if len(output.Skills) < 100 {
			break
		}
	}
	return folders, uris, nil
}

// namespaceSkillAuthorizer uses exact, generation-owned file identities. A
// prefix match would let a component folder impersonate namespace ownership.
func (s *Service) namespaceSkillAuthorizer(uris map[string]bool, fallback func(context.Context, string, string) error) func(context.Context, string, string) error {
	owned := make(map[string]bool, len(uris))
	for uri := range uris {
		owned[uri] = true
	}
	return func(ctx context.Context, uri, action string) error {
		if !owned[uri] {
			return fallback(ctx, uri, action)
		}
		if err := validateResourceURI(uri); err != nil {
			return err
		}
		if err := s.authorizeNamespace(ctx); err != nil {
			return err
		}
		if s.config.Access != nil && len(s.config.Access.ResourceBindings) > 0 {
			return s.authorizeBoundResourceAction(ctx, uri, action)
		}
		return nil
	}
}
