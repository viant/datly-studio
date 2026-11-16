package preview

import (
	"context"

	"github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/sqlx/io/read/cache/aerospike"
)

// initializedWarmup owns the same native cache initialization for inspection
// and execution. Only execution creates providers, prepares cases, or runs SQL.
type initializedWarmup struct {
	artifact *bootstrap.Artifact
	reader   dexec.Reader
	config   bootstrap.ReaderRuntimeConfig
	targets  []dexec.ReaderWarmupTarget
	pool     *aerospike.Pool
}

func (r *initializedWarmup) Close() { _ = r.pool.Close() }

func (d Dynamic) initializeWarmup(ctx context.Context, sources *sources, contract *transcribe.RuntimeContract) (*initializedWarmup, error) {
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Const: d.Const, Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources, CodecFactory: codecs.FromContext(ctx)})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "build dynamic reader artifact", Cause: err}
	}
	return d.initializeWarmupArtifact(sources, artifact)
}

func (d Dynamic) initializeWarmupArtifact(sources *sources, artifact *bootstrap.Artifact) (*initializedWarmup, error) {
	pool := &aerospike.Pool{}
	config := bootstrap.ReaderRuntimeConfig{SQL: sources.SQL, CacheSettings: d.ReaderCaches, Aerospike: pool}
	reader, err := artifact.ReaderCompilation().NewExecution(config)
	if err != nil {
		_ = pool.Close()
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize dynamic reader", Cause: err}
	}
	result := &initializedWarmup{artifact: artifact, reader: reader, config: config, pool: pool}
	if targeter, ok := reader.(dexec.ReaderWarmupTargeter); ok {
		result.targets = targeter.WarmupTargets()
	}
	return result, nil
}

// InspectEffectiveWarmup resolves native targets without providers, runtime
// registration, credential preparation, business reads, or warmup execution.
// The caller supplies existing exact-version warmup authorization.
func (d Dynamic) InspectEffectiveWarmup(ctx context.Context, reportID string, versionNo int) (*sdk.EffectiveWarmupInspection, error) {
	_, metadata, _, err := d.InspectExecutionMetadata(ctx, reportID, versionNo, false, true)
	return metadata, err
}

func inspectEffectiveWarmup(definition *definition, contract *transcribe.RuntimeContract, initialized *initializedWarmup) *sdk.EffectiveWarmupInspection {
	targets := make([]sdk.EffectiveWarmupTarget, 0, len(initialized.targets))
	for _, target := range initialized.targets {
		evidence := warmupTargetEvidence(contract.Component.RootView.Name, []dexec.ReaderWarmupTarget{target})
		projected := sdk.EffectiveWarmupTarget{WarmupScheduledTarget: evidence.ScheduledTargets[0]}
		if settings := target.Settings; settings != nil {
			projected.PolicyName = settings.EffectiveName()
			if settings.MaxCases != nil {
				value := *settings.MaxCases
				projected.MaxCases = &value
			}
			if settings.Limit != nil {
				value := *settings.Limit
				projected.RowLimit = &value
			}
			for _, item := range settings.Cases {
				if item == nil {
					continue
				}
				entry := sdk.EffectiveWarmupCase{Dimensions: []sdk.EffectiveWarmupDimension{}}
				for _, dimension := range item.Set {
					if dimension != nil {
						entry.Dimensions = append(entry.Dimensions, sdk.EffectiveWarmupDimension{Name: dimension.Name, ValueCount: len(dimension.Values), ExcludeDefault: dimension.ExcludeDefault})
					}
				}
				projected.Cases = append(projected.Cases, entry)
			}
		}
		targets = append(targets, projected)
	}
	return &sdk.EffectiveWarmupInspection{Status: "resolved", ReportID: definition.ReportID, VersionNo: definition.VersionNo, SourceRevision: definition.SourceRevision, Targets: &targets}
}
