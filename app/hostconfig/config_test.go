package hostconfig

import (
	"strings"
	"testing"
)

const base = `Endpoint:
  Address: 127.0.0.1:8080
MCP:
  Address: 127.0.0.1:8081
NamespaceMCP:
  Address: 127.0.0.1:8082
Store:
  Driver: mysql
  DSN: '${Username}:${Password}@tcp(localhost:3306)/studio'
  Secret:
    URL: file:///config/database.enc
    Key: blowfish://default
`

func TestDatlyConnectorAndVerifierCompatibility(t *testing.T) {
	for _, jwt := range []string{
		"JWTValidator:\n  RSA:\n    - URL: file:///config/public.enc\n      Key: blowfish://default\n",
		"JWTValidator:\n  CertURL: https://auth.example.com/certs\n",
	} {
		c, err := Decode([]byte(base + jwt))
		if err != nil {
			t.Fatal(err)
		}
		if c.Store == nil || c.Store.Secret.Key != "blowfish://default" {
			t.Fatal("Datly connector contract lost")
		}
		if !strings.Contains(c.Store.DSN, "${Password}") {
			t.Fatal("secret expanded during decode")
		}
		if c.JWTValidator == nil || len(c.JWTValidator.RSA) == 0 && c.JWTValidator.CertURL == "" {
			t.Fatal("verifier configuration lost")
		}
	}
}

func TestRejectAmbiguousOrNonInfrastructureConfiguration(t *testing.T) {
	for _, input := range []string{
		base + "Namespaces: [ar]\n", base + "Discovery: published-reports\n",
		base + "Connector: duplicate\n", base + "Connectors: []\n", base + "Store: {}\n", base + "---\nConnector: other\n",
		strings.Replace(base, "127.0.0.1:8081", "bad-address", 1),
		"null", "[]", `{"MCP":null,"MCP":{}}`,
	} {
		if _, err := Decode([]byte(input)); err == nil {
			t.Fatalf("accepted invalid configuration: %q", input)
		}
	}
}

func TestNamespaceConnectionConfigPreservesUnreadableSCYReference(t *testing.T) {
	raw := base + `NamespaceConnectionExport:
  Provider: employees
  ClientConfigURL: file:///does-not-exist/namespace-client.enc|blowfish://env/UNSET_NAMESPACE_KEY
  Scopes: [openid]
NamespaceOAuthProviders:
  employees:
    Issuer: https://idp.example
    CertURL: https://idp.example/certs
    Audience: namespace-client
`
	config, err := Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if config.NamespaceConnectionExport == nil || config.NamespaceConnectionExport.ClientConfigURL != "file:///does-not-exist/namespace-client.enc|blowfish://env/UNSET_NAMESPACE_KEY" || config.NamespaceConnectionExport.Provider != "employees" || len(config.NamespaceConnectionExport.Scopes) != 1 {
		t.Fatal("namespace SCY reference was resolved or changed")
	}
	if config.NamespaceOAuthProviders["employees"].Issuer != "https://idp.example" {
		t.Fatal("namespace issuer configuration lost")
	}
}
