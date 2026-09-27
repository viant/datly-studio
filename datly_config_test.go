package datly_studio_test

import (
	"context"
	"strings"
	"testing"

	"github.com/viant/datly/standalone/config"
)

func TestDatlyConfigDeclaresJWTVerifier(t *testing.T) {
	loaded, err := (config.Loader{}).Load(context.Background(), "datly.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.JWTValidator == nil || loaded.JWTValidator.CertURL != "https://idp.viantinc.com/v1/api/oauth2/certs" {
		t.Fatalf("JWTValidator = %+v", loaded.JWTValidator)
	}
	if loaded.JWTClaims == nil || loaded.JWTClaims.Issuer != "https://idp.viantinc.com" || loaded.JWTClaims.Audience != "datly-studio-web" || !loaded.JWTClaims.RequireSubject {
		t.Fatalf("JWTClaims = %+v", loaded.JWTClaims)
	}
	if loaded.Endpoint.Address != "127.0.0.1:8081" {
		t.Fatalf("Datly endpoint = %+v", loaded.Endpoint)
	}
	if len(loaded.Connectors) != 1 || loaded.Connectors[0].Driver != "sqlite" {
		t.Fatalf("expected one sqlite connector, got %#v", loaded.Connectors)
	}
	if loaded.GoBootstrap == nil || !loaded.GoBootstrap.EagerComponents {
		t.Fatal("expected eager static component bootstrap for imported authorization types")
	}
}

func TestResourcePolicyStoreIsLinkedOnlyForNativeAccess(t *testing.T) {
	loaded, err := (config.Loader{}).Load(context.Background(), "datly.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"reader": false, "writer": false, "access": false}
	for _, pkg := range loaded.GoBootstrap.Packages {
		const prefix = "github.com/viant/datly-studio/studio/resource_policy/"
		if strings.HasPrefix(pkg, prefix) {
			name := strings.TrimPrefix(pkg, prefix)
			if _, ok := want[name]; !ok {
				t.Fatalf("unexpected resource policy package in static host: %s", pkg)
			}
			want[name] = true
		}
	}
	for name, linked := range want {
		if !linked {
			t.Errorf("native access package %s is not linked", name)
		}
	}
}
