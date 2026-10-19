package sqltransport

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/dql"
)

// Datly accepts fully qualified package types as well as Go import aliases.
var qualifiedOutputType = regexp.MustCompile(`(?:[A-Za-z_][A-Za-z0-9_.-]*/)+[A-Za-z_][A-Za-z0-9_.-]*\.([A-Za-z_][A-Za-z0-9_]*)`)

// The pinned Reader Builder restricts updateField/removeField to inputs.
// Output commands use its source-span parser and compiler, and the same
// version/revision/persistence guards as every other Studio builder command.
func applyOutputField(ctx context.Context, service *readerbuilder.Service, source string, operation readerbuilder.Operation) readerbuilder.Response {
	candidate, err := editOutputField(source, operation)
	return compileFieldEdit(ctx, service, source, candidate, err)
}

func compileFieldEdit(ctx context.Context, service *readerbuilder.Service, source, candidate string, err error) readerbuilder.Response {
	if err == nil {
		result := service.Apply(ctx, readerbuilder.Request{DQL: candidate, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
		if len(result.Diagnostics) == 0 && result.Structure != nil && result.Structure.Status == "complete" {
			result.Applied = true
			return result
		}
		original := service.Apply(ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
		original.Applied = false
		original.Diagnostics = result.Diagnostics
		return original
	}
	result := service.Apply(ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	result.Applied = false
	result.Diagnostics = append(result.Diagnostics, &transcribe.Diagnostic{Code: "DQL-FIELD", Severity: transcribe.SeverityError, Message: err.Error()})
	return result
}

func editOutputField(source string, operation readerbuilder.Operation) (string, error) {
	field := operation.Field
	if field == nil {
		return "", fmt.Errorf("output field is required")
	}
	target := field.ExistingName
	if target == "" {
		target = field.Name
	}
	declarations, err := dql.DeclarationOccurrences(source)
	if err != nil {
		return "", err
	}
	var matches []dql.DeclarationOccurrence
	for _, declaration := range declarations {
		if declaration.Parameter != nil && strings.EqualFold(declaration.Parameter.Source.Kind, "output") && declaration.Parameter.Name == target {
			matches = append(matches, declaration)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("output field %q resolves to %d declarations; exactly one is required", target, len(matches))
	}
	selected := matches[0]
	if operation.Type == "removeOutputField" {
		return dql.ApplyPatch(source, selected.Span, "")
	}
	if operation.Type != "updateOutputField" {
		return "", fmt.Errorf("unsupported output field operation %q", operation.Type)
	}
	if !token.IsIdentifier(field.Name) || token.IsKeyword(field.Name) || strings.HasPrefix(field.Name, "_") || field.SourceKind != "output" || field.SourceName == "" || field.Type == "" || strings.ContainsAny(field.Type+field.SourceName, "\r\n)") {
		return "", fmt.Errorf("output name, type, kind output, and kind source are required")
	}
	if field.Type != "?" {
		parsedType, parseErr := parser.ParseFile(token.NewFileSet(), "output_type.go", "package output\ntype OutputType "+qualifiedOutputType.ReplaceAllString(field.Type, "qualified.$1"), 0)
		if parseErr != nil || len(parsedType.Decls) != 1 {
			return "", fmt.Errorf("output type must be one valid Go type expression")
		}
	}
	for _, declaration := range declarations {
		if declaration.Parameter != nil && declaration.Span != selected.Span && strings.EqualFold(declaration.Parameter.Source.Kind, "output") && strings.EqualFold(declaration.Parameter.Name, field.Name) {
			return "", fmt.Errorf("output field %q already exists", field.Name)
		}
	}
	// Replace only the head; preserve codecs, tags, descriptions, and SQL bodies.
	// Input declarations with the same name retain their independent identity.
	head := fmt.Sprintf("$%s<%s>(output/%s)", field.Name, field.Type, field.SourceName)
	candidate, err := dql.ApplyPatch(source, selected.HeadSpan, head)
	if err != nil {
		return "", err
	}
	parsed, err := dql.DeclarationOccurrences(candidate)
	if err != nil {
		return "", err
	}
	if len(parsed) != len(declarations) {
		return "", fmt.Errorf("output edit changed declaration structure")
	}
	for _, declaration := range parsed {
		if declaration.Parameter != nil && declaration.Parameter.Name == field.Name && declaration.Parameter.Source.Kind == "output" {
			if declaration.Parameter.Source.Name != field.SourceName || declaration.Parameter.TypeExpr != field.Type {
				return "", fmt.Errorf("invalid output declaration head")
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("output declaration was not preserved")
}
