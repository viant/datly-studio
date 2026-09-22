package datly_studio_test

import (
	"context"
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
