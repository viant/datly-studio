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
	if config.HTTP.Address != "127.0.0.1:8082" || config.MCP.Address != "127.0.0.1:8091" || config.Authentication.DefaultMode != "required" {
		t.Fatalf("config=%+v", config)
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
