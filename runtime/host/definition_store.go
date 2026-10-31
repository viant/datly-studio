package host

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/authz"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	presence "github.com/viant/datly-studio/studio/resource_policy/presence"
	active "github.com/viant/datly-studio/studio/runtime_generations/store_active"
	candidate "github.com/viant/datly-studio/studio/runtime_generations/store_candidate"
	generationcatalog "github.com/viant/datly-studio/studio/runtime_generations/store_catalog"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// publishedDefinitionStore reads the exact active or candidate publication
// snapshot through server-only Datly v1 components. It is not a public route.
type publishedDefinitionStore struct {
	runtime           *druntime.Runtime
	active            exec.ComponentTarget
	candidate         exec.ComponentTarget
	policyPresence    exec.ComponentTarget
	generationCatalog exec.ComponentTarget
}

func newPublishedDefinitionStore(db *sql.DB) (*publishedDefinitionStore, error) {
	resources := resource.New()
	if err := resources.Register(active.DefinitionDatlyResourceNamespace, active.DefinitionDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(candidate.DefinitionDatlyResourceNamespace, candidate.DefinitionDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(presence.HeadDatlyResourceNamespace, presence.HeadDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(generationcatalog.GenerationDatlyResourceNamespace, generationcatalog.GenerationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	activeRegistration, activeTarget, err := readercomponent.Compile(
		reflect.TypeOf(active.DefinitionComponent{}), "store_active",
		reflect.TypeOf(active.Input{}), reflect.TypeOf(active.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	candidateRegistration, candidateTarget, err := readercomponent.Compile(
		reflect.TypeOf(candidate.DefinitionComponent{}), "store_candidate",
		reflect.TypeOf(candidate.Input{}), reflect.TypeOf(candidate.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	presenceRegistration, presenceTarget, err := readercomponent.Compile(reflect.TypeFor[presence.HeadComponent](), "presence", reflect.TypeFor[presence.Input](), reflect.TypeFor[presence.Output](), resources, connector)
	if err != nil {
		return nil, err
	}
	generationRegistration, generationTarget, err := readercomponent.Compile(reflect.TypeFor[generationcatalog.GenerationComponent](), "store_catalog", reflect.TypeFor[generationcatalog.Input](), reflect.TypeFor[generationcatalog.Output](), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{activeRegistration, candidateRegistration, presenceRegistration, generationRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &publishedDefinitionStore{runtime: runtime, active: activeTarget, candidate: candidateTarget, policyPresence: presenceTarget, generationCatalog: generationTarget}, nil
}

func (s *publishedDefinitionStore) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Shutdown(ctx)
}

func (s *publishedDefinitionStore) Definitions(ctx context.Context, generation *int64, namespaceID string) ([]definition, error) {
	if s == nil || s.runtime == nil {
		return nil, fmt.Errorf("published definition store is unavailable")
	}
	var rows []definition
	if generation == nil {
		value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.active, Input: &active.Input{NamespaceId: namespaceID, Has: &active.InputHas{NamespaceId: true}}})
		if err != nil {
			return nil, err
		}
		output, ok := value.(*active.Output)
		if !ok {
			return nil, fmt.Errorf("active definition reader returned %T", value)
		}
		for _, item := range output.Definitions {
			if item == nil {
				return nil, fmt.Errorf("active definition reader returned nil row")
			}
			mapped, err := mapPublishedDefinition(item.ReportId, item.VersionNo, item.ComponentScope, item.ComponentName,
				item.DefaultConnectorName, item.Driver, item.DsnTemplate, item.SecretRef, item.GeneratedDql, item.AuthoredDql)
			if err != nil {
				return nil, err
			}
			mapped.owner, mapped.namespace, mapped.resourceCheck = item.OwnerId, item.Namespace, item.ResourceCheckEnabled
			mapped.sourceRevision, mapped.authoredDQL = item.SourceRevision, item.AuthoredDql
			rows = append(rows, mapped)
		}
		return rows, nil
	}
	if *generation <= 0 {
		return nil, fmt.Errorf("candidate generation must be positive")
	}
	if err := s.requireGeneration(ctx, namespaceID, *generation); err != nil {
		return nil, err
	}
	value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.candidate,
		Input: &candidate.Input{NamespaceId: namespaceID, CandidateGeneration: *generation, Has: &candidate.InputHas{NamespaceId: true, CandidateGeneration: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*candidate.Output)
	if !ok {
		return nil, fmt.Errorf("candidate definition reader returned %T", value)
	}
	for _, item := range output.Definitions {
		if item == nil {
			return nil, fmt.Errorf("candidate definition reader returned nil row")
		}
		mapped, err := mapPublishedDefinition(item.ReportId, item.VersionNo, item.ComponentScope, item.ComponentName,
			item.DefaultConnectorName, item.Driver, item.DsnTemplate, item.SecretRef, item.GeneratedDql, item.AuthoredDql)
		if err != nil {
			return nil, err
		}
		mapped.owner, mapped.namespace, mapped.resourceCheck = item.OwnerId, item.Namespace, item.ResourceCheckEnabled
		mapped.sourceRevision, mapped.authoredDQL = item.SourceRevision, item.AuthoredDql
		rows = append(rows, mapped)
	}
	return rows, nil
}

// Positive loads are server-selected publication snapshots, including an active
// generation used by SDK compensation. Never silently substitute active rows
// for a missing, foreign, failed or retired candidate.
func (s *publishedDefinitionStore) requireGeneration(ctx context.Context, namespaceID string, generation int64) error {
	value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.generationCatalog, Input: &generationcatalog.Input{
		NamespaceId: namespaceID, GenerationNo: generation,
		Has: &generationcatalog.InputHas{NamespaceId: true, GenerationNo: true},
	}})
	if err != nil {
		return err
	}
	output, ok := value.(*generationcatalog.Output)
	if !ok || len(output.Generations) != 1 || output.Generations[0] == nil || output.Generations[0].GenerationNo != generation ||
		(output.Generations[0].Status != "building" && output.Generations[0].Status != "active") {
		return fmt.Errorf("requested runtime generation is unavailable for this namespace")
	}
	return nil
}

func mapPublishedDefinition(reportID string, versionNo int, scope, name, connector, driver string,
	dsn *string, secretRef, generated, authored string) (definition, error) {
	if reportID == "" || versionNo <= 0 || scope == "" || name == "" || connector == "" || driver == "" || (dsn == nil && strings.TrimSpace(secretRef) == "") {
		return definition{}, fmt.Errorf("published definition %q has incomplete runtime metadata", reportID)
	}
	dsnValue := ""
	if dsn != nil {
		dsnValue = *dsn
	}
	dql := generated
	if strings.TrimSpace(dql) == "" {
		dql = authored
	}
	if strings.TrimSpace(dql) == "" {
		return definition{}, fmt.Errorf("published report %s has no DQL", reportID)
	}
	return definition{reportID: reportID, versionNo: versionNo, scope: scope, name: name,
		connector: connector, driver: driver, dsn: dsnValue, secretRef: secretRef, dql: dql}, nil
}

func (s *publishedDefinitionStore) HasPolicy(ctx context.Context, r authz.Resource) (bool, error) {
	if s == nil || s.runtime == nil {
		return false, fmt.Errorf("policy presence reader is unavailable")
	}
	value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.policyPresence, Input: &presence.Input{Tenant: r.Tenant, ResourceKind: r.Kind, ResourceId: r.ID, Has: &presence.InputHas{Tenant: true, ResourceKind: true, ResourceId: true}}})
	if err != nil {
		return false, err
	}
	output, ok := value.(*presence.Output)
	if !ok || output == nil || len(output.Heads) > 1 {
		return false, fmt.Errorf("policy presence reader returned invalid %T", value)
	}
	if len(output.Heads) == 0 {
		return false, nil
	}
	if output.Heads[0] == nil || output.Heads[0].ResourceId != r.ID {
		return false, fmt.Errorf("policy presence identity mismatch")
	}
	return true, nil
}
