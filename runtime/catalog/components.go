package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

type ComponentCatalog struct{ db *sql.DB }

func NewComponentCatalog(db *sql.DB) *ComponentCatalog { return &ComponentCatalog{db: db} }

func (c *ComponentCatalog) LoadPublishedComponents(ctx context.Context) ([]*PublishedReport, error) {
	rows, err := c.db.QueryContext(ctx, `
SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id, r.status, r.default_connector_name, r.component_scope, r.component_name, r.current_draft_version, r.etag, r.created_at, r.updated_at,
       p.active_version_no, p.desired_generation, p.active_generation, p.publication_status, p.runtime_revision, p.published_at,
       v.state, v.authoring_mode, v.authored_sql, v.authored_dql, v.component_spec_json, v.spec_format_version, v.spec_hash, v.generated_dql, v.dql_export_limits_json, v.type_manifest_json, v.resource_manifest_json, v.component_descriptor_json, v.compile_status, v.compile_diagnostics_json, v.datly_version, v.compiler_version, v.source_revision, v.notes, v.created_by, v.created_at, v.validated_at, v.published_at
FROM report_publications p
JOIN reports r ON r.id = p.report_id AND r.deleted_at IS NULL
JOIN report_versions v ON v.report_id = p.report_id AND v.version_no = p.active_version_no
WHERE p.publication_status = 'active'
ORDER BY p.published_at DESC, r.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("load published reports: %w", err)
	}
	defer rows.Close()
	var result []*PublishedReport
	for rows.Next() {
		value, err := scanPublishedReport(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanPublishedReport(scanner interface{ Scan(dest ...any) error }) (*PublishedReport, error) {
	var report sdk.Report
	var publication sdk.Publication
	var version sdk.ReportVersion
	var description, authoredSQL, authoredDQL, spec, generatedDQL, limits, typeManifest, resourceManifest, descriptor, diagnostics, notes sql.NullString
	var draft sql.NullInt64
	var activeGeneration sql.NullInt64
	var publishedAt, validatedAt, versionPublishedAt sql.NullTime
	if err := scanner.Scan(
		&report.ID, &report.Namespace, &report.Slug, &report.Title, &description, &report.OwnerID, &report.Status, &report.DefaultConnectorName, &report.ComponentScope, &report.ComponentName, &draft, &report.ETag, &report.CreatedAt, &report.UpdatedAt,
		&publication.ActiveVersionNo, &publication.DesiredGeneration, &activeGeneration, &publication.Status, &publication.RuntimeRevision, &publishedAt,
		&version.State, &version.AuthoringMode, &authoredSQL, &authoredDQL, &spec, &version.SpecFormatVersion, &version.SpecHash, &generatedDQL, &limits, &typeManifest, &resourceManifest, &descriptor, &version.CompileStatus, &diagnostics, &version.DatlyVersion, &version.CompilerVersion, &version.SourceRevision, &notes, &version.CreatedBy, &version.CreatedAt, &validatedAt, &versionPublishedAt,
	); err != nil {
		return nil, err
	}
	report.Description = description.String
	report.OwnerPackage = sdk.OwnerPackageSegment(report.OwnerID)
	if draft.Valid {
		value := int(draft.Int64)
		report.CurrentDraftVersion = &value
	}
	publication.ReportID = report.ID
	if activeGeneration.Valid {
		value := activeGeneration.Int64
		publication.ActiveGeneration = &value
	}
	if publishedAt.Valid {
		value := publishedAt.Time
		publication.PublishedAt = &value
	}
	version.ReportID, version.VersionNo = report.ID, publication.ActiveVersionNo
	version.AuthoredSQL, version.AuthoredDQL, version.GeneratedDQL, version.Notes = authoredSQL.String, authoredDQL.String, generatedDQL.String, notes.String
	version.ComponentSpec = catalogJSON(spec)
	version.DQLExportLimits = catalogJSON(limits)
	version.TypeManifest = catalogJSON(typeManifest)
	version.ResourceManifest = catalogJSON(resourceManifest)
	version.ComponentDescriptor = catalogJSON(descriptor)
	version.CompileDiagnostics = catalogJSON(diagnostics)
	if validatedAt.Valid {
		value := validatedAt.Time
		version.ValidatedAt = &value
	}
	if versionPublishedAt.Valid {
		value := versionPublishedAt.Time
		version.PublishedAt = &value
	}
	return &PublishedReport{Report: &report, Version: &version, Publication: &publication}, nil
}

func catalogJSON(raw sql.NullString) json.RawMessage {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" || raw.String == "null" {
		return nil
	}
	return json.RawMessage(raw.String)
}
