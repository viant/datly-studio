package preview

import (
	"context"

	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
)

// InspectExecutionMetadata shares one exact source snapshot, contract, artifact,
// and connector lifetime. Callers authorize each requested projection first.
// Projection errors are independent; neither business reads nor warmup run here.
func (d Dynamic) InspectExecutionMetadata(ctx context.Context, reportID string, versionNo int, includePreviewInput, includeEffectiveWarmup bool) (input *sdk.PreviewInputContract, warmup *sdk.EffectiveWarmupInspection, inputErr, warmupErr error) {
	if !includePreviewInput && !includeEffectiveWarmup {
		return
	}
	fail := func(err error) (*sdk.PreviewInputContract, *sdk.EffectiveWarmupInspection, error, error) {
		if includePreviewInput {
			inputErr = err
		}
		if includeEffectiveWarmup {
			warmupErr = err
		}
		return nil, nil, inputErr, warmupErr
	}
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return fail(err)
	}
	sources, err := openSources(ctx, definition, definition.DQL, d.Const)
	if err != nil {
		return fail(err)
	}
	defer sources.Close()
	contract, err := d.previewContract(ctx, definition, sources, definition.DQL, false, includePreviewInput)
	if err != nil {
		return fail(err)
	}
	if err = predicatepolicy.ValidateInternalComponents(contract.Component, definition.Policy); err != nil {
		return fail(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Const: d.Const, Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources, CodecFactory: codecs.FromContext(ctx)})
	if err != nil {
		return fail(err)
	}
	if includePreviewInput {
		input, inputErr = inspectPreviewInput(definition, contract, artifact)
	}
	if includeEffectiveWarmup {
		initialized, err := d.initializeWarmupArtifact(sources, artifact)
		if err != nil {
			warmupErr = err
		} else {
			defer initialized.Close()
			warmup = inspectEffectiveWarmup(definition, contract, initialized)
		}
	}
	return
}
