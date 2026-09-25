package sdk

import (
	"context"
	"errors"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

func TestSystemPrincipalIsExplicit(t *testing.T) {
	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Fatal("missing caller became a system principal")
	}
	system := SystemPrincipal()
	if system.Subject != "system:datly-studio" || system.Development {
		t.Fatalf("system principal=%+v", system)
	}
	bound, ok := PrincipalFromContext(WithPrincipal(context.Background(), system))
	if !ok || bound != system {
		t.Fatalf("explicit system binding=%+v ok=%v", bound, ok)
	}
}

func TestWithSystemIdentitySeparatesTrustedClaimsFromSignedBearer(t *testing.T) {
	userCtx := WithVerifiedCredential(WithPrincipal(context.Background(), Principal{Subject: "user"}),
		VerifiedCredential{Bearer: "Bearer user-token", Claims: &jwtv5.RegisteredClaims{Subject: "user"}})
	fallback, err := WithSystemIdentity(userCtx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if principal, ok := PrincipalFromContext(fallback); !ok || principal != SystemPrincipal() {
		t.Fatalf("fallback principal=%+v ok=%v", principal, ok)
	}
	if _, ok := VerifiedCredentialFromContext(fallback); ok {
		t.Fatal("in-process fallback was exposed as a verified bearer")
	}
	provider := func(context.Context) (VerifiedCredential, error) {
		return VerifiedCredential{Bearer: "Bearer signed-service-token", Claims: &jwtv5.RegisteredClaims{
			Subject: SystemPrincipal().Subject, ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Minute))}}, nil
	}
	bound, err := WithSystemIdentity(context.Background(), provider)
	if err != nil {
		t.Fatal(err)
	}
	if credential, ok := VerifiedCredentialFromContext(bound); !ok || credential.Bearer != "Bearer signed-service-token" {
		t.Fatalf("provider credential=%+v ok=%v", credential, ok)
	}
	wrong := func(context.Context) (VerifiedCredential, error) {
		return VerifiedCredential{Bearer: "Bearer signed-service-token", Claims: &jwtv5.RegisteredClaims{
			Subject: "other", ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Minute))}}, nil
	}
	if _, err := WithSystemIdentity(context.Background(), wrong); err == nil {
		t.Fatal("wrong service JWT subject accepted")
	}
	providerFailure := errors.New("provider unavailable")
	if _, err := WithSystemIdentity(userCtx, func(context.Context) (VerifiedCredential, error) {
		return VerifiedCredential{}, providerFailure
	}); !errors.Is(err, providerFailure) {
		t.Fatalf("configured provider failure must not fall back to in-process identity: %v", err)
	}
}
