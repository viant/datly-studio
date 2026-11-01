package predicatepolicy_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/spec"
	"testing"
)

func TestNamespaceInternalComponentsPolicy(t *testing.T) {
	strict := &settings.Namespace{Security: &settings.Security{RequireInternalComponents: true}}
	public := &spec.Component{Routes: []*spec.Route{{Method: "GET", Path: "/records"}}}
	require.NoError(t, predicatepolicy.ValidateDeclarations(public, nil))
	require.NoError(t, predicatepolicy.ValidateDeclarations(public, &settings.Namespace{Security: &settings.Security{}}))
	require.ErrorAs(t, predicatepolicy.ValidateDeclarations(public, strict), new(*predicatepolicy.InternalComponentRequiredError))
	require.Error(t, predicatepolicy.ValidateDeclarations(nil, strict))
	public.Routes[0].Internal = true
	require.NoError(t, predicatepolicy.ValidateDeclarations(public, strict))
	public.Routes = append(public.Routes, &spec.Route{Method: "POST", Path: "/records"})
	require.Error(t, predicatepolicy.ValidateDeclarations(public, strict))
	public.Routes = public.Routes[:1]
	public.Routes[0].MCP = []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "records"}}
	require.Error(t, predicatepolicy.ValidateDeclarations(public, strict))
}

func TestNamespaceInternalComponentsRejectsImplicitCubeExposure(t *testing.T) {
	strict := &settings.Namespace{Security: &settings.Security{RequireInternalComponents: true}}
	yes, no := true, false
	c := &spec.Component{Routes: []*spec.Route{{Method: "GET", Path: "/records", Internal: true}}, RootView: &spec.View{Groupable: &yes}, Settings: &spec.Settings{Report: &spec.ReportSettings{Enabled: true}}}
	require.Error(t, predicatepolicy.ValidateDeclarations(c, strict))
	c.Settings.Report.MCPTool = &no
	require.NoError(t, predicatepolicy.ValidateDeclarations(c, strict))
	c.Settings.Report.Compose = &spec.CubeComposeSettings{Enabled: true}
	require.Error(t, predicatepolicy.ValidateDeclarations(c, strict))
	c.Settings.Report.Compose.MCPTool = &no
	require.NoError(t, predicatepolicy.ValidateDeclarations(c, strict))
	c.Settings.Report.Compose.MCPTool = &yes
	require.Error(t, predicatepolicy.ValidateDeclarations(c, strict))
}
