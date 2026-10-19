package sqltransport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/transcribe/dql"
	sqltext "github.com/viant/sqlparser/source"
)

func wholeValueInputEdit(operation readerbuilder.Operation) bool {
	if operation.Type != readerbuilder.OperationUpdateField || operation.Field == nil || operation.Field.SourceName != "" {
		return false
	}
	switch operation.Field.SourceKind {
	case "body", "http_request", "input":
		return true
	}
	return false
}

// Reader Builder treats an empty source name as an omitted update. For whole
// body/request bindings it is an authored location; retain it while patching
// the declaration and its supported options through Datly's source spans.
func editWholeValueInput(source string, field *readerbuilder.Field) (string, error) {
	declarations, err := dql.DeclarationOccurrences(source)
	if err != nil {
		return "", err
	}
	var selected *dql.DeclarationOccurrence
	target := field.ExistingName
	if target == "" {
		target = field.Name
	}
	for index := range declarations {
		p := declarations[index].Parameter
		if p != nil && strings.EqualFold(p.Name, target) {
			if selected != nil {
				return "", fmt.Errorf("field %q is ambiguous", target)
			}
			selected = &declarations[index]
		}
	}
	if selected == nil || selected.Parameter.Source.Kind == "output" {
		return "", fmt.Errorf("input %q was not found", target)
	}
	// Validate the new head with the same declaration parser; do not accept a
	// second declaration or a fluent option injected through a field value.
	head := fmt.Sprintf("$%s<%s>(%s/)", field.Name, field.Type, field.SourceKind)
	parsed, err := dql.DeclarationOccurrences("#define($_ = " + head + ")")
	if err != nil {
		return "", err
	}
	if len(parsed) != 1 || len(parsed[0].Options) != 0 || parsed[0].Parameter.Name != field.Name || parsed[0].Parameter.TypeExpr != field.Type || parsed[0].Parameter.Source.Kind != field.SourceKind || parsed[0].Parameter.Source.Name != "" {
		return "", fmt.Errorf("invalid whole-value input head")
	}
	for _, declaration := range declarations {
		if declaration.Parameter != nil && declaration.Span != selected.Span && strings.EqualFold(declaration.Parameter.Name, field.Name) {
			return "", fmt.Errorf("field %q already exists", field.Name)
		}
	}
	changes := map[string]string{}
	if field.Required != nil {
		value := ".Optional()"
		if *field.Required {
			value = ".Required()"
		}
		changes["required"] = value
		changes["optional"] = value
	}
	if field.UpdateQuerySelector != nil {
		changes["queryselector"] = quotedOption("QuerySelector", *field.UpdateQuerySelector)
	}
	if field.UpdateURI {
		value := ""
		if field.URI != nil {
			value = *field.URI
		}
		changes["withuri"] = quotedOption("WithURI", value)
	}
	if field.UpdateValue {
		changes["value"] = ""
		if field.Value != nil {
			changes["value"] = ".Value(" + strconv.Quote(*field.Value) + ")"
		}
	}
	if field.UpdateCodec {
		changes["withcodec"] = ""
		if field.Codec != nil && field.Codec.Name != "" {
			args := []string{strconv.Quote(field.Codec.Name)}
			for _, arg := range field.Codec.Args {
				args = append(args, strconv.Quote(arg))
			}
			changes["withcodec"] = ".WithCodec(" + strings.Join(args, ",") + ")"
		}
	}
	if field.EmitOutput != nil {
		changes["output"] = ""
		if *field.EmitOutput {
			changes["output"] = ".Output()"
		}
	}
	if field.UpdateDescription {
		value := ""
		if field.Description != nil {
			value = *field.Description
		}
		changes["withdescription"] = quotedOption("WithDescription", value)
	}
	if field.UpdateExample {
		changes["withexample"] = ""
		if field.Example != nil {
			changes["withexample"] = ".WithExample(" + strconv.Quote(*field.Example) + ")"
		}
	}
	type patch struct {
		span dql.SourceSpan
		text string
	}
	patches := []patch{{span: selected.HeadSpan, text: head}}
	seen := map[string]bool{}
	for _, option := range selected.Options {
		key := strings.ToLower(option.Name)
		if value, ok := changes[key]; ok {
			if key == "required" || key == "optional" {
				key = "required"
				seen["optional"] = true
			}
			if seen[key] {
				value = ""
			}
			patches = append(patches, patch{span: option.Span, text: value})
			seen[key] = true
		}
	}
	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tail := ""
	for _, key := range keys {
		if key == "optional" {
			continue
		}
		if !seen[key] {
			tail += changes[key]
		}
	}
	if tail != "" {
		patches = append(patches, patch{span: dql.SourceSpan{Start: selected.OptionInsert, End: selected.OptionInsert}, text: tail})
	}
	sort.SliceStable(patches, func(i, j int) bool { return patches[i].span.Start > patches[j].span.Start })
	candidate := source
	for _, change := range patches {
		candidate, err = dql.ApplyPatch(candidate, change.span, change.text)
		if err != nil {
			return "", err
		}
	}
	if field.Name != selected.Parameter.Name {
		candidate = sqltext.Token("$"+selected.Parameter.Name).ReplaceAll(candidate, "$"+field.Name)
	}
	return candidate, nil
}

func quotedOption(name, value string) string {
	if value == "" {
		return ""
	}
	return "." + name + "(" + strconv.Quote(value) + ")"
}
