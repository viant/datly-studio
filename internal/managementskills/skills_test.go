package managementskills

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/mcp/resource"
	"github.com/viant/datly/spec"
	"github.com/viant/mcp-protocol/extension/skills"
)

func TestEmbeddedSkillPublishesThroughNativeResourcePlans(t *testing.T) {
	plans, err := (resource.Publisher{}).Compile(context.Background(), []resource.Folder{Folder()})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 14 {
		t.Fatalf("resource count=%d", len(plans))
	}
	handoffPublished := false
	resourceChecksPublished := false
	groupingPublished := false
	requirementsPublished := false
	for _, plan := range plans {
		if plan.URI() == "mcp://datastudio/skills/datly-studio-authoring/references/reader-from-requirements.md" {
			requirementsPublished = true
		}
		if plan.URI() == "mcp://datastudio/skills/datly-studio-authoring/references/parameterized-grouping-howto.md" {
			groupingPublished = true
		}
		if plan.URI() == "mcp://datastudio/skills/datly-studio-authoring/references/security-and-resource-checks.md" {
			resourceChecksPublished = true
		}
		if plan.URI() == "mcp://datastudio/skills/datly-studio-authoring/references/report-consumers.md" {
			handoffPublished = true
		}
	}
	if !resourceChecksPublished {
		t.Fatal("resource-check guide must be addressable through management MCP")
	}
	if !handoffPublished {
		t.Fatal("report-consumer contract must be addressable through management MCP")
	}
	if !groupingPublished {
		t.Fatal("parameterized grouping guide must be addressable through management MCP")
	}
	if !requirementsPublished {
		t.Fatal("reader requirements guide must be addressable through management MCP")
	}
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = skills.Frontmatter(data); err != nil {
		t.Fatal(err)
	}
}

func TestSkillReaderExampleCompilesWithNativeBuilder(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/references/dql.md")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.SplitN(string(data), "```sql\n", 2)[1]
	text = strings.SplitN(text, "```", 2)[0]
	result := readerbuilder.New(readerbuilder.Config{}).Apply(context.Background(), readerbuilder.Request{DQL: text, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if result.Structure == nil || result.Structure.Component == nil {
		t.Fatalf("example did not compile: %+v", result.Diagnostics)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("example: %+v", d)
		}
	}
}

func TestWorkedCubeExampleCompilesWithNativeBuilder(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/references/worked-example.md")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.SplitN(string(data), "```sql\n", 2)[1]
	text = strings.SplitN(text, "```", 2)[0]
	result := readerbuilder.New(readerbuilder.Config{}).Apply(context.Background(), readerbuilder.Request{DQL: text, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if result.Structure == nil || result.Structure.Component == nil {
		t.Fatalf("cube example did not compile: %+v", result.Diagnostics)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("cube example: %+v", d)
		}
	}
	for _, name := range []string{"Dates", "AccountIDs"} {
		found := false
		for _, parameter := range result.Structure.Component.Parameters {
			if parameter != nil && parameter.Name == name {
				found = parameter.Required != nil && *parameter.Required
			}
		}
		if !found {
			t.Fatalf("worked example must explicitly require %s", name)
		}
	}
	component := result.Structure.Component
	if len(component.RootView.Relations) != 2 {
		t.Fatalf("expected two dictionary relations: %+v", component.RootView.Relations)
	}
	for _, relation := range component.RootView.Relations {
		if relation.Cardinality != spec.CardinalityOne || relation.View.AllowNulls == nil || !*relation.View.AllowNulls || len(relation.On) != 1 {
			t.Fatalf("dictionary must retain one complete nullable to-one key: %+v", relation)
		}
	}
	if component.Settings == nil || component.Settings.Report == nil || component.Settings.Report.MCPTool == nil || !*component.Settings.Report.MCPTool {
		t.Fatal("example does not expose its cube")
	}
}

func TestEntryPointPredicateExampleCompilesWithNativeBuilder(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := strings.Split(string(data), "```sql\n")
	if len(blocks) < 3 {
		t.Fatal("entry point must explain input declarations and builder expansion")
	}
	declarations := strings.SplitN(blocks[1], "```", 2)[0]
	expansion := strings.SplitN(blocks[2], "```", 2)[0]
	dql := "#package('example.com/records/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/records','GET'))\n" + declarations + "\n#define($_ = $Rows<[]*Record>(output/view))\nSELECT records.*, type(records,'Record') FROM (SELECT r.id,r.name FROM records r\n" + expansion + ") records"
	result := readerbuilder.New(readerbuilder.Config{}).Apply(context.Background(), readerbuilder.Request{DQL: dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if result.Structure == nil || result.Structure.Component == nil {
		t.Fatalf("entry example failed: %+v", result.Diagnostics)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("entry example: %+v", d)
		}
	}
	required := map[string]bool{}
	for _, p := range result.Structure.Component.Parameters {
		if p != nil {
			required[p.Name] = p.Required != nil && *p.Required
		}
	}
	if !required["RecordID"] || required["Name"] {
		t.Fatalf("required/optional input contract changed: %+v", required)
	}
}

func TestCompositeDictionarySkillFragmentCompiles(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/references/worked-example.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := strings.Split(string(data), "```sql\n")
	if len(blocks) < 3 {
		t.Fatal("missing composite dictionary fragment")
	}
	fragment := strings.SplitN(blocks[2], "```", 2)[0]
	dql := "#package('example.com/regions/read')\n#setting($_ = $connector('warehouse'))\n#setting($_ = $route('/regions','GET'))\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT fact.*,type(fact,'Row'),region.*,use_connector(region,'dictionary'),allow_nulls(region) FROM (SELECT country_code,region_code FROM regional_sales) fact " + fragment
	result := readerbuilder.New(readerbuilder.Config{}).Apply(context.Background(), readerbuilder.Request{DQL: dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("composite fragment: %+v", d)
		}
	}
	if result.Structure == nil || result.Structure.Component == nil || len(result.Structure.Component.RootView.Relations) != 1 {
		t.Fatal("missing composite relation")
	}
	relation := result.Structure.Component.RootView.Relations[0]
	if relation.Cardinality != spec.CardinalityOne || len(relation.On) != 2 {
		t.Fatalf("composite to-one keys lost: %+v", relation)
	}
}

func TestSelectorBindingSkillExampleCompiles(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/references/dql.md")
	if err != nil {
		t.Fatal(err)
	}
	section := strings.SplitN(string(data), "### Public MCP paging and field selection", 2)[1]
	fragment := strings.SplitN(strings.SplitN(section, "```sql\n", 2)[1], "```", 2)[0]
	dql := "#package('example.com/records/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/records','GET'))\n" + fragment + "\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT records.*,type(records,'Row'),set_limit(records,20) FROM (SELECT id,name FROM records) records"
	result := readerbuilder.New(readerbuilder.Config{Name: "Records"}).Apply(context.Background(), readerbuilder.Request{DQL: dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("selector example: %+v", d)
		}
	}
	if result.Structure == nil || result.Structure.Component == nil {
		t.Fatal("missing component")
	}
	expected := map[string]spec.SelectorProperty{"Fields": spec.SelectorPropertyFields, "Limit": spec.SelectorPropertyLimit, "Offset": spec.SelectorPropertyOffset, "OrderBy": spec.SelectorPropertyOrderBy}
	for _, p := range result.Structure.Component.Parameters {
		if want, ok := expected[p.Name]; ok {
			if p.QuerySelector == nil || p.QuerySelector.View != "Records" || p.QuerySelector.Property != want {
				t.Fatalf("selector binding lost: %+v", p)
			}
			delete(expected, p.Name)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing selectors: %v", expected)
	}
}

func TestAggregateThresholdSkillFragmentsCompile(t *testing.T) {
	data, err := fs.ReadFile(source, "skills/datly-studio-authoring/references/parameterized-grouping-howto.md")
	if err != nil {
		t.Fatal(err)
	}
	section := strings.SplitN(string(data), "### Aggregate threshold authoring pattern", 2)[1]
	blocks := strings.Split(section, "```sql\n")
	if len(blocks) != 3 {
		t.Fatal("expected declaration and HAVING fragments")
	}
	declaration := strings.SplitN(blocks[1], "```", 2)[0]
	having := strings.SplitN(blocks[2], "```", 2)[0]
	dql := "#package('example.com/counts/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/counts','GET'))\n" + declaration + "\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT counts.*,type(counts,'Row'),grouping_enabled(counts),tag(counts.occurrences,'groupable:\"false\"') FROM (SELECT category,COUNT(*) AS occurrences FROM observations\n${predicate.Builder().CombineAnd($predicate.FilterGroup(0,\"AND\")).Build(\"WHERE\")}\nGROUP BY category\n" + having + ") counts"
	result := readerbuilder.New(readerbuilder.Config{}).Apply(context.Background(), readerbuilder.Request{DQL: dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if result.Structure == nil || result.Structure.Component == nil {
		t.Fatalf("missing aggregate contract: %+v", result.Diagnostics)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("aggregate threshold: %+v", d)
		}
	}
}
