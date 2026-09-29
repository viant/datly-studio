// Package accessconfig resolves server-owned ACL provider configuration.
package accessconfig

import (
	"context"
	"fmt"
	"os"
	"strings"

	access "github.com/viant/authz"
	accessoauth "github.com/viant/authz/oauth"
)

// FromEnvironment returns nil only when ACL verification is entirely absent.
// Partial or invalid configuration fails rather than falling back to public mode.
func FromEnvironment() (access.Provider, error) {
	return New(Config{Issuer: os.Getenv("STUDIO_ACCESS_ISSUER"), Audience: os.Getenv("STUDIO_ACCESS_AUDIENCE"),
		PublicKeyFile: os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), CertURL: os.Getenv("STUDIO_ACCESS_CERT_URL"), UserInfoURL: os.Getenv("STUDIO_ACCESS_USER_INFO_URL")})
}

// IdentityFromEnvironment verifies the same JWT without depending on the
// optional user-info authority service. Callers may use it only for access
// based on the verified subject, never to infer roles or entity grants.
func IdentityFromEnvironment() (access.Provider, error) {
	config := Config{Issuer: os.Getenv("STUDIO_ACCESS_ISSUER"), Audience: os.Getenv("STUDIO_ACCESS_AUDIENCE"),
		PublicKeyFile: os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"), CertURL: os.Getenv("STUDIO_ACCESS_CERT_URL"),
		UserInfoURL: os.Getenv("STUDIO_ACCESS_USER_INFO_URL")}
	if config.UserInfoURL == "" {
		return New(config)
	}
	key, err := keyfunc(config.PublicKeyFile, config.CertURL)
	if err != nil {
		return nil, err
	}
	provider, err := accessoauth.NewUserInfo(accessoauth.UserInfoConfig{Issuer: config.Issuer, Audience: config.Audience,
		Algorithms: []string{"RS256"}, Keyfunc: key, URL: config.UserInfoURL})
	if err != nil {
		return nil, err
	}
	return identityOnly{provider}, nil
}

type identityOnly struct{ provider *accessoauth.UserInfoProvider }

func (i identityOnly) Resolve(ctx context.Context) (access.Facts, error) {
	return i.provider.ResolveIdentity(ctx)
}

type Config struct {
	Issuer, Audience, PublicKeyFile, CertURL, UserInfoURL string
}

// New resolves one server-owned verifier for static, dynamic, and SDK paths.
func New(config Config) (access.Provider, error) {
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
		return accessoauth.NewUserInfo(accessoauth.UserInfoConfig{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc, URL: userInfoURL})
	}
	return accessoauth.New(accessoauth.Config{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc})
}
