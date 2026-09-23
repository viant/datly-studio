package host

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk/access"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

func boolPtr(v bool) *bool { return &v }

func scopedComponent(parameters ...*spec.Parameter) *spec.Component {
	return &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/app", Name: "tasks"}, Parameters: parameters}
}

func scopeParam(name, entity string, required bool) *spec.Parameter {
	return &spec.Parameter{Name: name, TypeExpr: "[]int", Required: boolPtr(required), Source: spec.BindSource{Kind: "scope", Name: entity}}
}

func TestResolveScopeBindingRejectsEveryMismatch(t *testing.T) {
	intInput := reflect.TypeOf(struct{ ProjectIDs []int }{})
	declaration := ScopeBinding{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}
	cases := []struct {
		name         string
		declarations []ScopeBinding
		access       bool
		component    *spec.Component
		input        reflect.Type
		wantNil      bool
		wantErr      string
		wantType     reflect.Type
	}{
		{name: "no scope parameter and no declaration", access: true, component: scopedComponent(&spec.Parameter{Name: "Limit", Source: spec.BindSource{Kind: "query", Name: "limit"}}), input: intInput, wantNil: true},
		{name: "declaration without scope parameter", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(), input: intInput, wantErr: "has no scope parameter"},
		{name: "scope parameter without declaration", access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantErr: "without a deployment scope binding"},
		{name: "scope parameter in legacy mode", declarations: []ScopeBinding{declaration}, access: false, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantErr: "requires generic resource access"},
		{name: "two scope parameters", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true), scopeParam("AccountIDs", "account", true)), input: intInput, wantErr: "one entity dimension"},
		{name: "two declarations", declarations: []ScopeBinding{declaration, declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantErr: "has 2 scope bindings"},
		{name: "parameter name mismatch", declarations: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "Other"}}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantErr: "names parameter"},
		{name: "entity type mismatch", declarations: []ScopeBinding{{Component: "tasks", Version: "1", EntityType: "account", Parameter: "ProjectIDs"}}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantErr: "does not match parameter scope/project"},
		{name: "optional scope parameter", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", false)), input: intInput, wantErr: "must be declared Required()"},
		{name: "missing compiled input", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: nil, wantErr: "no compiled input type"},
		{name: "field absent from compiled input", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ Other []int }{}), wantErr: "has no field"},
		{name: "scalar field", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ ProjectIDs int }{}), wantErr: "must be a slice"},
		{name: "float slice", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ ProjectIDs []float64 }{}), wantErr: "must be a slice of string or integer"},
		{name: "pointer to slice", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ ProjectIDs *[]int }{}), wantErr: "must be a slice"},
		{name: "int slice resolves", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: intInput, wantType: reflect.TypeOf([]int{})},
		{name: "string slice resolves", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ ProjectIDs []string }{}), wantType: reflect.TypeOf([]string{})},
		{name: "uint64 slice resolves", declarations: []ScopeBinding{declaration}, access: true, component: scopedComponent(scopeParam("ProjectIDs", "project", true)), input: reflect.TypeOf(struct{ ProjectIDs []uint64 }{}), wantType: reflect.TypeOf([]uint64{})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := resolveScopeBinding(tc.declarations, tc.access, "tasks", "1", tc.component, tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v (resolved=%+v)", tc.wantErr, err, resolved)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantNil {
				if resolved != nil {
					t.Fatalf("expected no binding, got %+v", resolved)
				}
				return
			}
			if resolved == nil || resolved.sliceType != tc.wantType || resolved.declaration != tc.declarations[0] {
				t.Fatalf("resolved=%+v", resolved)
			}
		})
	}
}

func TestResolveScopeBindingsMatchesActiveVersionOnly(t *testing.T) {
	// Declarations for other versions or reports are ignored; the active version
	// must still be covered.
	component := scopedComponent(scopeParam("ProjectIDs", "project", true))
	registered := []compiledComponent{{key: component.Key, component: component, inputType: reflect.TypeOf(struct{ ProjectIDs []int }{})}}
	reports := map[spec.Key]string{component.Key: "tasks"}
	versions := map[string]int{"tasks": 2}
	if _, err := resolveScopeBindings([]ScopeBinding{{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}}, true, registered, reports, versions); err == nil {
		t.Fatal("stale version declaration satisfied the active version")
	}
	result, err := resolveScopeBindings([]ScopeBinding{
		{Component: "other", Version: "2", EntityType: "project", Parameter: "ProjectIDs"},
		{Component: "tasks", Version: "2", EntityType: "project", Parameter: "ProjectIDs"},
	}, true, registered, reports, versions)
	if err != nil || len(result) != 1 || result["tasks"] == nil || result["tasks"].declaration.Version != "2" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err = resolveScopeBindings(nil, true, registered, map[spec.Key]string{}, versions); err == nil {
		t.Fatal("component without owning report was accepted")
	}
}

func TestScopeValuesConvertOnlyCanonicalIDsOfTheBoundDimension(t *testing.T) {
	bounded := func(entities ...access.Entity) access.Decision {
		return access.Decision{Bounded: true, Entities: entities}
	}
	ints := &resolvedScopeBinding{declaration: ScopeBinding{EntityType: "project", Parameter: "ProjectIDs"}, sliceType: reflect.TypeOf([]int{})}
	strs := &resolvedScopeBinding{declaration: ScopeBinding{EntityType: "project", Parameter: "ProjectIDs"}, sliceType: reflect.TypeOf([]string{})}
	small := &resolvedScopeBinding{declaration: ScopeBinding{EntityType: "project", Parameter: "ProjectIDs"}, sliceType: reflect.TypeOf([]int8{})}
	unsigned := &resolvedScopeBinding{declaration: ScopeBinding{EntityType: "project", Parameter: "ProjectIDs"}, sliceType: reflect.TypeOf([]uint16{})}
	cases := []struct {
		name     string
		binding  *resolvedScopeBinding
		decision access.Decision
		want     any
		wantErr  bool
	}{
		{name: "ints", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "101"}, access.Entity{Type: "project", ID: "102"}), want: []int{101, 102}},
		{name: "strings keep text", binding: strs, decision: bounded(access.Entity{Type: "project", ID: "p-1"}, access.Entity{Type: "project", ID: "007"}), want: []string{"p-1", "007"}},
		{name: "uint16", binding: unsigned, decision: bounded(access.Entity{Type: "project", ID: "65535"}), want: []uint16{65535}},
		{name: "negative int", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "-1"}), want: []int{-1}},
		{name: "unbounded", binding: ints, decision: access.Decision{}, wantErr: true},
		{name: "empty bounded", binding: ints, decision: access.Decision{Bounded: true}, wantErr: true},
		{name: "foreign dimension", binding: ints, decision: bounded(access.Entity{Type: "account", ID: "101"}), wantErr: true},
		{name: "mixed dimensions", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "101"}, access.Entity{Type: "account", ID: "102"}), wantErr: true},
		{name: "duplicate id", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "101"}, access.Entity{Type: "project", ID: "101"}), wantErr: true},
		{name: "empty id", binding: ints, decision: bounded(access.Entity{Type: "project", ID: ""}), wantErr: true},
		{name: "non numeric", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "abc"}), wantErr: true},
		{name: "leading zero", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "007"}), wantErr: true},
		{name: "leading plus", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "+7"}), wantErr: true},
		{name: "whitespace", binding: ints, decision: bounded(access.Entity{Type: "project", ID: " 7"}), wantErr: true},
		{name: "hex", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "0x10"}), wantErr: true},
		{name: "float", binding: ints, decision: bounded(access.Entity{Type: "project", ID: "7.0"}), wantErr: true},
		{name: "int8 overflow", binding: small, decision: bounded(access.Entity{Type: "project", ID: "128"}), wantErr: true},
		{name: "uint16 overflow", binding: unsigned, decision: bounded(access.Entity{Type: "project", ID: "65536"}), wantErr: true},
		{name: "negative unsigned", binding: unsigned, decision: bounded(access.Entity{Type: "project", ID: "-1"}), wantErr: true},
		{name: "sql fragment as id", binding: strs, decision: bounded(access.Entity{Type: "project", ID: "1' OR '1'='1"}), want: []string{"1' OR '1'='1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.binding.values(tc.decision)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("want %#v got %#v", tc.want, got)
			}
		})
	}
	var nilBinding *resolvedScopeBinding
	if _, err := nilBinding.values(bounded(access.Entity{Type: "project", ID: "1"})); err == nil {
		t.Fatal("nil binding converted values")
	}
}

func TestScopeBindingBindRequiresCapturedInvocationAndServesOnlyBoundDimension(t *testing.T) {
	binding := &resolvedScopeBinding{declaration: ScopeBinding{EntityType: "project", Parameter: "ProjectIDs"}, sliceType: reflect.TypeOf([]int{})}
	decision := access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "101"}, {Type: "project", ID: "102"}}}
	if err := binding.bind(context.Background(), decision); !errors.Is(err, dexec.ErrScopeBindingUnavailable) {
		t.Fatalf("binding without an adapter capture must fail: %v", err)
	}
	ctx, _ := dexec.CaptureScopeBinding(context.Background())
	if err := binding.bind(ctx, access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "x"}}}); err == nil {
		t.Fatal("invalid conversion was bound")
	}
	if dexec.ScopeProviders(ctx) != nil {
		t.Fatal("failed conversion left a provider behind")
	}
	if err := binding.bind(ctx, decision); err != nil {
		t.Fatal(err)
	}
	providers := dexec.ScopeProviders(ctx)
	if len(providers) != 1 || providers[0].Kind() != "scope" {
		t.Fatalf("providers=%+v", providers)
	}
	locate := providers[0].Locate(nil)
	value, ok, err := locate.Value(ctx, reflect.TypeOf([]int{}), "project")
	if err != nil || !ok || !reflect.DeepEqual(value, []int{101, 102}) {
		t.Fatalf("value=%#v ok=%v err=%v", value, ok, err)
	}
	if _, _, err = locate.Value(ctx, reflect.TypeOf([]int{}), "account"); err == nil {
		t.Fatal("unbound dimension was served")
	}
	if _, _, err = locate.Value(ctx, reflect.TypeOf([]string{}), "project"); err == nil {
		t.Fatal("incompatible target type was served")
	}
	if _, _, err = locate.Value(ctx, reflect.TypeOf(0), "project"); err == nil {
		t.Fatal("scalar target type was served")
	}
}

func TestConfigValidatesScopeBindings(t *testing.T) {
	base := func() Config {
		return Config{
			HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:1"},
			Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: "file:test.db"},
			Admin: Admin{Token: "deployment-owned"}, RootDir: ".",
			Access: &ResourceAccessConfig{Tenant: "one", Issuer: "https://issuer", Audience: "runtime", PublicKeyFile: "key.pem"},
		}
	}
	valid := ScopeBinding{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}
	config := base()
	config.Access.ScopeBindings = []ScopeBinding{valid}
	if err := config.Validate(); err != nil {
		t.Fatalf("valid binding rejected: %v", err)
	}
	for name, bindings := range map[string][]ScopeBinding{
		"duplicate component version": {valid, {Component: "tasks", Version: "1", EntityType: "account", Parameter: "AccountIDs"}},
		"missing component":           {{Version: "1", EntityType: "project", Parameter: "ProjectIDs"}},
		"missing version":             {{Component: "tasks", EntityType: "project", Parameter: "ProjectIDs"}},
		"non numeric version":         {{Component: "tasks", Version: "v1", EntityType: "project", Parameter: "ProjectIDs"}},
		"missing entity type":         {{Component: "tasks", Version: "1", Parameter: "ProjectIDs"}},
		"missing parameter":           {{Component: "tasks", Version: "1", EntityType: "project"}},
		"untrimmed component":         {{Component: " tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs"}},
		"parameter with dot":          {{Component: "tasks", Version: "1", EntityType: "project", Parameter: "Project.IDs"}},
		"parameter with sql":          {{Component: "tasks", Version: "1", EntityType: "project", Parameter: "ProjectIDs) OR 1=1 --"}},
		"entity type with slash":      {{Component: "tasks", Version: "1", EntityType: "project/all", Parameter: "ProjectIDs"}},
		"entity type with space":      {{Component: "tasks", Version: "1", EntityType: "pro ject", Parameter: "ProjectIDs"}},
	} {
		t.Run(name, func(t *testing.T) {
			config := base()
			config.Access.ScopeBindings = bindings
			if err := config.Validate(); err == nil {
				t.Fatalf("accepted %+v", bindings)
			}
		})
	}
	// Two versions of the same component may each carry a declaration.
	config = base()
	config.Access.ScopeBindings = []ScopeBinding{valid, {Component: "tasks", Version: "2", EntityType: "project", Parameter: "ProjectIDs"}}
	if err := config.Validate(); err != nil {
		t.Fatalf("per-version bindings rejected: %v", err)
	}
}
