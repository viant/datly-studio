package preview

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	mcptool "github.com/viant/datly/mcp/tool"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
)

// InspectPreviewInput compiles metadata only. It does not create an execution,
// invoke dependencies, execute the business reader, or run cache warmup.
// The caller supplies the same exact-version run authorization as Execute.
func (d Dynamic) InspectPreviewInput(ctx context.Context, reportID string, versionNo int) (*sdk.PreviewInputContract, error) {
	metadata, _, err, _ := d.InspectExecutionMetadata(ctx, reportID, versionNo, true, false)
	return metadata, err
}

func inspectPreviewInput(definition *definition, contract *transcribe.RuntimeContract, artifact *bootstrap.Artifact) (*sdk.PreviewInputContract, error) {
	plan, err := previewInputPlan(contract, artifact)
	if err != nil {
		return nil, err
	}
	schema, err := json.Marshal(plan.Metadata().InputSchema)
	if err != nil {
		return nil, err
	}
	route := plan.Target().Route
	return &sdk.PreviewInputContract{ReportID: definition.ReportID, VersionNo: definition.VersionNo, SourceRevision: definition.SourceRevision, Method: route.Method, Path: route.Path, Schema: schema}, nil
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
