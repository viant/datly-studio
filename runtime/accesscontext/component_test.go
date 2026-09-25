package accesscontext

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly/spec"
	xresponse "github.com/viant/xdatly/response"
)

func facts() access.Facts {
	return access.Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader", "admin"},
		EntityGroups: access.EntityGroups{"project": {"101", "102"}, "account": {"7"}}, ValidUntil: time.Now().Add(time.Minute)}
}

func TestConvertPublishesNarrowedDecisionNotRawFacts(t *testing.T) {
	// The decision is narrower than the facts: only 101 survived intersection.
	output, err := Convert(facts(), access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "101"}}}, "project")
	if err != nil {
		t.Fatal(err)
	}
	if output.Scope == nil || output.Scope.EntityType != "project" || !reflect.DeepEqual(output.Scope.IDs, []string{"101"}) {
		t.Fatalf("scope=%+v", output.Scope)
	}
	if output.Context.Subject != "alice" || output.Context.Tenant != "one" || !reflect.DeepEqual(output.Context.Roles, []string{"admin", "reader"}) {
		t.Fatalf("context=%+v", output.Context)
	}
	// The canonical context carries only the declared dimension and only the
	// narrowed decision: the facts' 102 and the account dimension are absent,
	// so consuming AllowedEntities cannot bypass local/remote narrowing.
	if !reflect.DeepEqual(output.Context.AllowedEntities, map[string][]string{"project": {"101"}}) {
		t.Fatalf("allowedEntities=%+v", output.Context.AllowedEntities)
	}
	if !reflect.DeepEqual(output.Context.AllowedEntities["project"], output.Scope.IDs) {
		t.Fatalf("context and scope disagree: %+v vs %+v", output.Context.AllowedEntities, output.Scope.IDs)
	}
	output.Context.AllowedEntities["project"][0] = "tampered"
	if output.Scope.IDs[0] != "101" {
		t.Fatal("context and scope share storage")
	}
}

func TestConvertDenies(t *testing.T) {
	expired := facts()
	expired.ValidUntil = time.Now().Add(-time.Second)
	anonymous := facts()
	anonymous.Subject = ""
	malformed := facts()
	malformed.EntityGroups = access.EntityGroups{"project": {"101", "101"}}
	bounded := access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "101"}}}
	for name, tc := range map[string]struct {
		facts     access.Facts
		decision  access.Decision
		dimension string
	}{
		"expired facts":       {expired, bounded, "project"},
		"anonymous facts":     {anonymous, bounded, "project"},
		"malformed entities":  {malformed, bounded, "project"},
		"unbounded decision":  {facts(), access.Decision{}, "project"},
		"empty bounded":       {facts(), access.Decision{Bounded: true}, "project"},
		"other dimension":     {facts(), access.Decision{Bounded: true, Entities: []access.Entity{{Type: "account", ID: "7"}}}, "project"},
		"mixed dimensions":    {facts(), access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "101"}, {Type: "account", ID: "7"}}}, "project"},
		"duplicate entity id": {facts(), access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: "101"}, {Type: "project", ID: "101"}}}, "project"},
		"empty entity id":     {facts(), access.Decision{Bounded: true, Entities: []access.Entity{{Type: "project", ID: ""}}}, "project"},
		"invalid dimension":   {facts(), bounded, "pro/ject"},
	} {
		t.Run(name, func(t *testing.T) {
			output, err := Convert(tc.facts, tc.decision, tc.dimension)
			if err == nil || xresponse.ErrorStatusCode(err, 0) != 403 || output != nil {
				t.Fatalf("output=%+v err=%v", output, err)
			}
		})
	}
}

func TestHandlerDeniesWhenDeciderFails(t *testing.T) {
	handler := Handler("project", func(context.Context) (access.Facts, access.Decision, error) {
		return access.Facts{}, access.Decision{}, errors.New("PRIVATE provider failure")
	})
	if handler.InputType() != reflect.TypeOf(Input{}) || handler.OutputType() != reflect.TypeOf(Output{}) {
		t.Fatalf("handler contract %v -> %v", handler.InputType(), handler.OutputType())
	}
	if reflect.TypeOf(Input{}).NumField() != 0 {
		t.Fatal("access context input must not expose client-bindable fields")
	}
	if _, err := Register(Dependency{ComponentID: "tasks", EntityType: "project"}, nil); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []Dependency{{"../other", "project"}, {"a b", "project"}, {"tasks", ""}, {"tasks", "pro/ject"}, {"", "project"}, {"tasks", ".."}} {
		if _, err := Register(invalid, nil); err == nil {
			t.Fatalf("invalid dependency %+v accepted", invalid)
		}
	}
}

func TestDependsOnParsesDeclaredContexts(t *testing.T) {
	param := func(kind, name string) *spec.Parameter {
		return &spec.Parameter{Name: "Auth", Source: spec.BindSource{Kind: kind, Name: name}}
	}
	tasks := Dependency{ComponentID: "tasks", EntityType: "project"}
	accounts := Dependency{ComponentID: "tasks", EntityType: "account"}
	other := Dependency{ComponentID: "other", EntityType: "project"}
	for name, tc := range map[string]struct {
		parameters []*spec.Parameter
		want       []Dependency
		wantErr    string
	}{
		"none":              {parameters: []*spec.Parameter{param("query", "x"), param("component", "GET:/v1/studio/auth/context")}},
		"own":               {parameters: []*spec.Parameter{param("component", tasks.Reference())}, want: []Dependency{tasks}},
		"repeated same":     {parameters: []*spec.Parameter{param("component", tasks.Reference()), param("component", tasks.Reference())}, want: []Dependency{tasks}},
		"two dimensions":    {parameters: []*spec.Parameter{param("component", tasks.Reference()), param("component", accounts.Reference())}, want: []Dependency{tasks, accounts}},
		"foreign component": {parameters: []*spec.Parameter{param("component", other.Reference())}, want: []Dependency{other}},
		"wrong method":      {parameters: []*spec.Parameter{param("component", "POST:"+tasks.Route())}, wantErr: "must use GET"},
		"missing dimension": {parameters: []*spec.Parameter{param("component", "GET:"+RoutePrefix+"tasks")}, wantErr: "exactly one component and one entity dimension"},
		"extra segment":     {parameters: []*spec.Parameter{param("component", "GET:"+RoutePrefix+"tasks/project/extra")}, wantErr: "exactly one component and one entity dimension"},
		"empty segments":    {parameters: []*spec.Parameter{param("component", "GET:"+RoutePrefix+"/")}, wantErr: "exactly one component and one entity dimension"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := DependsOn(&spec.Component{Parameters: tc.parameters})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
}
