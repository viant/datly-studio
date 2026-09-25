// auth-context-mock is a loopback-only synthetic auth-context fixture server.
// It is not an identity provider or production authorization service.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
	protocolclient "github.com/viant/mcp-protocol/client"
	"github.com/viant/mcp-protocol/logger"
	"github.com/viant/mcp-protocol/schema"
	protocolserver "github.com/viant/mcp-protocol/server"
	mcpserver "github.com/viant/mcp/server"
)

const maxRequestBytes = 16 << 10

type config struct {
	Address              string
	Scenario             string
	Delay                time.Duration
	Now                  func() time.Time
	RequireAuthorization bool
	AuthorizationHeader  string
}

type authorizedPrincipalKey struct{}

type contextRequest struct {
	PrincipalID   string `json:"principalId"`
	Tenant        string `json:"tenant"`
	Resource      string `json:"resource,omitempty"`
	Action        string `json:"action,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// entityID is a typed JSON string/integer union for synthetic IDs.
type entityID struct {
	Number *uint64
	Text   string
}

func (id entityID) MarshalJSON() ([]byte, error) {
	if id.Number != nil {
		return json.Marshal(*id.Number)
	}
	return json.Marshal(id.Text)
}

func numericID(value uint64) entityID { return entityID{Number: &value} }

type authContext struct {
	UserID          string                `json:"userId"`
	Tenant          string                `json:"tenant"`
	Roles           []string              `json:"roles"`
	Exposures       []string              `json:"exposures"`
	AllowedEntities map[string][]entityID `json:"allowedEntities"`
	ValidUntil      string                `json:"validUntil"`
}

type contextResponse struct {
	Context       authContext `json:"context"`
	CorrelationID string      `json:"correlationId,omitempty"`
}

type mockError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error mockError `json:"error"`
}

func main() {
	address := flag.String("address", "127.0.0.1:8098", "loopback listen address (port 0 is allowed)")
	scenario := flag.String("scenario", "normal", "mock scenario: normal, deny, malformed, delay")
	delay := flag.Duration("delay", 250*time.Millisecond, "delay scenario duration (0..2s)")
	requireAuthorization := flag.Bool("require-authorization", false, "require exact synthetic Bearer header on HTTP and every MCP request")
	authorizationHeader := flag.String("authorization-header", "Authorization", "incoming synthetic credential header name")
	flag.Parse()
	if *authorizationHeader == "" {
		log.Fatal("-authorization-header requires a non-empty header name")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, config{Address: *address, Scenario: *scenario, Delay: *delay, RequireAuthorization: *requireAuthorization, AuthorizationHeader: *authorizationHeader}); err != nil {
		log.Fatal(err)
	}
}

func validateConfig(c config) error {
	if !validHeaderName(headerName(c)) {
		return fmt.Errorf("-authorization-header must be a valid HTTP header name")
	}
	host, portText, err := net.SplitHostPort(c.Address)
	if err != nil || host == "" || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("-address must be a literal loopback IP and port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return fmt.Errorf("-address has an invalid port")
	}
	switch c.Scenario {
	case "normal", "deny", "malformed", "delay":
	default:
		return fmt.Errorf("unsupported mock scenario %q", c.Scenario)
	}
	if c.Delay < 0 || c.Delay > 2*time.Second {
		return fmt.Errorf("-delay must be between 0 and 2s")
	}
	return nil
}

func headerName(c config) string {
	if c.AuthorizationHeader == "" {
		return "Authorization"
	}
	return c.AuthorizationHeader
}

func validHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		default:
			return false
		}
	}
	return true
}

func run(ctx context.Context, c config) error {
	if err := validateConfig(c); err != nil {
		return err
	}
	handler, err := newHandler(ctx, c)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: time.Minute}
	serveCtx, stopServe := context.WithCancel(ctx)
	defer stopServe()
	shutdownDone := make(chan error, 1)
	go func() {
		<-serveCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()
	log.Printf("MOCK ONLY auth context HTTP http://%s/context and MCP http://%s/mcp", listener.Addr(), listener.Addr())
	serveErr := server.Serve(listener)
	stopServe()
	shutdownErr := <-shutdownDone
	if errors.Is(serveErr, http.ErrServerClosed) {
		return shutdownErr
	}
	return errors.Join(serveErr, shutdownErr)
}

func newHandler(ctx context.Context, c config) (http.Handler, error) {
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	newMCPHandler := func(_ context.Context, notifier transport.Notifier, log logger.Logger, ops protocolclient.Operations) (protocolserver.Handler, error) {
		base := protocolserver.NewDefaultHandler(notifier, log, ops)
		input := schema.ToolInputSchema{Type: "object", Properties: schema.ToolInputSchemaProperties{
			"principalId":   {"type": "string", "minLength": 1},
			"tenant":        {"type": "string", "minLength": 1},
			"resource":      {"type": "string"},
			"action":        {"type": "string"},
			"correlationId": {"type": "string"},
		}, Required: []string{"principalId", "tenant"}}
		base.Registry.RegisterToolWithSchema("auth_context", "MOCK ONLY: resolve a synthetic auth context", input, nil, func(callCtx context.Context, request *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			raw, err := json.Marshal(request.Params.Arguments)
			if err != nil {
				return mcpResult(nil, &mockError{Code: "invalid_request", Message: "invalid mock request"}), nil
			}
			input, failure := decodeRequest(raw)
			if failure != nil {
				return mcpResult(nil, failure), nil
			}
			output, failure := fixture(callCtx, c, input)
			return mcpResult(output, failure), nil
		})
		return base, nil
	}
	protocol, err := mcpserver.New(mcpserver.WithNewHandler(newMCPHandler), mcpserver.WithStreamableURI("/mcp"), mcpserver.WithRootRedirect(false))
	if err != nil {
		return nil, err
	}
	protocol.UseStreamableHTTP(true)
	mux := http.NewServeMux()
	mcpHandler := protocol.HTTP(ctx, "").Handler
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		mcpHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("POST /context", func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeHTTP(w, http.StatusBadRequest, errorResponse{Error: mockError{Code: "invalid_request", Message: "application/json is required"}})
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			writeHTTP(w, http.StatusRequestEntityTooLarge, errorResponse{Error: mockError{Code: "invalid_request", Message: "mock request is too large"}})
			return
		}
		input, failure := decodeRequest(raw)
		if failure != nil {
			writeHTTP(w, http.StatusBadRequest, errorResponse{Error: *failure})
			return
		}
		output, failure := fixture(r.Context(), c, input)
		if failure != nil {
			writeHTTP(w, failureStatus(failure), errorResponse{Error: *failure})
			return
		}
		writeHTTP(w, http.StatusOK, output)
	})
	if !c.RequireAuthorization {
		return mux, nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := authorizedPrincipal(r.Header.Values(headerName(c)))
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeHTTP(w, http.StatusUnauthorized, errorResponse{Error: mockError{Code: "unauthorized", Message: "synthetic Bearer credential required"}})
			return
		}
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authorizedPrincipalKey{}, principal)))
	}), nil
}

func authorizedPrincipal(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	switch values[0] {
	case "Bearer mock-alice":
		return "alice", true
	case "Bearer mock-bob":
		return "bob", true
	default:
		return "", false
	}
}

func decodeRequest(raw []byte) (contextRequest, *mockError) {
	var input contextRequest
	if len(raw) == 0 || len(raw) > maxRequestBytes {
		return input, &mockError{Code: "invalid_request", Message: "invalid mock request size"}
	}
	check := json.NewDecoder(bytes.NewReader(raw))
	opening, err := check.Token()
	if err != nil || opening != json.Delim('{') {
		return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
	}
	seen := map[string]bool{}
	for check.More() {
		keyToken, err := check.Token()
		if err != nil {
			return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
		}
		key, ok := keyToken.(string)
		if !ok || seen[key] {
			return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
		}
		seen[key] = true
		var value json.RawMessage
		if err := check.Decode(&value); err != nil {
			return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
		}
	}
	closing, err := check.Token()
	if err != nil || closing != json.Delim('}') {
		return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
	}
	if _, err := check.Token(); err != io.EOF {
		return input, &mockError{Code: "invalid_request", Message: "invalid mock request"}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return contextRequest{}, &mockError{Code: "invalid_request", Message: "invalid mock request"}
	}
	if _, err := decoder.Token(); err != io.EOF {
		return contextRequest{}, &mockError{Code: "invalid_request", Message: "invalid mock request"}
	}
	if input.PrincipalID == "" || input.Tenant == "" || len(input.PrincipalID) > 128 || len(input.Tenant) > 128 || len(input.Resource) > 256 || len(input.Action) > 128 || len(input.CorrelationID) > 128 {
		return contextRequest{}, &mockError{Code: "invalid_request", Message: "invalid mock request fields"}
	}
	return input, nil
}

// fixture is the one typed source of HTTP and MCP response content.
func fixture(ctx context.Context, c config, input contextRequest) (*contextResponse, *mockError) {
	if c.RequireAuthorization {
		principal, ok := ctx.Value(authorizedPrincipalKey{}).(string)
		if !ok {
			return nil, &mockError{Code: "unauthorized", Message: "synthetic Bearer credential required"}
		}
		if input.PrincipalID != principal {
			return nil, &mockError{Code: "identity_mismatch", Message: "request principal does not match synthetic credential"}
		}
	}
	if input.Tenant != "demo" || (input.PrincipalID != "alice" && input.PrincipalID != "bob") {
		return nil, &mockError{Code: "unknown_principal_or_tenant", Message: "unknown synthetic principal or tenant"}
	}
	if c.Scenario == "deny" {
		return nil, &mockError{Code: "denied", Message: "mock scenario denied"}
	}
	if c.Scenario == "delay" {
		timer := time.NewTimer(c.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, &mockError{Code: "cancelled", Message: "mock request cancelled"}
		case <-timer.C:
		}
	}
	result := &contextResponse{CorrelationID: input.CorrelationID, Context: authContext{
		UserID: input.PrincipalID, Tenant: input.Tenant,
		Roles: []string{"reader"}, Exposures: []string{"reports"},
		AllowedEntities: map[string][]entityID{"project": {numericID(101), numericID(102)}},
		ValidUntil:      c.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano),
	}}
	if input.PrincipalID == "bob" {
		result.Context.AllowedEntities["project"] = []entityID{numericID(103)}
	}
	if c.Scenario == "malformed" {
		result.Context.ValidUntil = "not-a-timestamp"
	}
	return result, nil
}

func failureStatus(failure *mockError) int {
	if failure.Code == "unknown_principal_or_tenant" || failure.Code == "denied" || failure.Code == "identity_mismatch" {
		return http.StatusForbidden
	}
	if failure.Code == "cancelled" {
		return http.StatusRequestTimeout
	}
	return http.StatusBadRequest
}

func mcpResult(output *contextResponse, failure *mockError) *schema.CallToolResult {
	var payload any = output
	failed := false
	if failure != nil {
		payload = errorResponse{Error: *failure}
		failed = true
	}
	raw, _ := json.Marshal(payload)
	return &schema.CallToolResult{IsError: &failed, StructuredContent: payload, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}}
}

func writeHTTP(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
