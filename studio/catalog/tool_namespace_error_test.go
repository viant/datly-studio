package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly/bootstrap"
	datlymcp "github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/xdatly/response"
	"testing"
)

func TestToolNamespaceValidationHasActionablePublicBody(t *testing.T) {
	_, _, err := toolRuntime(context.Background(), nil, nil, nil, "analytics")
	if err == nil {
		t.Fatal("invalid namespace accepted")
	}
	var typed *response.Error
	if !errors.As(err, &typed) || typed.StatusCode() != 400 || errors.Unwrap(typed) == nil {
		t.Fatalf("lost typed error: %v", err)
	}
	body, ok := response.ErrorBody(err)
	if !ok {
		t.Fatal("missing public body")
	}
	raw, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	var decoded map[string]any
	if e = json.Unmarshal(raw, &decoded); e != nil {
		t.Fatal(e)
	}
	if decoded["code"] != "invalid_namespace_id" || decoded["field"] != "namespaceId" || decoded["hint"] == "" || decoded["message"] == "" {
		t.Fatalf("unactionable body: %s", raw)
	}
}

func TestCatalogFailureHasSafePublicBody(t *testing.T) {
	err := fail(503, "Namespace skills could not be loaded.")
	body, ok := response.ErrorBody(err)
	if !ok || body == nil {
		t.Fatal("native MCP would return a null error body")
	}
	raw, marshalErr := json.Marshal(body)
	if marshalErr != nil || string(raw) != `{"message":"Namespace skills could not be loaded."}` || response.ErrorStatusCode(err, 500) != 503 {
		t.Fatalf("status or public payload changed: %s", raw)
	}
}

func TestCatalogFailureMCPEnvelopePreservesPublicExplanation(t *testing.T) {
	component := &spec.Component{
		Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/catalog", Name: "skills"}, Name: "skills",
		Routes: []*spec.Route{{Method: "POST", Path: "/catalog/skills", MCP: []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "catalog_skills"}}}},
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[struct{}](), OutputType: reflect.TypeFor[SkillListOutput](), HandlerOwnedOutput: true})
	require.NoError(t, err)
	cause := errors.New("private upstream details")
	failure := errors.Join(fail(503, "Namespace skills could not be loaded."), cause)
	require.ErrorIs(t, failure, cause)
	entry, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.NewFunc[struct{}, SkillListOutput](func(context.Context, *struct{}) (*SkillListOutput, error) {
		return new(SkillListOutput), failure
	})})
	require.NoError(t, err)
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry})
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	service, err := datlymcp.New(datlymcp.Config{Components: []*registry.RegisteredComponent{entry}, Invoker: runtime})
	require.NoError(t, err)
	tool, ok := service.Registry().ToolRegistry.Get("catalog_skills")
	require.True(t, ok)
	result, rpcErr := tool.Handler(t.Context(), &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "catalog_skills"}})
	require.Nil(t, rpcErr)
	require.NotNil(t, result.IsError)
	require.True(t, *result.IsError)
	require.NotNil(t, result.StructuredContent)
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(schema.TextContent)
	require.True(t, ok)
	require.JSONEq(t, `{"message":"Namespace skills could not be loaded."}`, text.Text)
	require.NotContains(t, text.Text, "private upstream")
}
