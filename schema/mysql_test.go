package schema

import (
	"strings"
	"testing"

	policySchema "github.com/viant/authz/component/schema"
)

func TestMySQLDDLComposesSharedPolicySchema(t *testing.T) {
	base, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"resource_policies", "resource_policy_revisions"} {
		if strings.Contains(string(base), "CREATE TABLE "+table) || strings.Contains(string(base), "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("Studio schema.ddl duplicates shared policy table %s", table)
		}
	}
	shared, err := policySchema.PolicyDDL("mysql")
	if err != nil {
		t.Fatal(err)
	}
	composed, err := MySQLDDL()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(composed), strings.TrimSpace(shared)) {
		t.Fatal("MySQL schema composer did not append the canonical fresh policy tables")
	}
	if strings.Count(composed, "CREATE TABLE resource_policy_namespace_bindings (") != 1 || !strings.Contains(composed, "PRIMARY KEY (tenant_id, resource_kind, resource_id, resource_version, policy_revision)") {
		t.Fatal("Studio namespace association must have one complete composite primary key")
	}
	for _, table := range []string{"resource_policies", "resource_policy_revisions"} {
		if strings.Count(composed, "CREATE TABLE IF NOT EXISTS "+table) != 1 {
			t.Fatalf("composed MySQL schema has an invalid canonical definition count for %s", table)
		}
	}
	if strings.Contains(composed, "authz_schema_versions") || strings.Contains(composed, "resource_policy_heads") {
		t.Fatal("fresh Studio schema includes Authz migration bookkeeping")
	}
	for _, expected := range []string{"tenant_id VARBINARY(512) NOT NULL", "resource_kind VARBINARY(256) NOT NULL", "resource_id VARBINARY(800) NOT NULL", "resource_version VARBINARY(256) NOT NULL", "PRIMARY KEY (tenant_id, resource_kind, resource_id, resource_version)"} {
		if !strings.Contains(composed, expected) {
			t.Fatalf("fresh MySQL schema is missing exact policy key definition %q", expected)
		}
	}
	for _, expected := range []string{
		"package_path            VARCHAR(1000) CHARACTER SET ascii COLLATE ascii_bin NOT NULL",
		"type_name               VARCHAR(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL",
		"UNIQUE KEY uq_authorization_predicate_type (package_path, type_name)",
	} {
		if !strings.Contains(composed, expected) {
			t.Fatalf("composed MySQL schema is missing exact key definition %q", expected)
		}
	}
}
