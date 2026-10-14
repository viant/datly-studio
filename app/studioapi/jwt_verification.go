package studioapi

import (
	"errors"
	"strings"

	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

// Both Studio and embedding applications use this existing SCY verification
// configuration. Issuer/audience and session authorization remain unchanged.
func jwtVerificationConfig(certURL, publicKey string) (*verifier.Config, error) {
	certURL, publicKey = strings.TrimSpace(certURL), strings.TrimSpace(publicKey)
	if (certURL == "") == (publicKey == "") {
		return nil, errors.New("configure exactly one of -jwt-public-key or -jwt-cert-url in authenticated mode")
	}
	if certURL != "" {
		return &verifier.Config{CertURL: certURL}, nil
	}
	location, key, _ := strings.Cut(publicKey, "|")
	if strings.TrimSpace(location) == "" {
		return nil, errors.New("JWT RSA public-key resource URL is required")
	}
	return &verifier.Config{RSA: []*scy.Resource{scy.NewResource(nil, location, key)}}, nil
}
