package host

import (
	"context"
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	nativeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
	xcodec "github.com/viant/xdatly/codec"
	"strings"
)

func runtimeCodecs(ctx context.Context, config Config) (xcodec.Factory, error) {
	if config.CodecFactory != nil {
		return config.CodecFactory, nil
	}
	validator := &verifier.Config{CertURL: config.Authentication.CertURL}
	issuer, audience := config.Authentication.Issuer, config.Authentication.Audience
	if config.Access != nil {
		issuer, audience = config.Access.Issuer, config.Access.Audience
		validator.CertURL = config.Access.CertURL
		if config.Access.PublicKeyFile != "" {
			validator.RSA = []*scy.Resource{{URL: config.Access.PublicKeyFile}}
		}
	}
	if strings.TrimSpace(validator.CertURL) == "" && len(validator.RSA) == 0 {
		return studiocodecs.New(nil), nil
	}
	jwtCodec, err := nativeauth.New(ctx, &nativeauth.Config{JWTValidator: validator, ClaimPolicy: &nativeauth.ClaimPolicy{Issuer: issuer, Audience: audience, RequireSubject: true}})
	if err != nil {
		return nil, err
	}
	return studiocodecs.New(jwtCodec), nil
}
