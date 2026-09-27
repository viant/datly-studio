package host

import (
	"path/filepath"
	"testing"
)

func TestRuntimeConfigUsesDedicatedMCPPortAndRequiredAuth(t *testing.T) {
	config, err := Load(filepath.Join("..", "..", "datly-runtime.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if config.HTTP.Address != "127.0.0.1:8082" || config.MCP.Address != "127.0.0.1:8091" || config.Authentication.DefaultMode != "required" ||
		config.Authentication.Issuer != "https://idp.viantinc.com" || config.Authentication.Audience != "datly-studio-web" {
		t.Fatalf("config=%+v", config)
	}
}

func TestRuntimeDefaultIdentityRequiresIssuerAndAudienceTogether(t *testing.T) {
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "required", CertURL: "https://identity.example/certs", Issuer: "https://identity.example"},
		Studio:         Studio{Driver: "sqlite", DSN: "file:test.db"}, Admin: Admin{Token: "test-token"}, RootDir: "."}
	if err := config.Validate(); err == nil {
		t.Fatal("issuer without audience was accepted")
	}
	config.Authentication.Issuer = ""
	config.Authentication.Audience = "studio-web"
	if err := config.Validate(); err == nil {
		t.Fatal("audience without issuer was accepted")
	}
	config.Authentication.Issuer = "https://identity.example"
	if err := config.Validate(); err != nil {
		t.Fatalf("bound identity rejected: %v", err)
	}
}

func TestRuntimeConfigRejectsDefaultAdminTokenOffLoopback(t *testing.T) {
	config := Config{
		HTTP: Listener{Address: "0.0.0.0:8082"}, MCP: Listener{Address: "0.0.0.0:8091"},
		Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: "file:test.db"},
		Admin: Admin{Token: LocalDevelopmentAdminToken}, RootDir: ".",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("default administration token was accepted on a non-loopback listener")
	}
	config.Admin.Token = "deployment-owned-secret"
	if err := config.Validate(); err != nil {
		t.Fatalf("deployment token rejected: %v", err)
	}
}
