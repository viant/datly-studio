// Package hostconfig defines shared infrastructure configuration for Studio hosts.
// Application namespaces and published components belong to the catalog database.
package hostconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/scy"
	"io"
	"strings"

	datly "github.com/viant/datly/standalone/config"
	"go.yaml.in/yaml/v3"
)

// Config retains Datly's configuration names and types. Endpoint is the public
// HTTP listener; MCP is authoring/admin MCP. NamespaceMCP is the default shared
// listener for database-discovered publications. Embedders register extensions
// through Go APIs, not executable code or namespace lists in this document.
type Login struct {
	// ClientConfig loads the complete OAuth client through SCY, without plaintext files.
	ClientConfig     *scy.Resource
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     *scy.Resource
	Scopes           []string
}

type Config struct {
	datly.Config
	// Store is the one Studio metadata storage connection. Application data
	// connectors are managed in the database, outside deployment configuration.
	Store        *connector.Config
	NamespaceMCP *datly.MCP
	// PublicURL overrides the browser origin when deployed behind a proxy.
	PublicURL string
	// Tenant is the existing resource-policy tenant; empty defaults to studio.
	Tenant string
	// SessionKey is optional SCY material for stable browser sessions across restarts.
	// Omission generates a process-local key and invalidates sessions on restart.
	SessionKey       *scy.Resource
	StaticRoot       string
	Login            *Login
	Identity         *oauth.HTTPIdentityConfig
	EntityEvaluation *oauth.HTTPEvaluationConfig
}

// Decode accepts one YAML or JSON object without resolving secrets or opening
// connections. Datly remains responsible for connector/SCY initialization and
// source-resource resolution when the host builds its native runtime.
func Decode(data []byte) (*Config, error) {
	if len(data) == 0 || len(data) > 1<<20 {
		return nil, errors.New("host configuration must be between 1 byte and 1 MiB")
	}
	var document map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil || document == nil {
		return nil, errors.New("invalid host configuration object")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("host configuration requires one document")
	}
	for key := range document {
		if strings.EqualFold(key, "Connector") || strings.EqualFold(key, "Connectors") {
			return nil, errors.New("use singular Store for Studio metadata storage")
		}
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, errors.New("host configuration requires string mapping keys")
	}
	result := &Config{}
	jsonDecoder := json.NewDecoder(bytes.NewReader(raw))
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(result); err != nil {
		return nil, errors.New("host configuration contains an unknown field or invalid value")
	}
	if err := result.ValidateListeners(); err != nil {
		return nil, err
	}
	if _, err := result.CatalogConnector(); err != nil {
		return nil, err
	}
	return result, nil
}

// ValidateListeners uses the same validation as Datly's standalone listeners.
// Binding, connector initialization and authentication validation happen during
// startup; decoding alone does not establish a ready or authenticated host.
func (c *Config) ValidateListeners() error {
	if c == nil {
		return errors.New("host configuration is required")
	}
	if _, err := c.Endpoint.ListenAddress(); err != nil {
		return err
	}
	if c.MCP == nil || c.NamespaceMCP == nil {
		return errors.New("MCP and NamespaceMCP listeners are required")
	}
	if _, err := c.MCP.ListenAddress(); err != nil {
		return err
	}
	if _, err := c.NamespaceMCP.ListenAddress(); err != nil {
		return err
	}
	return nil
}
