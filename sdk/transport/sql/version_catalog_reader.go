package sqltransport

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type versionCatalogRequest struct {
	ReportID                                       string
	VersionNo                                      int
	State, AuthoringMode, CompileStatus, CreatedBy string
	Limit, Offset                                  int
}

func (t *Transport) readVersionCatalog(ctx context.Context, request versionCatalogRequest) ([]*sdk.ReportVersion, error) {
	resources := resource.New()
	if err := resources.Register(stored.VersionDatlyResourceNamespace, stored.VersionDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.VersionComponent{}), "store_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ReportId: request.ReportID, VersionNo: request.VersionNo,
		State: request.State, AuthoringMode: request.AuthoringMode, CompileStatus: request.CompileStatus,
		CreatedBy: request.CreatedBy, Limit: request.Limit, Offset: request.Offset,
		Has: &stored.InputHas{ReportId: true, VersionNo: request.VersionNo > 0,
			State: request.State != "", AuthoringMode: request.AuthoringMode != "",
			CompileStatus: request.CompileStatus != "", CreatedBy: request.CreatedBy != "",
			Limit: true, Offset: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("version catalog returned %T", value)
	}
	if len(output.Versions) > request.Limit {
		return nil, fmt.Errorf("version catalog exceeded requested limit")
	}
	result := make([]*sdk.ReportVersion, 0, len(output.Versions))
	for _, row := range output.Versions {
		if row == nil || row.ReportId != request.ReportID || request.VersionNo > 0 && row.VersionNo != request.VersionNo {
			return nil, fmt.Errorf("version catalog returned a mismatched row")
		}
		result = append(result, &sdk.ReportVersion{ReportID: row.ReportId, VersionNo: row.VersionNo,
			State: row.State, AuthoringMode: row.AuthoringMode,
			AuthoredSQL: versionOptionalString(row.AuthoredSql), AuthoredDQL: versionOptionalString(row.AuthoredDql),
			ComponentSpec: versionJSON(row.ComponentSpecJson), DQLExportLimits: versionJSON(row.DqlExportLimitsJson),
			TypeManifest: versionJSON(row.TypeManifestJson), ResourceManifest: versionJSON(row.ResourceManifestJson),
			ComponentDescriptor: versionJSON(row.ComponentDescriptorJson),
			SpecFormatVersion:   row.SpecFormatVersion, SpecHash: row.SpecHash,
			GeneratedDQL: versionOptionalString(row.GeneratedDql), CompileStatus: row.CompileStatus,
			CompileDiagnostics: versionJSON(row.CompileDiagnosticsJson), DatlyVersion: row.DatlyVersion,
			CompilerVersion: row.CompilerVersion, SourceRevision: row.SourceRevision,
			Notes: versionOptionalString(row.Notes), CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt, ValidatedAt: row.ValidatedAt, PublishedAt: row.PublishedAt})
	}
	return result, nil
}

func versionOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func versionJSON(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}
