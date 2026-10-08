package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"

	"github.com/viant/authz/oauth"
	forgehandler "github.com/viant/forge/backend/mcp/mcp"
	forgeservice "github.com/viant/forge/backend/mcp/service"
	mcpserver "github.com/viant/mcp/server"
)

func (s *Service) forgeProviderHTTP(ctx context.Context) (http.Handler, error) {
	provider := s.config.ForgeProvider
	if provider == nil || provider.Host == nil || provider.Authority == nil {
		return nil, fmt.Errorf("Forge provider requires a host and invocation authority")
	}
	protocol, err := mcpserver.New(mcpserver.WithNewHandler(forgehandler.NewPortableHandler(
		forgeservice.NewService(&forgeservice.Config{PortableProvider: provider, UseData: true}))))
	if err != nil {
		return nil, err
	}
	protocol.UseStreamableHTTP(true)
	return browserMCPCORS(s.config.MCP.CORS, s.authenticated(protocol.HTTP(ctx, "").Handler)), nil
}

// ExecutePublishedJSON rejects unbound active-route execution.
// Deprecated: active-route dispatch is not a revision-bound datasource API.
// Use ResolveComponentBinding and ExecuteComponentJSON with the authorized pin.
func (s *Service) ExecutePublishedJSON(context.Context, string, string, map[string]any) (json.RawMessage, error) {
	return nil, fmt.Errorf("an explicit component revision binding is required")
}

func executeRuntimeJSON(ctx context.Context, handler http.Handler, method, route string, inputs map[string]any) (json.RawMessage, error) {
	if ctx == nil || ctx.Err() != nil || handler == nil {
		return nil, fmt.Errorf("component runtime unavailable")
	}
	data, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}
	if method == http.MethodGet {
		query, err := publishedQuery(inputs)
		if err != nil {
			return nil, err
		}
		if encoded := query.Encode(); encoded != "" {
			route += "?" + encoded
		}
		data = nil
	}
	request, err := http.NewRequestWithContext(ctx, method, route, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if token := oauth.Bearer(ctx); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code < 200 || response.Code >= 300 || !json.Valid(response.Body.Bytes()) {
		return nil, fmt.Errorf("component datasource execution failed (%d)", response.Code)
	}
	return append(json.RawMessage(nil), response.Body.Bytes()...), nil
}

// Structured query parameters are JSON values; fmt.Sprint(map) cannot be
// decoded by a published component's JSON binding. Scalar arrays retain normal
// repeated-query semantics and exact json.Number strings are not rounded.
func publishedQuery(inputs map[string]any) (url.Values, error) {
	query := url.Values{}
	add := func(key string, value any) error {
		if value == nil {
			return nil
		}
		kind := reflect.TypeOf(value).Kind()
		switch kind {
		case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
			query.Add(key, fmt.Sprint(value))
		default:
			encoded, err := json.Marshal(value)
			if err != nil {
				return err
			}
			query.Add(key, string(encoded))
		}
		return nil
	}
	for key, value := range inputs {
		if value == nil {
			continue
		}
		items := reflect.ValueOf(value)
		if items.Kind() == reflect.Array || items.Kind() == reflect.Slice {
			for index := 0; index < items.Len(); index++ {
				if err := add(key, items.Index(index).Interface()); err != nil {
					return nil, err
				}
			}
		} else if err := add(key, value); err != nil {
			return nil, err
		}
	}
	return query, nil
}
