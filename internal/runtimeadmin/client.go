package runtimeadmin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	agentlyconfig "github.com/viant/agently-core/protocol/mcp/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
)

// ErrNamespaceUnavailable means the runtime manager has no active listener for
// the selected namespace. It is distinct from an unreachable or invalid admin API.
var ErrNamespaceUnavailable = errors.New("namespace runtime is unavailable")

// Client talks only to the deployment-configured dynamic Datly admin origin.
// The token is never taken from an SDK or MCP request.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string, client *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("dynamic runtime admin origin is invalid")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("dynamic runtime admin token is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimSuffix(parsed.String(), "/"), token: token, http: client}, nil
}

func (c *Client) Reload(ctx context.Context, generation int64) error {
	if c == nil {
		return fmt.Errorf("dynamic runtime admin client is unavailable")
	}
	body := map[string]any{"generation": generation}
	if namespaceID, selected := sdk.NamespaceSelectionFromContext(ctx); selected {
		body["namespaceId"] = namespaceID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/_studio/reload", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Studio-Runtime-Token", c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("dynamic runtime reload returned %s", response.Status)
	}
	return nil
}

func (c *Client) ProbeRuntime(ctx context.Context) (*sdk.RuntimeHost, error) {
	if c == nil {
		return nil, fmt.Errorf("dynamic runtime admin client is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	endpoint := c.baseURL + "/_studio/status"
	if namespaceID, selected := sdk.NamespaceSelectionFromContext(ctx); selected {
		endpoint += "?namespaceId=" + url.QueryEscape(namespaceID)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Studio-Runtime-Token", c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNamespaceUnavailable
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("dynamic runtime status returned %s", response.Status)
	}
	var payload struct {
		NamespaceID        string `json:"namespaceId"`
		MCPURL             string `json:"mcpUrl"`
		AuthenticationMode string `json:"authenticationMode"`
		Status             string `json:"status"`
		Revision           int64  `json:"revision"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&payload); err != nil {
		return nil, err
	}
	if namespaceID, selected := sdk.NamespaceSelectionFromContext(ctx); selected && payload.NamespaceID != namespaceID {
		return nil, fmt.Errorf("dynamic runtime status namespace does not match selection")
	}
	if payload.Status != "ready" {
		return nil, fmt.Errorf("dynamic runtime is not ready")
	}
	if payload.MCPURL != "" {
		endpoint, parseErr := url.Parse(payload.MCPURL)
		validPath := endpoint != nil && endpoint.Path == "/mcp"
		if namespaceID, selected := sdk.NamespaceSelectionFromContext(ctx); selected && endpoint != nil {
			validPath = validPath || endpoint.Path == "/namespaces/"+namespaceID+"/mcp"
		}

		if parseErr != nil || endpoint.Host == "" || endpoint.User != nil ||
			(endpoint.Scheme != "http" && endpoint.Scheme != "https") || !validPath || endpoint.RawPath != "" ||
			endpoint.RawQuery != "" || endpoint.Fragment != "" {
			return nil, fmt.Errorf("dynamic runtime MCP endpoint is invalid")
		}
	}
	return &sdk.RuntimeHost{NamespaceID: payload.NamespaceID, MCPURL: payload.MCPURL, AuthenticationMode: payload.AuthenticationMode, Status: payload.Status, Revision: payload.Revision, CheckedAt: time.Now().UTC()}, nil
}

// Connection reads the deployed namespace's virtual Agently configuration.
// The caller must authorize namespace visibility before invoking this protected
// deployment API. No endpoint, credentials or URL may come from the caller's body.
func (c *Client) Connection(ctx context.Context) (json.RawMessage, error) {
	if c == nil {
		return nil, fmt.Errorf("dynamic runtime admin client is unavailable")
	}
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	decoded, err := hex.DecodeString(id)
	if !selected || err != nil || len(decoded) != 32 || strings.ToLower(id) != id {
		return nil, fmt.Errorf("canonical namespace selection is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/_studio/connection?namespaceId="+url.QueryEscape(id), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Studio-Runtime-Token", c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNamespaceUnavailable
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("namespace connection returned %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1024*1024 {
		return nil, fmt.Errorf("namespace connection exceeds size limit")
	}
	var payload struct {
		NamespaceID string          `json:"namespaceId"`
		Config      json.RawMessage `json:"config"`
	}
	if err = json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.NamespaceID != id {
		return nil, fmt.Errorf("namespace connection does not match selection")
	}
	var consumer agentlyconfig.MCPClient
	if err = json.Unmarshal(payload.Config, &consumer); err != nil {
		return nil, fmt.Errorf("invalid Agently MCP connection: %w", err)
	}
	if consumer.ClientOptions == nil || consumer.Name == "" || consumer.Transport.Type != "streamable" {
		return nil, fmt.Errorf("namespace connection must be a named streamable MCP client")
	}
	endpoint, err := url.Parse(consumer.Transport.URL)
	if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, fmt.Errorf("invalid namespace connection endpoint")
	}
	return append(json.RawMessage(nil), payload.Config...), nil
}
