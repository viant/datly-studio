// Package accessconfig resolves server-owned ACL provider configuration.
package accessconfig

import (
	"fmt"
	"os"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	access "github.com/viant/authz"
	accessoauth "github.com/viant/authz/oauth"
)

// UserInfoProviderFactory binds a Studio user-info URL to a host-owned
// identity provider. Studio deliberately does not define the remote response
// schema or interpret its claims.
type UserInfoProviderFactory func(issuer, audience, userInfoURL string, keyfunc jwtlib.Keyfunc) (access.Provider, error)

// FromEnvironment returns nil only when ACL verification is entirely absent.
// Partial or invalid configuration fails rather than falling back to public mode.
func FromEnvironment() (access.Provider, error) {
	return New(Config{Issuer: os.Getenv("STUDIO_ACCESS_ISSUER"), Audience: os.Getenv("STUDIO_ACCESS_AUDIENCE"),
		PublicKeyFile: os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), CertURL: os.Getenv("STUDIO_ACCESS_CERT_URL"), UserInfoURL: os.Getenv("STUDIO_ACCESS_USER_INFO_URL")})
}

// FromEnvironmentWithUserInfo resolves environment configuration with an
// explicitly supplied host-owned user-info provider factory.
func FromEnvironmentWithUserInfo(factory UserInfoProviderFactory) (access.Provider, error) {
	return NewWithUserInfo(Config{Issuer: os.Getenv("STUDIO_ACCESS_ISSUER"), Audience: os.Getenv("STUDIO_ACCESS_AUDIENCE"),
		PublicKeyFile: os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), CertURL: os.Getenv("STUDIO_ACCESS_CERT_URL"), UserInfoURL: os.Getenv("STUDIO_ACCESS_USER_INFO_URL")}, factory)
}

// IdentityFromEnvironment verifies the same JWT without depending on the
// optional user-info authority service. Callers may use it only for access
// based on the verified subject, never to infer roles or entity grants.
func IdentityFromEnvironment() (access.Provider, error) {
	config := Config{Issuer: os.Getenv("STUDIO_ACCESS_ISSUER"), Audience: os.Getenv("STUDIO_ACCESS_AUDIENCE"),
		PublicKeyFile: os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), CertURL: os.Getenv("STUDIO_ACCESS_CERT_URL"),
		UserInfoURL: os.Getenv("STUDIO_ACCESS_USER_INFO_URL")}
	if strings.TrimSpace(config.Issuer) == "" && strings.TrimSpace(config.Audience) == "" && strings.TrimSpace(config.PublicKeyFile) == "" && strings.TrimSpace(config.CertURL) == "" && strings.TrimSpace(config.UserInfoURL) == "" {
		return nil, nil
	}
	key, err := keyfunc(strings.TrimSpace(config.PublicKeyFile), strings.TrimSpace(config.CertURL))
	if err != nil {
		return nil, err
	}
	return accessoauth.NewIdentity(accessoauth.Config{Issuer: strings.TrimSpace(config.Issuer), Audience: strings.TrimSpace(config.Audience),
		Algorithms: []string{"RS256"}, Keyfunc: key})
}

type Config struct {
	Issuer, Audience, PublicKeyFile, CertURL, UserInfoURL string
}

// New resolves one server-owned verifier for static, dynamic, and SDK paths.
func New(config Config) (access.Provider, error) {
	return newProvider(config, nil)
}

// NewWithUserInfo resolves configuration using an explicitly injected
// user-info provider factory when UserInfoURL is configured.
func NewWithUserInfo(config Config, factory UserInfoProviderFactory) (access.Provider, error) {
	return newProvider(config, factory)
}

func newProvider(config Config, factory UserInfoProviderFactory) (access.Provider, error) {
	issuer := strings.TrimSpace(config.Issuer)
	audience := strings.TrimSpace(config.Audience)
	keyPath := strings.TrimSpace(config.PublicKeyFile)
	certURL := strings.TrimSpace(config.CertURL)
	userInfoURL := strings.TrimSpace(config.UserInfoURL)
	if issuer == "" && audience == "" && keyPath == "" && certURL == "" && userInfoURL == "" {
		return nil, nil
	}
	if issuer == "" || audience == "" {
		return nil, fmt.Errorf("ACL issuer and audience are required")
	}
	keyfunc, err := keyfunc(keyPath, certURL)
	if err != nil {
		return nil, err
	}
	if userInfoURL != "" {
		if factory == nil {
			return nil, fmt.Errorf("ACL user-info URL requires an explicitly configured identity provider")
		}
		provider, err := factory(issuer, audience, userInfoURL, keyfunc)
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf("ACL identity provider factory returned nil")
		}
		return provider, nil
	}
	return accessoauth.New(accessoauth.Config{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc})
}
