package host

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	access "github.com/viant/authz"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	forgemcp "github.com/viant/forge/backend/mcp/service"
	mcpprotocol "github.com/viant/mcp/server"
	"go.yaml.in/yaml/v3"
)

const LocalDevelopmentAdminToken = "datly-studio-local-runtime"

type Listener struct {
	Address string            `yaml:"Address"`
	CORS    *mcpprotocol.Cors `yaml:"CORS,omitempty"`
}
type Authentication struct {
	PublicMCPURL string                             `yaml:"PublicMCPURL"`
	Providers    map[string]OAuthProvider           `yaml:"Providers"`
	Components   map[string]ComponentAuthentication `yaml:"Components"`
	DefaultMode  string                             `yaml:"DefaultMode"`
	CertURL      string                             `yaml:"CertURL"`
	Issuer       string                             `yaml:"Issuer"`
	Audience     string                             `yaml:"Audience"`
}

type OAuthProvider struct {
	CertURL  string `yaml:"CertURL"`
	Issuer   string `yaml:"Issuer"`
	Audience string `yaml:"Audience"`
}

// ComponentAuthentication is a deployment-owned policy keyed by Studio report ID.
// Provider selects runtime credentials independently of Studio's author identity.
type ComponentAuthentication struct {
	Provider string   `yaml:"Provider"`
	Public   bool     `yaml:"Public"`
	Scopes   []string `yaml:"Scopes"`
}
type Studio struct {
	Driver string `yaml:"Driver"`
	DSN    string `yaml:"DSN"`
}
type Admin struct {
	Token string `yaml:"Token"`
}
type Config struct {
	// ModulePath supplies virtual dynamic package authority in a source-free deployment.
	ModulePath string `yaml:"ModulePath,omitempty"`
	// LinkedComponents are immutable application builds supplied by the embedding host.
	LinkedComponents []LinkedComponentSource `yaml:"-"`
	// ForgeProvider is an opt-in host extension. Studio does not own Forge
	// definitions or report schemas; the embedding process supplies them.
	ForgeProvider *forgemcp.PortableProvider `yaml:"-"`
	// Forge configures the generic stock provider for operator-owned portable
	// window definitions. Reporting remains outside this host contract.
	Forge       *ForgeConfig          `yaml:"Forge,omitempty"`
	NamespaceID string                `yaml:"NamespaceID,omitempty"`
	Access      *ResourceAccessConfig `yaml:"Access"`
	// DecisionProvider is supplied by the embedding process, never by YAML.
	DecisionProvider  access.DecisionProvider    `yaml:"-"`
	PredicatePackages []predicatecatalog.Package `yaml:"-"`
	HTTP              Listener                   `yaml:"HTTP"`
	MCP               Listener                   `yaml:"MCP"`
	Authentication    Authentication             `yaml:"Authentication"`
	Studio            Studio                     `yaml:"Studio"`
	Admin             Admin                      `yaml:"Admin"`
	RootDir           string                     `yaml:"RootDir"`
}

type ResourceAccessConfig struct {
	// Provider is a trusted embedding-host identity binding, never YAML input.
	// When supplied, it owns verification and overrides file verifier settings.
	Provider         access.Provider            `yaml:"-"`
	Tenant           string                     `yaml:"Tenant"`
	Issuer           string                     `yaml:"Issuer"`
	Audience         string                     `yaml:"Audience"`
	PublicKeyFile    string                     `yaml:"PublicKeyFile"`
	CertURL          string                     `yaml:"CertURL,omitempty"`
	UserInfoURL      string                     `yaml:"UserInfoURL,omitempty"`
	ResourceBindings map[string]access.Resource `yaml:"ResourceBindings"`
}

func Load(path string) (*Config, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := &Config{}
	if err = yaml.Unmarshal(payload, result); err != nil {
		return nil, err
	}
	if err = result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Config) Validate() error {
	if c.NamespaceID != "" {
		decoded, err := hex.DecodeString(c.NamespaceID)
		if err != nil || len(decoded) != 32 || c.NamespaceID != strings.ToLower(c.NamespaceID) {
			return fmt.Errorf("NamespaceID must be a canonical namespace identity")
		}
	}

	if c == nil {
		return fmt.Errorf("dynamic Datly host config is required")
	}
	if (strings.TrimSpace(c.Authentication.Issuer) == "") != (strings.TrimSpace(c.Authentication.Audience) == "") {
		return fmt.Errorf("default runtime identity requires both Issuer and Audience")
	}
	for name, provider := range c.Authentication.Providers {
		if strings.ContainsAny(name, "/\\?#%\" \t\r\n") {
			return fmt.Errorf("invalid provider name %q", name)
		}
		if strings.TrimSpace(name) == "" || provider.CertURL == "" || provider.Issuer == "" || provider.Audience == "" {
			return fmt.Errorf("runtime provider %q requires CertURL, Issuer and Audience", name)
		}
	}
	if base := c.Authentication.PublicMCPURL; base != "" {
		u, err := url.Parse(base)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
			return fmt.Errorf("PublicMCPURL must be an absolute origin")
		}
		if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback())) {
			return fmt.Errorf("PublicMCPURL requires HTTPS or loopback HTTP")
		}
	}
	for id, policy := range c.Authentication.Components {
		if id == "" {
			return fmt.Errorf("component authentication requires report ID")
		}
		if policy.Public {
			if policy.Provider != "" || len(policy.Scopes) > 0 {
				return fmt.Errorf("public component %s cannot require a provider or scopes", id)
			}
			continue
		}
		if _, ok := c.Authentication.Providers[policy.Provider]; !ok {
			return fmt.Errorf("component %s has unknown runtime provider %q", id, policy.Provider)
		}
	}
	if err := validAddress("HTTP", c.HTTP.Address); err != nil {
		return err
	}
	if err := validAddress("MCP", c.MCP.Address); err != nil {
		return err
	}
	if c.HTTP.Address == c.MCP.Address && !strings.HasSuffix(c.HTTP.Address, ":0") {
		return fmt.Errorf("dynamic HTTP and MCP listeners must use dedicated addresses")
	}
	if strings.TrimSpace(c.Studio.Driver) == "" || strings.TrimSpace(c.Studio.DSN) == "" {
		return fmt.Errorf("dynamic Studio database configuration is required")
	}
	if strings.TrimSpace(c.RootDir) == "" {
		c.RootDir = "."
	}
	if strings.TrimSpace(c.Admin.Token) == "" {
		return fmt.Errorf("dynamic runtime administration token is required")
	}
	host, _, _ := net.SplitHostPort(strings.TrimSpace(c.HTTP.Address))
	if ip := net.ParseIP(host); (ip == nil || !ip.IsLoopback()) && c.Admin.Token == LocalDevelopmentAdminToken {
		return fmt.Errorf("default dynamic runtime administration token is restricted to loopback listeners")
	}
	switch strings.ToLower(strings.TrimSpace(c.Authentication.DefaultMode)) {
	case "required", "trusted_backend", "public":
	default:
		return fmt.Errorf("unsupported dynamic authentication mode %q", c.Authentication.DefaultMode)
	}
	if c.Access != nil {
		if c.Access.Tenant == "" || c.Access.Provider == nil && (c.Access.Issuer == "" || c.Access.Audience == "" || (c.Access.PublicKeyFile == "") == (c.Access.CertURL == "")) {
			return fmt.Errorf("resource access requires Tenant, Issuer, Audience and one public key file or CertURL")
		}
		for prefix, r := range c.Access.ResourceBindings {
			if prefix == "" || !strings.HasSuffix(prefix, "/") || r.Kind == "" || r.ID == "" || r.Version == "" || r.Tenant != c.Access.Tenant {
				return fmt.Errorf("invalid resource access binding %q", prefix)
			}
		}
	}
	if c.Access == nil && !strings.EqualFold(c.Authentication.DefaultMode, "public") && strings.TrimSpace(c.Authentication.CertURL) == "" {
		return fmt.Errorf("dynamic authenticated mode needs CertURL")
	}
	if err := validateForgeConfig(c); err != nil {
		return err
	}
	return nil
}

func validAddress(name, address string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil || host == "" || port == "" {
		return fmt.Errorf("dynamic %s listener address is invalid", name)
	}
	return nil
}
