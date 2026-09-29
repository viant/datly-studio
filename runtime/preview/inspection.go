package preview

import (
	"context"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

// InspectContract resolves embedded resources and discovers columns using the
// validation compiler. It never executes the component's business reader.
func (d Dynamic) InspectContract(ctx context.Context, reportID string, versionNo int) (*spec.Component, error) {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	return d.inspectDefinitionContract(ctx, definition)
}

// InspectCandidateContract resolves proposed authoring text before the native
// writer starts its transaction. Resource files remain tied to the exact version.
func (d Dynamic) InspectCandidateContract(ctx context.Context, reportID string, versionNo int, source string) (*spec.Component, error) {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	definition.DQL = source
	return d.inspectDefinitionContract(ctx, definition)
}

func (d Dynamic) inspectDefinitionContract(ctx context.Context, definition *definition) (*spec.Component, error) {
	sources, err := openSources(ctx, definition, definition.DQL)
	if err != nil {
		return nil, err
	}
	defer sources.Close()
	contract, err := d.runtimeContracts(ctx, &transcribe.Source{Types: d.Types, Scope: definition.Scope, Name: definition.Name, Text: definition.DQL, Connector: definition.Connector, Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections)})
	if err != nil {
		return nil, err
	}
	return contract.Component.Clone(), nil
}
