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

// FromEnvironment returns nil only when ACL verification is entirely absent.
// Partial or invalid configuration fails rather than falling back to public mode.
func FromEnvironment() (access.Provider, error) {
	issuer := strings.TrimSpace(os.Getenv("STUDIO_ACCESS_ISSUER"))
	audience := strings.TrimSpace(os.Getenv("STUDIO_ACCESS_AUDIENCE"))
	keyPath := strings.TrimSpace(os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE"))
	userInfoURL := strings.TrimSpace(os.Getenv("STUDIO_ACCESS_USER_INFO_URL"))
	if issuer == "" && audience == "" && keyPath == "" && userInfoURL == "" {
		return nil, nil
	}
	if issuer == "" || audience == "" || keyPath == "" {
		return nil, fmt.Errorf("ACL issuer, audience and public key are required")
	}
	pem, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ACL public key is unavailable")
	}
	key, err := jwtlib.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, fmt.Errorf("ACL public key is invalid")
	}
	keyfunc := func(*jwtlib.Token) (any, error) { return key, nil }
	if userInfoURL != "" {
		return accessoauth.NewUserInfo(accessoauth.UserInfoConfig{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc, URL: userInfoURL})
	}
	return accessoauth.New(accessoauth.Config{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: keyfunc})
}
