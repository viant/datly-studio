package host

import (
	"fmt"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"net"
	"net/url"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

const LocalDevelopmentAdminToken = "datly-studio-local-runtime"

type Listener struct {
	Address string `yaml:"Address"`
}
type Authentication struct {
	PublicMCPURL string                             `yaml:"PublicMCPURL"`
	Providers    map[string]OAuthProvider           `yaml:"Providers"`
	Components   map[string]ComponentAuthentication `yaml:"Components"`
	DefaultMode  string                             `yaml:"DefaultMode"`
	CertURL      string                             `yaml:"CertURL"`
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
	PredicatePackages []predicatecatalog.Package `yaml:"-"`
	HTTP              Listener                   `yaml:"HTTP"`
	MCP               Listener                   `yaml:"MCP"`
	Authentication    Authentication             `yaml:"Authentication"`
	Studio            Studio                     `yaml:"Studio"`
	Admin             Admin                      `yaml:"Admin"`
	RootDir           string                     `yaml:"RootDir"`
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
	if c == nil {
		return fmt.Errorf("dynamic Datly host config is required")
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
	if !strings.EqualFold(c.Authentication.DefaultMode, "public") && strings.TrimSpace(c.Authentication.CertURL) == "" {
		return fmt.Errorf("dynamic authenticated mode needs CertURL")
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
