package accessconfig

import (
	"context"
	"errors"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/scy/auth/jwt/verifier"
)

// configuredKeyfunc preserves certificate refresh and SCY encrypted RSA resource
// loading while keeping token claim checks in the existing authz provider.
func configuredKeyfunc(config *verifier.Config, publicKeyFile, certURL string) (jwtlib.Keyfunc, error) {
	if config == nil {
		return keyfunc(publicKeyFile, certURL)
	}
	if publicKeyFile != "" || certURL != "" {
		return nil, errors.New("shared JWTValidator conflicts with legacy ACL key settings")
	}
	if (len(config.RSA) == 0) == (strings.TrimSpace(config.CertURL) == "") || config.HMAC != nil || len(config.Rules) != 0 {
		return nil, errors.New("ACL JWTValidator requires RSA resources or CertURL")
	}
	if config.CertURL != "" {
		return keyfunc("", config.CertURL)
	}
	for _, resource := range config.RSA {
		if resource == nil || strings.TrimSpace(resource.URL) == "" {
			return nil, errors.New("ACL JWTValidator RSA resource is required")
		}
	}
	service := verifier.New(config)
	if err := service.Init(context.Background()); err != nil {
		return nil, errors.New("ACL JWTValidator RSA resources could not be initialized")
	}
	keys, err := service.PublicKeys()
	if err != nil || len(keys) == 0 {
		return nil, errors.New("ACL JWTValidator RSA keys are unavailable")
	}
	set := jwtlib.VerificationKeySet{}
	for _, key := range keys {
		set.Keys = append(set.Keys, key)
	}
	return func(*jwtlib.Token) (any, error) { return set, nil }, nil
}

// VerificationKey exposes the configured verifier to the public process binding.
func VerificationKey(config *verifier.Config, publicKeyFile, certURL string) (jwtlib.Keyfunc, error) {
	return configuredKeyfunc(config, publicKeyFile, certURL)
}
