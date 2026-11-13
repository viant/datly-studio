package predicatepolicy

import (
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/spec"
)

type InternalComponentRequiredError struct{}

func (*InternalComponentRequiredError) Error() string {
	return "namespace requires internal DQL components without direct HTTP or MCP exposure"
}

// Validate compiled contracts, including generated cube exposure. Source text and
// client-supplied component names are not authority for this namespace policy.
func ValidateInternalComponents(component *spec.Component, policy *settings.Namespace) error {
	if policy == nil || policy.Security == nil || !policy.Security.RequireInternalComponents {
		return nil
	}
	if component == nil || len(component.Routes) == 0 {
		return &InternalComponentRequiredError{}
	}
	for _, route := range component.Routes {
		if route == nil || !route.Internal || len(route.MCP) != 0 {
			return &InternalComponentRequiredError{}
		}
	}
	if component.Settings != nil && component.Settings.Report != nil && component.Settings.Report.Enabled && component.RootView != nil && component.RootView.Groupable != nil && *component.RootView.Groupable {
		report := component.Settings.Report
		if !report.LinkedFacade && (report.MCPTool == nil || *report.MCPTool) {
			return &InternalComponentRequiredError{}
		}
		if compose := report.Compose; compose != nil && compose.Enabled && (compose.MCPTool == nil || *compose.MCPTool) {
			return &InternalComponentRequiredError{}
		}
	}
	return nil
}
