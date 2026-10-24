package host

import (
	"context"
	"testing"

	"github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
)

type releaseManagerFixture struct{ listed string }

var _ artifactrelease.Manager = (*releaseManagerFixture)(nil)

func (f *releaseManagerFixture) List(_ context.Context, in contract.ListInput) (contract.ListOutput, error) {
	f.listed = in.Namespace
	return contract.ListOutput{}, nil
}
func (*releaseManagerFixture) Candidates(context.Context, contract.CandidatesInput) (contract.CandidatesOutput, error) {
	return contract.CandidatesOutput{}, nil
}
func (*releaseManagerFixture) Publish(context.Context, contract.PublishInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (*releaseManagerFixture) Promote(context.Context, contract.PromoteInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (*releaseManagerFixture) History(context.Context, contract.HistoryInput) (contract.HistoryOutput, error) {
	return contract.HistoryOutput{}, nil
}

func TestArtifactReleaseToolsAreOptInAndUseTypedDTO(t *testing.T) {
	ctx := context.Background()
	base := protoserver.NewDefaultHandler(nil, nil, nil)
	s := &Service{config: Config{}}
	if err := s.registerReleaseTools(base); err != nil {
		t.Fatal(err)
	}
	if len(base.ListRegisteredTools()) != 0 {
		t.Fatal("release methods advertised without manager")
	}
	manager := &releaseManagerFixture{}
	s.config.ReleaseManagement = manager
	s.config.ArtifactCohorts = NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	if err := s.registerReleaseTools(base); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range base.ListRegisteredTools() {
		names[tool.Name] = true
	}
	for _, name := range []string{contract.MethodList, contract.MethodCandidates, contract.MethodPublish, contract.MethodPromote, contract.MethodHistory} {
		if !names[name] {
			t.Fatalf("missing %s", name)
		}
	}
	request := &jsonrpc.TypedRequest[*schema.CallToolRequest]{Request: &schema.CallToolRequest{Params: schema.CallToolRequestParams{Name: contract.MethodList, Arguments: map[string]any{"namespace": "demo"}}}}
	result, callErr := base.CallTool(ctx, request)
	if callErr != nil || result == nil || result.IsError == nil || *result.IsError || manager.listed != "demo" {
		t.Fatalf("typed management request not dispatched: %+v %v", result, callErr)
	}
	request.Request.Params.Arguments = map[string]any{"namespace": ""}
	result, callErr = base.CallTool(ctx, request)
	if callErr != nil || result == nil || result.IsError == nil || !*result.IsError || manager.listed != "demo" {
		t.Fatal("invalid namespace reached manager")
	}
	capabilities := schema.ServerCapabilities{}
	artifactReleaseCapability(&capabilities)
	if capabilities.Extensions[artifactReleaseCapabilityName]["schemaVersion"] != 1 {
		t.Fatal("release capability missing")
	}
}
