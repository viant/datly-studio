package hostconfig

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/viant/scy"
	"golang.org/x/oauth2"
)

// Auth configures UI login through the existing opaque-session BFF. It does not
// configure downstream MCP providers or weaken the host JWT/resource policies.
type Auth struct {
	Enabled    bool
	CookieName string
	OAuth      *OAuth
}
type OAuth struct {
	Client          *OAuthClient
	Label           string
	Mode            string
	RedirectSameTab *bool
}
type OAuthClient struct {
	ConfigURL      string
	Scopes         []string
	RedirectURIs   []string
	WebUIScopes    []string
	MobileUIScopes []string
}

func validateAuthDocument(document map[string]any) error {
	for key, value := range document {
		if strings.EqualFold(key, "Auth") {
			if key != "Auth" {
				return errors.New("use uppercase Auth property names")
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return errors.New("invalid Auth object")
			}
			return validateAuthObject(raw, map[string][]string{"": {"Enabled", "CookieName", "OAuth"}, "OAuth": {"Client", "Label", "Mode", "RedirectSameTab"}, "Client": {"ConfigURL", "Scopes", "RedirectURIs", "WebUIScopes", "MobileUIScopes"}}, "")
		}
	}
	return nil
}
func validateAuthObject(raw []byte, fields map[string][]string, kind string) error {
	if string(raw) == "null" {
		return errors.New("Auth configuration objects cannot be null")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return errors.New("Auth configuration requires objects")
	}
	for key, value := range object {
		if !slices.Contains(fields[kind], key) {
			return errors.New("unsupported Auth property: " + key)
		}
		if key == "OAuth" || key == "Client" {
			if err := validateAuthObject(value, fields, key); err != nil {
				return err
			}
		}
	}
	return nil
}
func (c *Config) ValidateAuth() error {
	if c == nil {
		return errors.New("host configuration is required")
	}
	if c.Auth == nil {
		return nil
	}
	if c.Login != nil {
		return errors.New("Auth and legacy Login cannot be combined")
	}
	a := c.Auth
	if a.CookieName != "" {
		if err := (&http.Cookie{Name: a.CookieName, Value: "validation"}).Valid(); err != nil {
			return errors.New("Auth.CookieName is invalid")
		}
	}
	if !a.Enabled {
		return nil
	}
	if a.CookieName == "" {
		return errors.New("enabled Auth requires an application-specific CookieName")
	}
	if a.OAuth == nil || a.OAuth.Client == nil || a.OAuth.Mode != "bff" {
		return errors.New("enabled Auth requires OAuth.Client and Mode: bff")
	}
	if a.OAuth.RedirectSameTab != nil && !*a.OAuth.RedirectSameTab {
		return errors.New("BFF Auth supports RedirectSameTab: true only")
	}
	client := a.OAuth.Client
	if len(client.MobileUIScopes) > 0 {
		return errors.New("MobileUIScopes is not supported by the browser BFF")
	}
	resource := scy.EncodedResource(client.ConfigURL).Decode(context.Background(), nil)
	if resource.URL == "" || resource.Key == "" || strings.HasPrefix(resource.URL, "inlined:") || strings.HasPrefix(resource.URL, "data:") {
		return errors.New("Auth OAuth Client.ConfigURL requires a SCY resource and key reference")
	}
	if !slices.Contains(client.browserScopes(), "openid") {
		return errors.New("Auth OAuth Client.Scopes requires openid")
	}
	for _, scope := range client.browserScopes() {
		if scope == "" || strings.ContainsAny(scope, " \t\r\n") {
			return errors.New("Auth OAuth scopes must be nonempty individual scope names")
		}
	}
	for _, redirect := range client.RedirectURIs {
		parsed, err := url.Parse(redirect)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "/v1/studio/auth/callback" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return errors.New("Auth RedirectURIs must use the Studio callback path")
		}
	}
	return nil
}
func (c *OAuthClient) browserScopes() []string {
	result := []string{}
	for _, scopes := range [][]string{c.Scopes, c.WebUIScopes} {
		for _, scope := range scopes {
			if !slices.Contains(result, scope) {
				result = append(result, scope)
			}
		}
	}
	return result
}
func (a *Auth) Resolve(ctx context.Context, origin string) (*oauth2.Config, error) {
	if a == nil || !a.Enabled {
		return nil, nil
	}
	callback := origin + "/v1/studio/auth/callback"
	if len(a.OAuth.Client.RedirectURIs) > 0 && !slices.Contains(a.OAuth.Client.RedirectURIs, callback) {
		return nil, errors.New("Auth RedirectURIs does not include this application's callback")
	}
	resource := scy.EncodedResource(a.OAuth.Client.ConfigURL).Decode(ctx, nil)
	login := &Login{ClientConfig: resource, Scopes: a.OAuth.Client.browserScopes()}
	return login.Resolve(ctx)
}
