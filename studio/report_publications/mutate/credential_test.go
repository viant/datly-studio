package mutate

import (
	"context"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/scy/auth/jwt"
	"testing"
)

type publicationIdentity struct{}

func (publicationIdentity) Resolve(ctx context.Context) (authz.Facts, error) {
	if oauth.Bearer(ctx) != "verified-credential" {
		return authz.Facts{}, errors.New("namespace identity cannot resolve without bearer")
	}
	return authz.Facts{Subject: "owner"}, nil
}
func TestDetachedPublicationRetainsNamespaceIdentityCredential(t *testing.T) {
	base := accessprovider.WithProvider(context.Background(), publicationIdentity{})
	claims := new(jwt.Claims)
	claims.Subject = "owner"
	ctx, err := publicationCredentialContext(base, claims, "Bearer verified-credential")
	if err != nil {
		t.Fatal(err)
	}
	provider, ok := accessprovider.FromContext(ctx)
	if !ok {
		t.Fatal("identity provider lost")
	}
	facts, err := provider.Resolve(ctx)
	if err != nil || facts.Subject != "owner" {
		t.Fatalf("namespace identity: %+v %v", facts, err)
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || principal.Subject != facts.Subject {
		t.Fatal("verified principal lost")
	}
	if _, err = publicationCredentialContext(base, claims, "invalid"); err == nil {
		t.Fatal("malformed credential accepted")
	}
}
