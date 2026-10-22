package managementskills

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/viant/datly/authoring/readerbuilder"
)

// Keep the published operation schema tied to the linked builder's JSON types.
// Regenerate explicitly with UPDATE_SKILL_CATALOG=1 go test ./internal/managementskills.
func TestPublishedBuilderOperationSchemaMatchesLinkedTypes(t *testing.T) {
	defs := map[string]any{}
	var project func(reflect.Type) map[string]any
	project = func(typ reflect.Type) map[string]any {
		if typ.Kind() == reflect.Pointer {
			return project(typ.Elem())
		}
		switch typ.Kind() {
		case reflect.Struct:
			name := typ.Name()
			if _, found := defs[name]; !found {
				defs[name] = map[string]any{}
				props := map[string]any{}
				required := []string{}
				for i := 0; i < typ.NumField(); i++ {
					f := typ.Field(i)
					parts := strings.Split(f.Tag.Get("json"), ",")
					key := parts[0]
					if key == "-" || !f.IsExported() {
						continue
					}
					if key == "" {
						key = f.Name
					}
					props[key] = project(f.Type)
					if len(parts) < 2 || parts[1] != "omitempty" {
						required = append(required, key)
					}
				}
				defs[name] = map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
			}
			return map[string]any{"$ref": "#/$defs/" + name}
		case reflect.Slice:
			return map[string]any{"type": "array", "items": project(typ.Elem())}
		case reflect.Map:
			return map[string]any{"type": "object", "additionalProperties": project(typ.Elem())}
		case reflect.Bool:
			return map[string]any{"type": "boolean"}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return map[string]any{"type": "integer"}
		default:
			return map[string]any{"type": "string"}
		}
	}
	root := project(reflect.TypeFor[readerbuilder.Operation]())
	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	root["$defs"] = defs
	root["$comment"] = "Structural payload types and operation envelopes; shared mutation fields have operation-specific semantic requirements checked by the native builder."
	operations := map[readerbuilder.OperationType]string{
		readerbuilder.OperationInspect: "", readerbuilder.OperationCreateReader: "reader", readerbuilder.OperationSetPackage: "package",
		readerbuilder.OperationAddField: "field", readerbuilder.OperationUpdateField: "field", readerbuilder.OperationRemoveField: "field",
		readerbuilder.OperationAddFieldPredicate: "predicate", readerbuilder.OperationUpdateFieldPredicate: "predicate", readerbuilder.OperationRemoveFieldPredicate: "predicate",
		readerbuilder.OperationUpdatePredicateGroup: "predicateGroup", readerbuilder.OperationUpdatePredicateComposition: "predicateComposition",
		readerbuilder.OperationAddFunction: "function", readerbuilder.OperationUpdateFunction: "function", readerbuilder.OperationRemoveFunction: "function",
		readerbuilder.OperationSetSetting: "setting", readerbuilder.OperationAddView: "view", readerbuilder.OperationUpdateView: "view", readerbuilder.OperationRemoveView: "view",
		readerbuilder.OperationUpdateRelation: "relation", readerbuilder.OperationSetColumnRole: "columnRole", readerbuilder.OperationSetColumnContract: "column", readerbuilder.OperationBatch: "operations",
	}
	// Keys are sorted by JSON encoding; each variant declares its exact payload.
	variants := map[string]any{}
	for operation, field := range operations {
		required := []string{"type"}
		if field != "" {
			required = append(required, field)
		}
		variants[string(operation)] = map[string]any{"required": required, "payloadField": field}
	}
	keys := make([]string, 0, len(variants))
	for key := range variants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	branches := []any{}
	for _, key := range keys {
		entry := variants[key].(map[string]any)
		branches = append(branches, map[string]any{"properties": map[string]any{"type": map[string]any{"const": key}}, "required": entry["required"]})
	}
	defs["Operation"].(map[string]any)["oneOf"] = branches
	root["x-operation-contracts"] = variants
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := "skills/datly-studio-authoring/references/builder-operations.schema.json"
	if os.Getenv("UPDATE_SKILL_CATALOG") == "1" {
		if err = os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(data) {
		t.Fatal("published builder schema is stale; regenerate against the linked Datly version")
	}
}
