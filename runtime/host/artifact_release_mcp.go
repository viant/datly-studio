package host

import (
	"context"
	"encoding/json"
	"fmt"

	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
)

const artifactReleaseCapabilityName = "io.viant/artifact-releases/v1"

func artifactReleaseCapability(c *schema.ServerCapabilities) {
	if c.Extensions == nil {
		c.Extensions = map[string]map[string]interface{}{}
	}
	c.Extensions[artifactReleaseCapabilityName] = map[string]interface{}{
		"schemaVersion": 1,
		"methods":       []string{contract.MethodList, contract.MethodCandidates, contract.MethodPublish, contract.MethodPromote, contract.MethodHistory},
	}
}

func releaseFailure() *schema.CallToolResult {
	failed := true
	value := map[string]string{"code": "release_unavailable", "message": "release operation unavailable"}
	return &schema.CallToolResult{IsError: &failed, StructuredContent: value, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: `{"code":"release_unavailable","message":"release operation unavailable"}`}}}
}

// registerReleaseTools is opt-in. The manager resolves the verified editor and
// account scope itself; tools/call fields can select only namespace/channel and
// exact candidate bytes, never trusted tenant/account/workspace authority.
func (s *Service) registerReleaseTools(base *protoserver.DefaultHandler) error {
	if s == nil || s.config.ReleaseManagement == nil {
		return nil
	}
	if base == nil || s.config.ArtifactCohorts == nil {
		return fmt.Errorf("release management requires an authenticated artifact catalog")
	}
	manager := s.config.ReleaseManagement
	register := func(name string, prototype any, run func(context.Context, map[string]any) (any, error)) error {
		var input schema.ToolInputSchema
		if err := input.Load(prototype); err != nil {
			return err
		}
		base.Registry.RegisterToolWithSchema(name, "Manage a named published artifact release", input, nil, func(ctx context.Context, request *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			if request == nil {
				return releaseFailure(), nil
			}
			value, err := run(ctx, request.Params.Arguments)
			if err != nil {
				return releaseFailure(), nil
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return releaseFailure(), nil
			}
			failed := false
			return &schema.CallToolResult{IsError: &failed, StructuredContent: value, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}}, nil
		})
		return nil
	}
	if err := register(contract.MethodList, &contract.ListInput{}, func(ctx context.Context, args map[string]any) (any, error) {
		var in contract.ListInput
		err := decodePrimitiveArgs(args, &in)
		if err != nil {
			return nil, err
		}
		if err = in.Validate(); err != nil {
			return nil, err
		}
		return manager.List(ctx, in)
	}); err != nil {
		return err
	}
	if err := register(contract.MethodCandidates, &contract.CandidatesInput{}, func(ctx context.Context, args map[string]any) (any, error) {
		var in contract.CandidatesInput
		err := decodePrimitiveArgs(args, &in)
		if err != nil {
			return nil, err
		}
		if err = in.Validate(); err != nil {
			return nil, err
		}
		return manager.Candidates(ctx, in)
	}); err != nil {
		return err
	}
	if err := register(contract.MethodPublish, &contract.PublishInput{}, func(ctx context.Context, args map[string]any) (any, error) {
		var in contract.PublishInput
		err := decodePrimitiveArgs(args, &in)
		if err != nil {
			return nil, err
		}
		if err = in.Validate(); err != nil {
			return nil, err
		}
		return manager.Publish(ctx, in)
	}); err != nil {
		return err
	}
	if err := register(contract.MethodPromote, &contract.PromoteInput{}, func(ctx context.Context, args map[string]any) (any, error) {
		var in contract.PromoteInput
		err := decodePrimitiveArgs(args, &in)
		if err != nil {
			return nil, err
		}
		if err = in.Validate(); err != nil {
			return nil, err
		}
		return manager.Promote(ctx, in)
	}); err != nil {
		return err
	}
	return register(contract.MethodHistory, &contract.HistoryInput{}, func(ctx context.Context, args map[string]any) (any, error) {
		var in contract.HistoryInput
		err := decodePrimitiveArgs(args, &in)
		if err != nil {
			return nil, err
		}
		if err = in.Validate(); err != nil {
			return nil, err
		}
		return manager.History(ctx, in)
	})
}
