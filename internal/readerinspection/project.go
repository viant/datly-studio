package readerinspection

import (
	"encoding/json"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/transcribe"
)

// Project keeps Studio's inspection wire shape consistent across the generic
// SDK and native Datly endpoint. Without a DQL grant, only curated metadata
// crosses the boundary; Reader Builder internals contain authored SQL.
func Project(version *sdk.ReportVersion, response *readerbuilder.Response, capabilities sdk.ReportCapabilities) *sdk.ReaderInspection {
	version = RedactVersion(version, capabilities.CanUseDQL)
	if response == nil {
		return &sdk.ReaderInspection{Version: version, Capabilities: capabilities}
	}
	var structure json.RawMessage
	if capabilities.CanUseDQL {
		structure, _ = json.Marshal(response.Structure)
	} else {
		structure = redactedStructure(response.Structure)
	}
	diagnostics := RedactDiagnostics(projectDiagnostics(response.Diagnostics), capabilities.CanUseDQL)
	result := &sdk.ReaderInspection{Version: version, Structure: structure,
		Diagnostics: diagnostics, Capabilities: capabilities}
	if capabilities.CanUseDQL {
		result.DQL = response.DQL
	}
	return result
}

func RedactVersion(version *sdk.ReportVersion, allowed bool) *sdk.ReportVersion {
	if version == nil || allowed {
		return version
	}
	copy := *version
	copy.AuthoredDQL, copy.GeneratedDQL = "", ""
	// Compiler diagnostics may quote source text or include generated DQL.
	copy.CompileDiagnostics = nil
	return &copy
}

// RedactDiagnostics retains stable diagnostic identity and location without
// exposing messages or hints that may quote authored source.
func RedactDiagnostics(diagnostics []sdk.Diagnostic, allowed bool) []sdk.Diagnostic {
	if allowed || len(diagnostics) == 0 {
		return diagnostics
	}
	result := append([]sdk.Diagnostic(nil), diagnostics...)
	for index := range result {
		result[index].Message, result[index].Hint = "", ""
	}
	return result
}

func redactedStructure(structure *readerbuilder.Structure) json.RawMessage {
	if structure == nil {
		return json.RawMessage(`null`)
	}
	type view struct {
		Name                string `json:"name"`
		SourceProjectionAll bool   `json:"sourceProjectionAll,omitempty"`
	}
	projected := struct {
		Status string `json:"status"`
		Views  []view `json:"views,omitempty"`
	}{Status: structure.Status}
	for _, item := range structure.Views {
		projected.Views = append(projected.Views, view{Name: item.Name, SourceProjectionAll: item.SourceProjectionAll})
	}
	data, _ := json.Marshal(projected)
	return data
}

func projectDiagnostics(diagnostics []*transcribe.Diagnostic) []sdk.Diagnostic {
	result := make([]sdk.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic == nil {
			continue
		}
		result = append(result, sdk.Diagnostic{Severity: string(diagnostic.Severity), Code: diagnostic.Code,
			Message: diagnostic.Message, Hint: diagnostic.Hint,
			Line: diagnostic.Span.Start.Line, Column: diagnostic.Span.Start.Char})
	}
	return result
}
