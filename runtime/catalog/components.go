package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_publications/store_runtime_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type ComponentCatalog struct{ db *sql.DB }

func NewComponentCatalog(db *sql.DB) *ComponentCatalog { return &ComponentCatalog{db: db} }

func (c *ComponentCatalog) LoadPublishedComponents(ctx context.Context) ([]*PublishedReport, error) {
	if c == nil || c.db == nil {
		return nil, fmt.Errorf("published component catalog database is unavailable")
	}
	resources := resource.New()
	if err := resources.Register(stored.PublicationDatlyResourceNamespace, stored.PublicationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: c.db}
	if err := connector.RegisterConnector("studio", c.db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.PublicationComponent{}), "store_runtime_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{Status: "active", Live: true, Has: &stored.InputHas{Status: true, Live: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, fmt.Errorf("load published reports: %w", err)
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("published component catalog returned %T", value)
	}
	result := make([]*PublishedReport, 0, len(output.Publications))
	for _, row := range output.Publications {
		if row == nil || row.ReportId == "" || !row.ReportLive || row.PublicationStatus != "active" ||
			row.PublicationActiveVersionNo <= 0 || row.PublicationActiveGeneration == nil || row.VersionSourceRevision <= 0 {
			return nil, fmt.Errorf("published component catalog returned an invalid row")
		}
		result = append(result, publishedFromStored(row))
	}
	return result, nil
}

func publishedFromStored(row *stored.PublishedComponent) *PublishedReport {
	report := &sdk.Report{ID: row.ReportId, Namespace: row.ReportNamespace, Slug: row.ReportSlug,
		Title: row.ReportTitle, OwnerID: row.ReportOwnerId, OwnerPackage: sdk.OwnerPackageSegment(row.ReportOwnerId),
		Status: row.ReportStatus, DefaultConnectorName: row.ReportConnector,
		ComponentScope: row.ReportComponentScope, ComponentName: row.ReportComponentName,
		ETag: row.ReportEtag, CreatedAt: row.ReportCreatedAt, UpdatedAt: row.ReportUpdatedAt}
	if row.ReportDescription != nil {
		report.Description = *row.ReportDescription
	}
	if row.ReportDraftVersion != nil {
		draft := *row.ReportDraftVersion
		report.CurrentDraftVersion = &draft
	}
	active := *row.PublicationActiveGeneration
	publication := &sdk.Publication{ReportID: row.ReportId, ActiveVersionNo: row.PublicationActiveVersionNo,
		DesiredGeneration: row.PublicationDesiredGeneration, ActiveGeneration: &active,
		Status: row.PublicationStatus, PublishedAt: row.PublicationPublishedAt}
	if row.PublicationRuntimeRevision != nil {
		publication.RuntimeRevision = *row.PublicationRuntimeRevision
	}
	version := &sdk.ReportVersion{ReportID: row.ReportId, VersionNo: row.PublicationActiveVersionNo,
		State: row.VersionState, AuthoringMode: row.VersionAuthoringMode,
		SpecFormatVersion: row.VersionSpecFormatVersion, SpecHash: row.VersionSpecHash,
		CompileStatus: row.VersionCompileStatus, DatlyVersion: row.VersionDatlyVersion,
		CompilerVersion: row.VersionCompilerVersion, SourceRevision: row.VersionSourceRevision,
		CreatedBy: row.VersionCreatedBy, CreatedAt: row.VersionCreatedAt,
		ValidatedAt: row.VersionValidatedAt, PublishedAt: row.VersionPublishedAt}
	version.AuthoredSQL = optionalString(row.VersionAuthoredSql)
	version.AuthoredDQL = optionalString(row.VersionAuthoredDql)
	version.GeneratedDQL = optionalString(row.VersionGeneratedDql)
	version.Notes = optionalString(row.VersionNotes)
	version.ComponentSpec = cloneJSON(row.VersionComponentSpecJson)
	version.DQLExportLimits = cloneJSON(row.VersionDqlExportLimitsJson)
	version.TypeManifest = cloneJSON(row.VersionTypeManifestJson)
	version.ResourceManifest = cloneJSON(row.VersionResourceManifestJson)
	version.ComponentDescriptor = cloneJSON(row.VersionComponentDescriptorJson)
	version.CompileDiagnostics = cloneJSON(row.VersionCompileDiagnosticsJson)
	return &PublishedReport{Report: report, Version: version, Publication: publication}
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneJSON(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}
