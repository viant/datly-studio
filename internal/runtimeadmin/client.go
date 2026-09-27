package runtimeadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
)

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
	payload, err := json.Marshal(map[string]int64{"generation": generation})
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
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/_studio/status", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Studio-Runtime-Token", c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("dynamic runtime status returned %s", response.Status)
	}
	var payload struct {
		AuthenticationMode string `json:"authenticationMode"`
		Status             string `json:"status"`
		Revision           int64  `json:"revision"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Status != "ready" {
		return nil, fmt.Errorf("dynamic runtime is not ready")
	}
	return &sdk.RuntimeHost{AuthenticationMode: payload.AuthenticationMode, Status: payload.Status, Revision: payload.Revision, CheckedAt: time.Now().UTC()}, nil
}
