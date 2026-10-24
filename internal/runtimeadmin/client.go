package runtimeadmin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mcpconfig "github.com/viant/mcp/client/config"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	dexec "github.com/viant/datly/exec"
)

// ErrNamespaceUnavailable means the runtime manager has no active listener for
// the selected namespace. It is distinct from an unreachable or invalid admin API.
var ErrNamespaceUnavailable = errors.New("namespace runtime is unavailable. Open Namespaces, edit the selected namespace, enable MCP, and save. If MCP is already enabled, check its runtime status. Then retry Release on the same component version")

// Client talks only to the deployment-configured dynamic Datly admin origin.
// The token is never taken from an SDK or MCP request.
type Client struct {
	reload     func(context.Context, int64) error
	probe      func(context.Context) (*sdk.RuntimeHost, error)
	connection func(context.Context) (json.RawMessage, error)
	invalidate func(context.Context, string, int, string, string) ([]dexec.CacheInvalidation, error)

	baseURL string
	token   string
	http    *http.Client
}

// NewInProcess binds trusted callbacks; no listener or deployment token exists.
func NewInProcess(reload func(context.Context, int64) error, probe func(context.Context) (*sdk.RuntimeHost, error), connection func(context.Context) (json.RawMessage, error)) *Client {
	return &Client{reload: reload, probe: probe, connection: connection, http: http.DefaultClient}
}

// NewInProcessWithCache adds a trusted namespace control callback. The caller
// still performs the existing SDK publish authorization before invoking it.
func NewInProcessWithCache(reload func(context.Context, int64) error, probe func(context.Context) (*sdk.RuntimeHost, error), connection func(context.Context) (json.RawMessage, error), invalidate func(context.Context, string, int, string, string) ([]dexec.CacheInvalidation, error)) *Client {
	return &Client{reload: reload, probe: probe, connection: connection, invalidate: invalidate, http: http.DefaultClient}
}

type contextKey struct{}

func WithClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}
func Resolve(ctx context.Context) (*Client, error) {
	if client, ok := ctx.Value(contextKey{}).(*Client); ok && client != nil {
		return client, nil
	}
	origin := os.Getenv("STUDIO_DYNAMIC_HTTP_URL")
	if origin == "" {
		origin = "http://127.0.0.1:8082"
	}
	return New(origin, os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN"), nil)
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
	if c.reload != nil {
		return c.reload(ctx, generation)
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
	if response.StatusCode == http.StatusNotFound {
		if _, selected := sdk.NamespaceSelectionFromContext(ctx); selected {
			return ErrNamespaceUnavailable
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("dynamic runtime reload returned %s", response.Status)
	}
	return nil
}

func (c *Client) ProbeRuntime(ctx context.Context) (*sdk.RuntimeHost, error) {
	if c == nil {
		return nil, fmt.Errorf("dynamic runtime admin client is unavailable")
	}
	if c.probe != nil {
		return c.probe(ctx)
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
	if c.connection != nil {
		return c.connection(ctx)
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
	var consumer mcpconfig.MCPClient
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

// InvalidatePublishedCacheView is a control-plane operation only. Its callback
// is installed by the process that owns the selected namespace runtime.
func (c *Client) InvalidatePublishedCacheView(ctx context.Context, reportID string, version int, routePath, view string) ([]dexec.CacheInvalidation, error) {
	if c == nil || reportID == "" || version < 1 || routePath == "" || view == "" {
		return nil, fmt.Errorf("exact published cache view is required")
	}
	if c.invalidate != nil {
		return c.invalidate(ctx, reportID, version, routePath, view)
	}
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	decoded, err := hex.DecodeString(id)
	if !selected || err != nil || len(decoded) != 32 || id != strings.ToLower(id) {
		return nil, fmt.Errorf("canonical namespace selection is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	body, err := json.Marshal(struct {
		NamespaceID string `json:"namespaceId"`
		ReportID    string `json:"reportId"`
		VersionNo   int    `json:"versionNo"`
		RoutePath   string `json:"routePath"`
		View        string `json:"view"`
	}{id, reportID, version, routePath, view})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/_studio/cache/invalidate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Studio-Runtime-Token", c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("namespace cache invalidation returned %s", response.Status)
	}
	var payload struct {
		NamespaceID string                    `json:"namespaceId"`
		Invalidated []dexec.CacheInvalidation `json:"invalidated"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&payload); err != nil || payload.NamespaceID != id || len(payload.Invalidated) != 1 || payload.Invalidated[0].View != view {
		return nil, fmt.Errorf("namespace cache invalidation response differs from exact view")
	}
	return payload.Invalidated, nil
}
