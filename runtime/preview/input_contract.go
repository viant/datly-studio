package preview

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	mcptool "github.com/viant/datly/mcp/tool"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

// InspectPreviewInput compiles metadata only. It does not create an execution,
// invoke dependencies, execute the business reader, or run cache warmup.
// The caller supplies the same exact-version run authorization as Execute.
func (d Dynamic) InspectPreviewInput(ctx context.Context, reportID string, versionNo int) (*sdk.PreviewInputContract, error) {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	sources, err := openSources(ctx, definition, definition.DQL, d.Const)
	if err != nil {
		return nil, err
	}
	defer sources.Close()
	contract, err := d.runtimeContracts(ctx, &transcribe.Source{Types: definition.Types, Scope: definition.Scope, Name: definition.Name, Text: definition.DQL, Connector: definition.Connector, Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections)})
	if err != nil {
		return nil, err
	}
	if err = predicatepolicy.ValidateDeclarations(contract.Component, definition.Policy); err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Const: d.Const, Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources, CodecFactory: codecs.FromContext(ctx)})
	if err != nil {
		return nil, err
	}
	plan, err := previewInputPlan(contract, artifact)
	if err != nil {
		return nil, err
	}
	schema, err := json.Marshal(plan.Metadata().InputSchema)
	if err != nil {
		return nil, err
	}
	route := plan.Target().Route
	return &sdk.PreviewInputContract{ReportID: reportID, VersionNo: versionNo, SourceRevision: definition.SourceRevision, Method: route.Method, Path: route.Path, Schema: schema}, nil
}

// previewInputPlan is shared with execution: full-reader preview selects the
// first compiled route, independently of authored MCP exposures.
func previewInputPlan(contract *transcribe.RuntimeContract, artifact *bootstrap.Artifact) (*mcptool.Plan, error) {
	if len(contract.Component.Routes) == 0 {
		return nil, errors.New("preview route binding is unavailable")
	}
	route := contract.Component.Routes[0]
	input, found := artifact.Input.ForRoute(spec.RouteRef{Method: route.Method, Path: route.Path})
	if !found {
		return nil, errors.New("preview route binding is unavailable")
	}
	return mcptool.NewCompiler().Compile(mcptool.Input{Component: contract.Component.Key, Exposure: &spec.MCPExposure{Kind: spec.MCPExposureTool, Name: "preview"}, Contract: input, Fields: input.Fields(), TransportReady: true})
}
