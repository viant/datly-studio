package host

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	active "github.com/viant/datly-studio/studio/runtime_generations/store_active"
	candidate "github.com/viant/datly-studio/studio/runtime_generations/store_candidate"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// publishedDefinitionStore reads the exact active or candidate publication
// snapshot through server-only Datly v1 components. It is not a public route.
type publishedDefinitionStore struct {
	runtime   *druntime.Runtime
	active    exec.ComponentTarget
	candidate exec.ComponentTarget
}

func newPublishedDefinitionStore(db *sql.DB) (*publishedDefinitionStore, error) {
	resources := resource.New()
	if err := resources.Register(active.DefinitionDatlyResourceNamespace, active.DefinitionDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(candidate.DefinitionDatlyResourceNamespace, candidate.DefinitionDatlyResources); err != nil {
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
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{activeRegistration, candidateRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &publishedDefinitionStore{runtime: runtime, active: activeTarget, candidate: candidateTarget}, nil
}

func (s *publishedDefinitionStore) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Shutdown(ctx)
}

func (s *publishedDefinitionStore) Definitions(ctx context.Context, generation *int64) ([]definition, error) {
	if s == nil || s.runtime == nil {
		return nil, fmt.Errorf("published definition store is unavailable")
	}
	var rows []definition
	if generation == nil {
		value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.active, Input: &active.Input{}})
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
			rows = append(rows, mapped)
		}
		return rows, nil
	}
	if *generation <= 0 {
		return nil, fmt.Errorf("candidate generation must be positive")
	}
	value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.candidate,
		Input: &candidate.Input{CandidateGeneration: *generation, Has: &candidate.InputHas{CandidateGeneration: true}}})
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
		rows = append(rows, mapped)
	}
	return rows, nil
}

func mapPublishedDefinition(reportID string, versionNo int, scope, name, connector, driver string,
	dsn *string, secretRef, generated, authored string) (definition, error) {
	if reportID == "" || versionNo <= 0 || scope == "" || name == "" || connector == "" || driver == "" || dsn == nil {
		return definition{}, fmt.Errorf("published definition %q has incomplete runtime metadata", reportID)
	}
	dql := generated
	if strings.TrimSpace(dql) == "" {
		dql = authored
	}
	if strings.TrimSpace(dql) == "" {
		return definition{}, fmt.Errorf("published report %s has no DQL", reportID)
	}
	return definition{reportID: reportID, versionNo: versionNo, scope: scope, name: name,
		connector: connector, driver: driver, dsn: *dsn, secretRef: secretRef, dql: dql}, nil
}
