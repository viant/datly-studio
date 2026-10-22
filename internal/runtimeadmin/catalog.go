package runtimeadmin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/mcp-protocol/schema"
)

// CatalogError preserves only the JSON-RPC code, never upstream diagnostic text.
type CatalogError struct{ Code int }

func (e *CatalogError) Error() string {
	return fmt.Sprintf("namespace catalog rejected request (%d)", e.Code)
}

// ReadCatalog uses the selected live namespace's ordinary MCP endpoint. The
// namespace server therefore performs its normal credential verification and
// resource policy checks. The caller must first authorize namespace visibility.
// Deployment credentials are sent only to ProbeRuntime, never to the MCP endpoint.
func (c *Client) ReadCatalog(ctx context.Context, bearer, method, uri string, cursor *string) (json.RawMessage, error) {
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	decoded, err := hex.DecodeString(id)
	if !selected || err != nil || len(decoded) != 32 || strings.ToLower(id) != id {
		return nil, fmt.Errorf("canonical namespace selection is required")
	}
	params := map[string]any{"_meta": map[string]any{
		"io.modelcontextprotocol/protocolVersion":    schema.LatestProtocolVersion,
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}}
	switch method {
	case "skills/list", "resources/list", "tools/list":
		if uri != "" {
			return nil, fmt.Errorf("catalog listing does not accept a resource URI")
		}
		if cursor != nil {
			params["cursor"] = *cursor
		}
	case "skills/get", "resources/read":
		if uri == "" || cursor != nil {
			return nil, fmt.Errorf("catalog retrieval requires a resource URI and no cursor")
		}
		params["uri"] = uri
	default:
		return nil, fmt.Errorf("unsupported read-only namespace catalog method")
	}
	return c.namespaceRPC(ctx, bearer, method, params, uri, 10*time.Second)
}

// CallTool forwards only the caller credential to the selected namespace. The
// deployment credential is used solely to resolve the trusted runtime endpoint.
func (c *Client) CallTool(ctx context.Context, bearer, name string, arguments map[string]any, timeout time.Duration) (json.RawMessage, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(bearer) == "" || timeout < 0 {
		return nil, fmt.Errorf("caller credential, tool name and nonnegative timeout are required")
	}
	params := map[string]any{"name": name, "arguments": arguments, "_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": schema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}
	return c.namespaceRPC(ctx, bearer, "tools/call", params, name, timeout)
}

func (c *Client) namespaceRPC(ctx context.Context, bearer, method string, params map[string]any, name string, timeout time.Duration) (json.RawMessage, error) {
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	decoded, err := hex.DecodeString(id)
	if !selected || err != nil || len(decoded) != 32 || strings.ToLower(id) != id {
		return nil, fmt.Errorf("canonical namespace selection is required")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	host, err := c.ProbeRuntime(ctx)
	if err != nil {
		return nil, err
	}
	if host.MCPURL == "" {
		return nil, ErrNamespaceUnavailable
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, host.MCPURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", schema.LatestProtocolVersion)
	request.Header.Set("Mcp-Method", method)
	if name != "" {
		request.Header.Set("Mcp-Name", name)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	// Never forward the caller's credential through an HTTP redirect.
	client := *c.http
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("namespace catalog transport failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("namespace catalog returned HTTP %d", response.StatusCode)
	}
	const maxResponse = 24 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("namespace catalog response exceeds size limit")
	}
	if strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		var found []byte
		for _, line := range bytes.Split(data, []byte("\n")) {
			if bytes.HasPrefix(line, []byte("data:")) {
				candidate := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
				var message struct {
					ID int `json:"id"`
				}
				if json.Unmarshal(candidate, &message) == nil && message.ID == 1 {
					found = candidate
					break
				}
			}
		}
		data = found
	}
	var rpc struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &rpc) != nil || rpc.JSONRPC != "2.0" || rpc.ID != 1 {
		return nil, fmt.Errorf("invalid namespace catalog response")
	}
	if rpc.Error != nil {
		return nil, &CatalogError{Code: rpc.Error.Code}
	}
	if len(rpc.Result) == 0 || bytes.Equal(rpc.Result, []byte("null")) {
		return nil, fmt.Errorf("namespace catalog result is missing")
	}
	return append(json.RawMessage(nil), rpc.Result...), nil
}
