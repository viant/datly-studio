package namespacevisibility

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	xexec "github.com/viant/xdatly/exec"
	"strings"
	"testing"
	"time"
)

func TestNamespaceAdministratorUsesExactVerifiedRole(t *testing.T) {
	now := time.Now()
	valid := authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", Roles: []string{"NAMESPACE_ADMIN"}, ValidUntil: now.Add(time.Minute)}
	if !matchesAdministrator("NAMESPACE_ADMIN", "alice", valid, now) {
		t.Fatal("exact verified role denied")
	}
	for _, role := range []string{"", "*", "namespace_admin", "ADMIN", "NAMESPACE_ADMIN "} {
		if matchesAdministrator(role, "alice", valid, now) {
			t.Fatalf("unexpected role match %q", role)
		}
	}
	for _, mutate := range []func(*authz.Facts){func(f *authz.Facts) { f.Subject = "bob" }, func(f *authz.Facts) { f.Issuer = "" }, func(f *authz.Facts) { f.Tenant = "" }, func(f *authz.Facts) { f.ValidUntil = now }, func(f *authz.Facts) { f.Roles = nil }} {
		facts := valid
		mutate(&facts)
		if matchesAdministrator("NAMESPACE_ADMIN", "alice", facts, now) {
			t.Fatal("invalid identity granted administration")
		}
	}
}

// The predicate's derived credential context and the output finalizer must
// agree, without exporting bearer tokens or reusing a decision across requests.
func TestAdministratorDecisionIsInvocationBound(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	provider := &adminTestProvider{facts: authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", Roles: []string{"NAMESPACE_ADMIN"}, ValidUntil: time.Now().Add(time.Minute)}}
	invocation := xexec.New()
	base := accessprovider.WithProvider(xexec.WithContext(context.Background(), invocation), provider)
	if !IsAdministrator(oauth.WithBearer(base, "verified-credential"), "alice") {
		t.Fatal("verified predicate denied")
	}
	provider.fail = true
	if !IsAdministrator(base, "alice") {
		t.Fatal("finalizer lost verified invocation decision")
	}
	if IsAdministrator(base, "bob") {
		t.Fatal("decision reused for different subject")
	}
	if IsAdministrator(oauth.WithBearer(base, "different-credential"), "alice") {
		t.Fatal("decision reused for different credential")
	}
	if IsAdministrator(xexec.WithContext(base, xexec.New()), "alice") {
		t.Fatal("decision leaked to another invocation")
	}
	encoded, err := json.Marshal(invocation)
	if err != nil || strings.Contains(string(encoded), "verified-credential") || strings.Contains(string(encoded), "NAMESPACE_ADMIN") {
		t.Fatal("private decision leaked into execution output")
	}
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "OTHER_ADMIN")
	if IsAdministrator(base, "alice") {
		t.Fatal("decision reused after configuration changed")
	}
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	invocation.SetValue(adminDecisionKey, adminDecision{subject: "alice", role: "NAMESPACE_ADMIN", until: time.Now().Add(-time.Second)})
	if IsAdministrator(base, "alice") {
		t.Fatal("expired decision accepted")
	}
}

type adminTestProvider struct {
	facts authz.Facts
	fail  bool
}

func (p *adminTestProvider) Resolve(context.Context) (authz.Facts, error) {
	if p.fail {
		return authz.Facts{}, errors.New("provider unavailable")
	}
	return p.facts, nil
}
