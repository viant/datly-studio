package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk"
)

func namespaceMCPTarget(transport sdk.Transport) bffauth.ProxyTargetResolver {
	return func(ctx context.Context, _ *http.Request) (*url.URL, error) {
		id, selected := sdk.NamespaceSelectionFromContext(ctx)
		if !selected {
			return nil, fmt.Errorf("namespace selection is required")
		}
		var status sdk.RuntimeStatus
		// The transport resolves namespace visibility from storage before probing
		// the deployment-configured admin service. No client URL is accepted.
		if err := transport.Invoke(ctx, sdk.OperationRuntimeStatus, nil, &status); err != nil {
			return nil, err
		}
		if status.Host == nil || status.Host.Status != "ready" || status.Host.NamespaceID != id || status.Host.MCPURL == "" {
			return nil, fmt.Errorf("namespace MCP endpoint is unavailable")
		}
		target, err := url.Parse(status.Host.MCPURL)
		if err != nil || target.Path != "/mcp" {
			return nil, fmt.Errorf("namespace MCP endpoint is invalid")
		}
		target.Path = ""
		return target, nil
	}
}

func nativeSDKTarget(target *url.URL) bffauth.ProxyTargetResolver {
	return func(ctx context.Context, request *http.Request) (*url.URL, error) {
		operation := strings.TrimPrefix(request.URL.Path, "/v1/studio/sdk/")
		if sdk.RequiresNamespaceSelection(operation) {
			if _, present := sdk.NamespaceSelectionFromContext(ctx); !present {
				return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "Choose a namespace before accessing resources"}
			}
		}
		return target, nil
	}
}
