package versionprojection

import (
	"encoding/json"

	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_versions/store_catalog"
)

func FromCatalog(row *stored.StoredVersion) *sdk.ReportVersion {
	if row == nil {
		return nil
	}
	value := &sdk.ReportVersion{ReportID: row.ReportId, VersionNo: row.VersionNo, State: row.State,
		AuthoringMode: row.AuthoringMode, ComponentSpec: append(json.RawMessage(nil), row.ComponentSpecJson...),
		DQLExportLimits:     append(json.RawMessage(nil), row.DqlExportLimitsJson...),
		TypeManifest:        append(json.RawMessage(nil), row.TypeManifestJson...),
		ResourceManifest:    append(json.RawMessage(nil), row.ResourceManifestJson...),
		ComponentDescriptor: append(json.RawMessage(nil), row.ComponentDescriptorJson...),
		SpecFormatVersion:   row.SpecFormatVersion, SpecHash: row.SpecHash,
		CompileStatus: row.CompileStatus, CompileDiagnostics: append(json.RawMessage(nil), row.CompileDiagnosticsJson...),
		DatlyVersion: row.DatlyVersion, CompilerVersion: row.CompilerVersion,
		SourceRevision: row.SourceRevision, CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt, ValidatedAt: row.ValidatedAt, PublishedAt: row.PublishedAt}
	if row.AuthoredSql != nil {
		value.AuthoredSQL = *row.AuthoredSql
	}
	if row.AuthoredDql != nil {
		value.AuthoredDQL = *row.AuthoredDql
	}
	if row.GeneratedDql != nil {
		value.GeneratedDQL = *row.GeneratedDql
	}
	if row.Notes != nil {
		value.Notes = *row.Notes
	}
	return value
}
