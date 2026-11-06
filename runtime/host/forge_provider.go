package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"

	"github.com/viant/authz/oauth"
	requestprovider "github.com/viant/bindly/provider/request"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly/application"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/output"
	"github.com/viant/datly/spec"
	forgeservice "github.com/viant/forge/backend/provider"
	forgehandler "github.com/viant/forge/backend/provider/mcp"
	"github.com/viant/jsonrpc/transport"
	protoclient "github.com/viant/mcp-protocol/client"
	"github.com/viant/mcp-protocol/logger"
	protoserver "github.com/viant/mcp-protocol/server"
	mcpserver "github.com/viant/mcp/server"
	xexec "github.com/viant/xdatly/exec"
	xresponse "github.com/viant/xdatly/response"
)

func (s *Service) forgeProviderHTTP(ctx context.Context) (http.Handler, error) {
	provider := s.config.ForgeProvider
	if provider == nil || provider.Host == nil || provider.Authority == nil {
		return nil, fmt.Errorf("Forge provider requires a host and invocation authority")
	}
	if s.config.ReleaseManagement != nil {
		controller, ok := s.config.ReleaseManagement.(*sdkrelease.Controller)
		catalog, sourceOK := s.config.ArtifactCohorts.(*PublishedArtifactCatalog)
		if !ok || controller == nil || !sourceOK || catalog == nil || catalog.Publisher != controller.Publisher() {
			return nil, fmt.Errorf("release management requires the matching trusted catalog controller")
		}
		if err := catalog.BindReleaseGate(controller.Gate()); err != nil {
			return nil, err
		}
		if err := controller.Recover(ctx); err != nil {
			return nil, fmt.Errorf("reconcile release channels: %w", err)
		}
		s.releaseSyncController = controller
	}
	legacy := forgehandler.NewProviderHandler(forgeservice.NewService(&forgeservice.Config{PrimitiveProvider: provider, UseData: true}))
	protocol, err := mcpserver.New(mcpserver.WithOnSessionClose(func(notifier transport.Notifier) {
		if source, ok := s.config.ArtifactCohorts.(interface{ UnsubscribeResources(transport.Notifier) }); ok {
			source.UnsubscribeResources(notifier)
		}
	}), mcpserver.WithNewHandler(func(ctx context.Context, notifier transport.Notifier, log logger.Logger, client protoclient.Operations) (protoserver.Handler, error) {
		handler, err := legacy(ctx, notifier, log, client)
		if err != nil {
			return nil, err
		}
		if err := s.registerWindowPrimitives(handler.(*forgehandler.Handler).DefaultHandler); err != nil {
			return nil, err
		}
		var cohort *artifactCohortHandler
		if s.config.ArtifactCohorts != nil {
			cohort = &artifactCohortHandler{notifier: notifier, releases: s.config.ReleaseManagement != nil}
			if source, ok := s.config.ArtifactCohorts.(interface {
				SubscribeResources(transport.Notifier) func()
			}); ok {
				cohort.source = source
			}
		}
		var activate func()
		if cohort != nil {
			activate = cohort.activateNotifications
		}
		if err := s.registerArtifactCohortTools(ctx, handler.(*forgehandler.Handler).DefaultHandler, activate); err != nil {
			return nil, err
		}
		if err := s.registerReleaseTools(handler.(*forgehandler.Handler).DefaultHandler); err != nil {
			return nil, err
		}
		if s.windowPrimitiveSource != nil && len(s.windowPrimitiveSource.datasources) > 0 {
			handler = &stockResourceHandler{Handler: handler, host: s}
		}
		if cohort != nil {
			cohort.Handler = handler
			handler = cohort
		}
		return handler, nil
	}))
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

// Internal exact datasource calls retain transport authentication and native
// binding/encoding without adding the route to an external HTTP or MCP catalog.
func (s *Service) executeInternalComponentJSON(ctx context.Context, manager *application.Manager, metadata *exactComponentMetadata, ref ComponentReference, inputs map[string]any) (json.RawMessage, error) {
	return executeRuntimeJSON(ctx, s.internalComponentJSONHandler(manager, metadata, ref), ref.Method, ref.Route, inputs)
}

func (s *Service) internalComponentJSONHandler(manager *application.Manager, metadata *exactComponentMetadata, ref ComponentReference) http.Handler {
	target := dexec.ComponentTarget{Component: metadata.registered.Component.Key, Route: spec.RouteRef{Method: ref.Method, Path: ref.Route}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := dexec.CaptureOutputSelection(xexec.WithContext(r.Context(), xexec.NewContext(r.Method, r.RequestURI, r.Header, ref.Revision)))
		r = r.WithContext(ctx)
		options := []requestprovider.Option{}
		if r.Method == http.MethodGet {
			options = append(options, requestprovider.WithForm(r.URL.Query()))
		}
		scope, err := requestprovider.New(r, options...)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer scope.Close()
		invoker := &exactNativeInvoker{manager: manager, metadata: metadata, authority: ctx, targets: map[dexec.ComponentTarget]bool{target: true}, active: true}
		defer invoker.close()
		value, err := invoker.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Providers: scope.Providers()})
		if err != nil {
			code := xresponse.ErrorStatusCode(err, http.StatusInternalServerError)
			http.Error(w, http.StatusText(nativeComponentStatus(code)), nativeComponentStatus(code))
			return
		}
		status := xexec.GetContext(ctx).StatusCode
		if response, ok := value.(xresponse.Response); ok {
			writeNativeComponentResponse(w, response, status)
			return
		}
		plan := metadata.registered.Output
		if plan == nil {
			plan, err = (output.Compiler{}).Compile(output.CompileInput{Component: metadata.registered.Component, Type: metadata.registered.OutputType})
		}
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		encoded, err := plan.Encode(ctx, "json", value)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", encoded.ContentType)
		w.WriteHeader(nativeComponentStatus(status))
		_, _ = w.Write(encoded.Data)
	})
	return s.authenticated(handler)
}

// Transport-ready outputs retain the existing Xdatly response contract rather
// than being serialized as their Go wrapper. Status reaches the ordinary JSON
// dispatch checker, which withholds every non-success body.
func writeNativeComponentResponse(writer http.ResponseWriter, response xresponse.Response, status int) {
	if size := response.Size(); size > 0 && writer.Header().Get("Content-Length") == "" {
		writer.Header().Set("Content-Length", strconv.Itoa(size))
	}
	for key, values := range response.Headers() {
		writer.Header().Del(key)
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	if compressed, ok := response.(xresponse.Compressed); ok && compressed.CompressionType() != "" {
		writer.Header().Set("Content-Encoding", compressed.CompressionType())
	}
	if status == 0 || status == http.StatusOK {
		status = response.StatusCode()
	}
	writer.WriteHeader(nativeComponentStatus(status))
	if body := response.Body(); body != nil {
		_, _ = io.Copy(writer, body)
	}
}

func nativeComponentStatus(code int) int {
	if code == 0 {
		return http.StatusOK
	}
	if code < 100 || code > 999 {
		return http.StatusInternalServerError
	}
	return code
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
