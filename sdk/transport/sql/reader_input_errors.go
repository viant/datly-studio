package sqltransport

import (
	"encoding/json"
	"fmt"
	"github.com/viant/datly/transcribe/dql"
	"sort"
	"strconv"
	"strings"
)

// error fields extend the pinned Reader Builder without reconstructing declarations.
// A nil field preserves the option; zero/empty explicitly removes it.
func editInputErrors(source string, raw json.RawMessage) (string, error) {
	var op struct {
		Type  string `json:"type"`
		Field *struct {
			Name    string  `json:"name"`
			Code    *int    `json:"errorStatusCode"`
			Message *string `json:"errorMessage"`
		} `json:"field"`
		Operations []json.RawMessage `json:"operations"`
	}
	if err := json.Unmarshal(raw, &op); err != nil {
		return "", err
	}
	if op.Type == "batch" {
		var err error
		for _, child := range op.Operations {
			source, err = editInputErrors(source, child)
			if err != nil {
				return "", err
			}
		}
		return source, nil
	}
	if (op.Type != "addField" && op.Type != "updateField") || op.Field == nil || (op.Field.Code == nil && op.Field.Message == nil) {
		return source, nil
	}
	if op.Field.Code != nil && *op.Field.Code != 0 && (*op.Field.Code < 400 || *op.Field.Code > 599) {
		return "", fmt.Errorf("parameter error code must be 400–599, or empty")
	}
	declarations, err := dql.DeclarationOccurrences(source)
	if err != nil {
		return "", err
	}
	var selected *dql.DeclarationOccurrence
	for i := range declarations {
		p := declarations[i].Parameter
		if p != nil && strings.EqualFold(p.Name, op.Field.Name) && p.Source.Kind != "output" {
			if selected != nil {
				return "", fmt.Errorf("ambiguous input %q", op.Field.Name)
			}
			selected = &declarations[i]
		}
	}
	if selected == nil {
		return "", fmt.Errorf("input %q was not found", op.Field.Name)
	}
	changes := map[string]string{}
	if op.Field.Code != nil {
		changes["withstatuscode"] = ""
		if *op.Field.Code != 0 {
			changes["withstatuscode"] = ".WithStatusCode(" + strconv.Itoa(*op.Field.Code) + ")"
		}
	}
	if op.Field.Message != nil {
		changes["witherrormessage"] = quotedOption("WithErrorMessage", *op.Field.Message)
	}
	type patch struct {
		span dql.SourceSpan
		text string
	}
	var patches []patch
	for _, option := range selected.Options {
		key := strings.ToLower(option.Name)
		if value, ok := changes[key]; ok {
			patches = append(patches, patch{option.Span, value})
			changes[key] = ""
		}
	}
	tail := changes["withstatuscode"] + changes["witherrormessage"]
	if tail != "" {
		patches = append(patches, patch{dql.SourceSpan{Start: selected.OptionInsert, End: selected.OptionInsert}, tail})
	}
	sort.SliceStable(patches, func(i, j int) bool { return patches[i].span.Start > patches[j].span.Start })
	for _, p := range patches {
		source, err = dql.ApplyPatch(source, p.span, p.text)
		if err != nil {
			return "", err
		}
	}
	return source, nil
}
