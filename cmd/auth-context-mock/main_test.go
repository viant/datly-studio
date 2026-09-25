package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	streamable "github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type wireContext struct {
	Context struct {
		UserID          string                   `json:"userId"`
		Tenant          string                   `json:"tenant"`
		Roles           []string                 `json:"roles"`
		Exposures       []string                 `json:"exposures"`
		AllowedEntities map[string][]json.Number `json:"allowedEntities"`
		ValidUntil      string                   `json:"validUntil"`
	} `json:"context"`
	CorrelationID string `json:"correlationId"`
}

func testServer(t *testing.T, scenario string) *httptest.Server {
	t.Helper()
	handler, err := newHandler(context.Background(), config{Address: "127.0.0.1:0", Scenario: scenario, Delay: 10 * time.Millisecond, Now: func() time.Time { return time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func postContext(t *testing.T, server *httptest.Server, payload string) (int, []byte) {
	t.Helper()
	response, err := server.Client().Post(server.URL+"/context", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, body
}

func testMCPClient(t *testing.T, server *httptest.Server) *mcpclient.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithHandshakeTimeout(3*time.Second))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	client := mcpclient.New("auth-context-mock-test", "1.0.0", wire)
	if _, err := client.Initialize(ctx); err != nil {
		client.Close()
		_ = wire.Close()
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close(); _ = wire.Close(); cancel() })
	return client
}

func callContext(t *testing.T, client *mcpclient.Client, args map[string]any) *schema.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "auth_context", Arguments: args})
	if err != nil || result == nil {
		t.Fatalf("MCP call: %+v %v", result, err)
	}
	return result
}

func decodeWire(t *testing.T, payload any) wireContext {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var out wireContext
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHTTPAndMCPReturnSameSyntheticContexts(t *testing.T) {
	server := testServer(t, "normal")
	client := testMCPClient(t, server)
	catalog, err := client.ListTools(context.Background(), nil)
	if err != nil || len(catalog.Tools) != 1 || catalog.Tools[0].Name != "auth_context" {
		t.Fatalf("tool catalog %+v: %v", catalog, err)
	}
	for _, tc := range []struct {
		principal string
		projects  []json.Number
	}{
		{principal: "alice", projects: []json.Number{"101", "102"}},
		{principal: "bob", projects: []json.Number{"103"}},
	} {
		t.Run(tc.principal, func(t *testing.T) {
			input := contextRequest{PrincipalID: tc.principal, Tenant: "demo", Resource: "sample", Action: "read", CorrelationID: "request-1"}
			rawInput, _ := json.Marshal(input)
			status, httpBody := postContext(t, server, string(rawInput))
			if status != 200 {
				t.Fatalf("HTTP %d: %s", status, httpBody)
			}
			var httpOutput wireContext
			if err := json.Unmarshal(httpBody, &httpOutput); err != nil {
				t.Fatal(err)
			}
			mcpOutput := callContext(t, client, map[string]any{"principalId": tc.principal, "tenant": "demo", "resource": "sample", "action": "read", "correlationId": "request-1"})
			if mcpOutput.IsError != nil && *mcpOutput.IsError {
				t.Fatalf("MCP error %+v", mcpOutput)
			}
			structured := decodeWire(t, mcpOutput.StructuredContent)
			if !reflect.DeepEqual(httpOutput, structured) {
				t.Fatalf("transport mismatch: HTTP %+v MCP %+v", httpOutput, structured)
			}
			if httpOutput.Context.UserID != tc.principal || httpOutput.Context.Tenant != "demo" || !reflect.DeepEqual(httpOutput.Context.AllowedEntities["project"], tc.projects) || httpOutput.CorrelationID != "request-1" {
				t.Fatalf("unexpected context %+v", httpOutput)
			}
			if !reflect.DeepEqual(httpOutput.Context.Roles, []string{"reader"}) || !reflect.DeepEqual(httpOutput.Context.Exposures, []string{"reports"}) || httpOutput.Context.ValidUntil != "2026-09-23T12:05:00Z" {
				t.Fatalf("typed fields %+v", httpOutput.Context)
			}
		})
	}
}

func TestMockRejectsUnknownAndMalformedRequests(t *testing.T) {
	server := testServer(t, "normal")
	client := testMCPClient(t, server)
	for _, payload := range []string{
		`{}`, `{"principalId":"alice","tenant":"other"}`, `{"principalId":"mallory","tenant":"demo"}`,
		`{"principalId":"alice","tenant":"demo","url":"file:///secret"}`,
		`{"principalId":"mallory","principalId":"alice","tenant":"demo"}`,
		`{"principalId":5,"tenant":"demo"}`, `{"principalId":"alice","tenant":"demo"} {}`,
	} {
		status, body := postContext(t, server, payload)
		if status < 400 || !bytes.Contains(body, []byte(`"error"`)) {
			t.Fatalf("accepted %s: %d %s", payload, status, body)
		}
	}
	for _, args := range []map[string]any{
		{"principalId": "mallory", "tenant": "demo"},
		{"principalId": "alice", "tenant": "other"},
		{"principalId": "alice", "tenant": "demo", "url": "file:///secret"},
	} {
		result := callContext(t, client, args)
		if result.IsError == nil || !*result.IsError {
			t.Fatalf("accepted MCP input %+v: %+v", args, result)
		}
	}
	large := `{"principalId":"alice","tenant":"demo","resource":"` + strings.Repeat("x", maxRequestBytes) + `"}`
	if status, _ := postContext(t, server, large); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("large HTTP status %d", status)
	}
}

func TestNegativeScenariosAndLoopbackValidation(t *testing.T) {
	for _, address := range []string{"0.0.0.0:8098", "192.0.2.1:8098", "localhost:8098", "127.0.0.1:65536", "127.0.0.1:-1"} {
		if err := validateConfig(config{Address: address, Scenario: "normal"}); err == nil {
			t.Fatalf("accepted address %q", address)
		}
	}
	if err := validateConfig(config{Address: "127.0.0.1:0", Scenario: "normal"}); err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(config{Address: "127.0.0.1:0", Scenario: "delay", Delay: 3 * time.Second}); err == nil {
		t.Fatal("unbounded delay accepted")
	}
	for _, scenario := range []string{"deny", "malformed", "delay"} {
		t.Run(scenario, func(t *testing.T) {
			server := testServer(t, scenario)
			client := testMCPClient(t, server)
			status, body := postContext(t, server, `{"principalId":"alice","tenant":"demo"}`)
			result := callContext(t, client, map[string]any{"principalId": "alice", "tenant": "demo"})
			if scenario == "deny" {
				if status != http.StatusForbidden || result.IsError == nil || !*result.IsError || !bytes.Contains(body, []byte(`"denied"`)) {
					t.Fatalf("deny: HTTP %d %s MCP %+v", status, body, result)
				}
				return
			}
			if status != 200 || (result.IsError != nil && *result.IsError) {
				t.Fatalf("scenario %s: HTTP %d %s MCP %+v", scenario, status, body, result)
			}
			var output wireContext
			if err := json.Unmarshal(body, &output); err != nil {
				t.Fatal(err)
			}
			if scenario == "malformed" && output.Context.ValidUntil != "not-a-timestamp" {
				t.Fatalf("malformed scenario %+v", output)
			}
		})
	}
}

func TestRunStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, config{Address: "127.0.0.1:0", Scenario: "normal"}) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("mock did not shut down")
	}
}

type testRoundTrip func(*http.Request) (*http.Response, error)

func (f testRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAuthorizationHeaderOnHTTP(t *testing.T) {
	handler, err := newHandler(context.Background(), config{Address: "127.0.0.1:0", Scenario: "normal", RequireAuthorization: true})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	request := func(headers []string, principal string) (int, []byte) {
		t.Helper()
		body, _ := json.Marshal(contextRequest{PrincipalID: principal, Tenant: "demo"})
		req, err := http.NewRequest(http.MethodPost, server.URL+"/context", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for _, header := range headers {
			req.Header.Add("Authorization", header)
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, data
	}
	for _, headers := range [][]string{nil, {"Bearer unknown"}, {"bearer mock-alice"}, {"Bearer mock-alice", "Bearer mock-bob"}, {"Bearer mock-alice extra"}} {
		status, body := request(headers, "alice")
		if status != http.StatusUnauthorized || !bytes.Contains(body, []byte(`"unauthorized"`)) {
			t.Fatalf("headers %v: %d %s", headers, status, body)
		}
	}
	if status, body := request([]string{"Bearer mock-alice"}, "bob"); status != http.StatusForbidden || !bytes.Contains(body, []byte(`"identity_mismatch"`)) {
		t.Fatalf("mismatch: %d %s", status, body)
	}
	if status, body := request([]string{"Bearer mock-alice"}, "alice"); status != http.StatusOK || !bytes.Contains(body, []byte(`"userId":"alice"`)) {
		t.Fatalf("alice: %d %s", status, body)
	}
	if status, body := request([]string{"Bearer mock-bob"}, "bob"); status != http.StatusOK || !bytes.Contains(body, []byte(`"project":[103]`)) {
		t.Fatalf("bob: %d %s", status, body)
	}
}

func TestAuthorizationHeaderOnEveryMCPRequest(t *testing.T) {
	handler, err := newHandler(context.Background(), config{Address: "127.0.0.1:0", Scenario: "normal", RequireAuthorization: true})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	var credential atomic.Value
	credential.Store("")
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: testRoundTrip(func(req *http.Request) (*http.Response, error) {
		copy := req.Clone(req.Context())
		copy.Header = req.Header.Clone()
		if value := credential.Load().(string); value != "" {
			copy.Header.Set("Authorization", value)
		}
		return http.DefaultTransport.RoundTrip(copy)
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// The transport may contact /mcp during construction or Initialize; either
	// way a missing header must prevent a successful MCP handshake.
	unauthorizedWire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithHTTPClient(httpClient), streamable.WithHandshakeTimeout(2*time.Second))
	if err == nil {
		unauthorizedClient := mcpclient.New("mock-unauthorized", "1.0.0", unauthorizedWire)
		if _, err = unauthorizedClient.Initialize(ctx); err == nil {
			t.Fatal("MCP handshake succeeded without Authorization")
		}
		unauthorizedClient.Close()
		_ = unauthorizedWire.Close()
	}
	credential.Store("Bearer mock-alice")
	wire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithHTTPClient(httpClient), streamable.WithHandshakeTimeout(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer wire.Close()
	client := mcpclient.New("mock-authorized", "1.0.0", wire)
	defer client.Close()
	if _, err := client.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	credential.Store("")
	if _, err := client.ListTools(ctx, nil); err == nil {
		t.Fatal("MCP discovery succeeded without Authorization")
	}
	credential.Store("Bearer unknown")
	if _, err := client.ListTools(ctx, nil); err == nil {
		t.Fatal("MCP discovery accepted unknown credential")
	}
	credential.Store("Bearer mock-alice")
	catalog, err := client.ListTools(ctx, nil)
	if err != nil || len(catalog.Tools) != 1 {
		t.Fatalf("authorized discovery: %+v %v", catalog, err)
	}
	credential.Store("")
	if _, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "auth_context", Arguments: map[string]any{"principalId": "alice", "tenant": "demo"}}); err == nil {
		t.Fatal("MCP tool call succeeded without Authorization")
	}
	credential.Store("Bearer mock-alice")
	alice := callContext(t, client, map[string]any{"principalId": "alice", "tenant": "demo"})
	if alice.IsError != nil && *alice.IsError {
		t.Fatalf("alice tool call: %+v", alice)
	}
	credential.Store("Bearer mock-bob")
	mismatch := callContext(t, client, map[string]any{"principalId": "alice", "tenant": "demo"})
	if mismatch.IsError == nil || !*mismatch.IsError {
		t.Fatalf("switched principal reused alice: %+v", mismatch)
	}
	mismatchBody, _ := json.Marshal(mismatch.StructuredContent)
	if !bytes.Contains(mismatchBody, []byte(`"identity_mismatch"`)) {
		t.Fatalf("wrong mismatch error: %s", mismatchBody)
	}
	bob := callContext(t, client, map[string]any{"principalId": "bob", "tenant": "demo"})
	if bob.IsError != nil && *bob.IsError {
		t.Fatalf("bob tool call: %+v", bob)
	}
	bobBody, _ := json.Marshal(bob.StructuredContent)
	if !bytes.Contains(bobBody, []byte(`"project":[103]`)) || bytes.Contains(bobBody, []byte(`"project":[101,102]`)) {
		t.Fatalf("wrong switched context: %s", bobBody)
	}
}

func TestConfigurableAuthorizationHeader(t *testing.T) {
	for _, name := range []string{"", "Authorization", "X-Mock-Authorization"} {
		if err := validateConfig(config{Address: "127.0.0.1:0", Scenario: "normal", AuthorizationHeader: name}); err != nil {
			t.Fatalf("valid header %q: %v", name, err)
		}
	}
	for _, name := range []string{"X Mock", "Bad:Header", "é", "X\nBad"} {
		if err := validateConfig(config{Address: "127.0.0.1:0", Scenario: "normal", AuthorizationHeader: name}); err == nil {
			t.Fatalf("invalid header name accepted: %q", name)
		}
	}
	handler, err := newHandler(context.Background(), config{Address: "127.0.0.1:0", Scenario: "normal", RequireAuthorization: true, AuthorizationHeader: "X-Mock-Authorization"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	post := func(header string) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, server.URL+"/context", strings.NewReader(`{"principalId":"alice","tenant":"demo"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if header != "" {
			req.Header.Set(header, "Bearer mock-alice")
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		return response.StatusCode
	}
	if got := post(""); got != 401 {
		t.Fatalf("missing custom HTTP header: %d", got)
	}
	if got := post("Authorization"); got != 401 {
		t.Fatalf("default HTTP header bypassed custom option: %d", got)
	}
	if got := post("X-Mock-Authorization"); got != 200 {
		t.Fatalf("custom HTTP header: %d", got)
	}
	var credential atomic.Value
	credential.Store("")
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: testRoundTrip(func(req *http.Request) (*http.Response, error) {
		copy := req.Clone(req.Context())
		copy.Header = req.Header.Clone()
		if value := credential.Load().(string); value != "" {
			copy.Header.Set("X-Mock-Authorization", value)
		}
		return http.DefaultTransport.RoundTrip(copy)
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	unauthorizedWire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithHTTPClient(httpClient), streamable.WithHandshakeTimeout(2*time.Second))
	if err == nil {
		unauthorizedClient := mcpclient.New("custom-missing", "1.0.0", unauthorizedWire)
		if _, err := unauthorizedClient.Initialize(ctx); err == nil {
			t.Fatal("custom MCP handshake succeeded without header")
		}
		unauthorizedClient.Close()
		_ = unauthorizedWire.Close()
	}
	credential.Store("Bearer mock-alice")
	wire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithHTTPClient(httpClient), streamable.WithHandshakeTimeout(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer wire.Close()
	client := mcpclient.New("custom-auth", "1.0.0", wire)
	defer client.Close()
	if _, err := client.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	credential.Store("")
	if _, err := client.ListTools(ctx, nil); err == nil {
		t.Fatal("custom MCP discovery succeeded without header")
	}
	credential.Store("Bearer mock-alice")
	if catalog, err := client.ListTools(ctx, nil); err != nil || len(catalog.Tools) != 1 {
		t.Fatalf("custom MCP discovery %+v %v", catalog, err)
	}
	credential.Store("")
	if _, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "auth_context", Arguments: map[string]any{"principalId": "alice", "tenant": "demo"}}); err == nil {
		t.Fatal("custom MCP call succeeded without header")
	}
	credential.Store("Bearer mock-bob")
	if result := callContext(t, client, map[string]any{"principalId": "alice", "tenant": "demo"}); result.IsError == nil || !*result.IsError {
		t.Fatalf("custom header identity mismatch: %+v", result)
	}
	if result := callContext(t, client, map[string]any{"principalId": "bob", "tenant": "demo"}); result.IsError != nil && *result.IsError {
		t.Fatalf("custom header bob: %+v", result)
	}
}
