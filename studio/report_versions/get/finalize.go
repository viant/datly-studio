package get

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	xresponse "github.com/viant/xdatly/response"
)

// Finalize projects the authorized row into the direct Studio SDK response.
func (output *VersionGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*VersionGetInput]()).(*VersionGetInput)
	if !ok || input == nil {
		return fmt.Errorf("version get bound input is unavailable")
	}
	return output.ProjectItem(input.ReportId, input.VersionNo)
}

// ProjectItem performs the side-effect-free response projection. A parent
// component may need these fields before its root transaction completes;
// Datly still owns success-hook timing and response publication.
func (output *VersionGetOutput) ProjectItem(reportID string, versionNo int) error {
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("report version not found")}
	}
	item := output.Item
	if item.ReportId != reportID || item.VersionNo != versionNo {
		return fmt.Errorf("version get returned a mismatched row")
	}
	output.ResponseReportId = item.ReportId
	output.ResponseVersionNo = item.VersionNo
	output.State = item.State
	output.AuthoringMode = item.AuthoringMode
	if item.AuthoredSql != nil {
		output.AuthoredSql = *item.AuthoredSql
	}
	if item.AuthoredDql != nil {
		output.AuthoredDql = *item.AuthoredDql
	}
	output.ComponentSpec = item.ComponentSpecJson
	output.DqlExportLimits = item.DqlExportLimitsJson
	output.TypeManifest = item.TypeManifestJson
	output.ResourceManifest = item.ResourceManifestJson
	output.ComponentDescriptor = item.ComponentDescriptorJson
	output.SpecFormatVersion = item.SpecFormatVersion
	output.SpecHash = item.SpecHash
	if item.GeneratedDql != nil {
		output.GeneratedDql = *item.GeneratedDql
	}
	output.CompileStatus = item.CompileStatus
	output.CompileDiagnostics = item.CompileDiagnosticsJson
	output.DatlyVersion = item.DatlyVersion
	output.CompilerVersion = item.CompilerVersion
	output.SourceRevision = item.SourceRevision
	if item.Notes != nil {
		output.Notes = *item.Notes
	}
	output.CreatedBy = item.CreatedBy
	output.CreatedAt = item.CreatedAt
	output.ValidatedAt = item.ValidatedAt
	output.PublishedAt = item.PublishedAt
	return nil
}
