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

func TestVisibilityReusesVerifiedFactsOnlyWithinInvocation(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	provider := &countedRoleProvider{facts: authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", ValidUntil: time.Now().Add(time.Minute)}}
	base := accessprovider.WithProvider(context.Background(), provider)
	scope := oauth.WithBearer(xexec.WithContext(base, xexec.New()), "verified-caller")
	if _, err := Criteria(scope, nil, "alice", "n"); err != nil {
		t.Fatal(err)
	}
	if _, err := verifiedRoles(scope, "alice"); err != nil {
		t.Fatal(err)
	}
	if provider.reads != 1 {
		t.Fatalf("one visibility decision fetched identity %d times", provider.reads)
	}
	// A later source change cannot elevate this invocation's captured identity.
	provider.facts.Roles = []string{"NAMESPACE_ADMIN"}
	if IsAdministrator(scope, "alice") {
		t.Fatal("negative invocation snapshot gained a new administrator role")
	}
	if provider.reads != 1 {
		t.Fatal("invocation snapshot was re-fetched")
	}
	next := oauth.WithBearer(xexec.WithContext(base, xexec.New()), "verified-caller")
	roles, err := verifiedRoles(next, "alice")
	if err != nil || len(roles) != 1 || provider.reads != 2 {
		t.Fatalf("next invocation did not refresh facts: %v %v reads=%d", roles, err, provider.reads)
	}
	provider.fail = true
	if _, err = verifiedRoles(oauth.WithBearer(scope, "different-caller"), "alice"); err == nil {
		t.Fatal("role facts reused for a different credential")
	}
	if _, err = verifiedRoles(scope, "bob"); err == nil {
		t.Fatal("role facts reused for a different subject")
	}
}

type countedRoleProvider struct {
	facts authz.Facts
	reads int
	fail  bool
}

func (p *countedRoleProvider) Resolve(context.Context) (authz.Facts, error) {
	p.reads++
	if p.fail {
		return authz.Facts{}, errors.New("unavailable")
	}
	return p.facts, nil
}

func TestNamespaceAdministratorAcceptsEitherConfiguredRole(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "AUTHORING_ADMIN,DATA_ADMIN")
	for _, role := range []string{"AUTHORING_ADMIN", "DATA_ADMIN", "manager", "*", "data_admin"} {
		t.Run(role, func(t *testing.T) {
			provider := &adminTestProvider{facts: authz.Facts{Subject: "operator", Issuer: "idp", Tenant: "one", Roles: []string{role}, ValidUntil: time.Now().Add(time.Minute)}}
			ctx := accessprovider.WithProvider(oauth.WithBearer(xexec.WithContext(context.Background(), xexec.New()), "credential"), provider)
			want := role == "AUTHORING_ADMIN" || role == "DATA_ADMIN"
			if IsAdministrator(ctx, "operator") != want || HasAdministratorRole(provider.facts) != want {
				t.Fatal("configured administrator alternatives differ")
			}
		})
	}
}
