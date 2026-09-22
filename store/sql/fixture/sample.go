package fixture

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// SampleSeed identifies the deterministic rows written by SeedSampleCatalog.
type SampleSeed struct {
	PrimaryConnectorName   string
	SecondaryConnectorName string
	DraftComponentID       string
	DraftSlug              string
	PublishedComponentID   string
	PublishedSlug          string
}

func DefaultSampleSeed() SampleSeed {
	return SampleSeed{
		PrimaryConnectorName: "analytics", SecondaryConnectorName: "warehouse",
		DraftComponentID: "rpt_sales_region", DraftSlug: "sales-by-region",
		PublishedComponentID: "rpt_orders_daily", PublishedSlug: "orders-daily",
	}
}

// SeedSampleCatalog writes a relationally complete fixture for schema/schema.ddl.
func SeedSampleCatalog(ctx context.Context, db *sql.DB, now time.Time) (SampleSeed, error) {
	seed := DefaultSampleSeed()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return seed, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range []struct{ name, description string }{
		{seed.PrimaryConnectorName, "Sample analytics connector"},
		{seed.SecondaryConnectorName, "Sample warehouse connector"},
	} {
		if err := insertConnector(ctx, tx, c.name, c.description, now); err != nil {
			return seed, err
		}
	}
	if err := insertNamespace(ctx, tx, "system", "general", "General", "Default sample namespace", now); err != nil {
		return seed, err
	}
	if err := insertReport(ctx, tx, seed.DraftComponentID, seed.DraftSlug, "Sales By Region", "Draft sample report", seed.PrimaryConnectorName, "reports", "sales_by_region", "draft", 1, now); err != nil {
		return seed, err
	}
	if err := insertVersion(ctx, tx, seed.DraftComponentID, 1, "draft", "sql", "SELECT region, revenue FROM sales", "#set($Limit <int>(query/limit))\nSELECT region, revenue FROM sales", "draft-sales-region", "system", now, nil); err != nil {
		return seed, err
	}
	if err := insertView(ctx, tx, seed.DraftComponentID, 1, "sales_region", "SalesRegion", "root", "table", "sales", seed.PrimaryConnectorName); err != nil {
		return seed, err
	}
	if err := insertField(ctx, tx, seed.DraftComponentID, 1, "sales_region", "region", "region", "string", true, true, true, false); err != nil {
		return seed, err
	}
	if err := insertField(ctx, tx, seed.DraftComponentID, 1, "sales_region", "revenue", "revenue", "float64", false, true, false, true); err != nil {
		return seed, err
	}
	if err := insertParameter(ctx, tx, seed.DraftComponentID, 1, "region", "query", "region", "string", false, 1); err != nil {
		return seed, err
	}
	if err := insertPredicate(ctx, tx, seed.DraftComponentID, 1, "region", 0, "equal", `{"field":"region"}`); err != nil {
		return seed, err
	}
	if err := insertACL(ctx, tx, seed.DraftComponentID, "role", "report_editor", true, true, true, true, true); err != nil {
		return seed, err
	}
	if err := insertACL(ctx, tx, seed.DraftComponentID, "role", "analyst", true, true, false, false, false); err != nil {
		return seed, err
	}

	publishedAt := now.Add(15 * time.Minute)
	if err := insertReport(ctx, tx, seed.PublishedComponentID, seed.PublishedSlug, "Orders Daily", "Published sample report", seed.SecondaryConnectorName, "reports", "orders_daily", "active", 1, now); err != nil {
		return seed, err
	}
	if err := insertVersion(ctx, tx, seed.PublishedComponentID, 1, "published", "dql", "SELECT order_date, order_count FROM daily_orders", "#set($DateFrom <string>(query/dateFrom))\nSELECT order_date, order_count FROM daily_orders", "published-orders-daily", "system", now, &publishedAt); err != nil {
		return seed, err
	}
	if err := insertView(ctx, tx, seed.PublishedComponentID, 1, "orders_daily", "OrdersDaily", "root", "table", "daily_orders", seed.SecondaryConnectorName); err != nil {
		return seed, err
	}
	if err := insertField(ctx, tx, seed.PublishedComponentID, 1, "orders_daily", "order_date", "order_date", "time.Time", true, true, true, false); err != nil {
		return seed, err
	}
	if err := insertField(ctx, tx, seed.PublishedComponentID, 1, "orders_daily", "order_count", "order_count", "int", false, true, false, true); err != nil {
		return seed, err
	}
	if err := insertParameter(ctx, tx, seed.PublishedComponentID, 1, "date_from", "query", "order_date", "time.Time", false, 1); err != nil {
		return seed, err
	}
	if err := insertPredicate(ctx, tx, seed.PublishedComponentID, 1, "date_from", 0, "greater_equal", `{"field":"order_date"}`); err != nil {
		return seed, err
	}
	if err := insertCube(ctx, tx, seed.PublishedComponentID, 1); err != nil {
		return seed, err
	}
	if err := insertMCPExposure(ctx, tx, seed.PublishedComponentID, 1); err != nil {
		return seed, err
	}
	if err := insertACL(ctx, tx, seed.PublishedComponentID, "role", "ops", true, true, false, false, false); err != nil {
		return seed, err
	}
	if err := insertACL(ctx, tx, seed.PublishedComponentID, "role", "report_admin", true, true, true, true, true); err != nil {
		return seed, err
	}
	if err := insertPublication(ctx, tx, seed.PublishedComponentID, 1, 1, "rev-0001", "system", publishedAt); err != nil {
		return seed, err
	}
	if err := tx.Commit(); err != nil {
		return seed, err
	}
	return seed, nil
}

func insertNamespace(ctx context.Context, tx *sql.Tx, ownerID, name, title, description string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,description,status,etag,created_at,updated_at) VALUES(?,?,?,?, 'active',1,?,?)`, ownerID, name, title, description, now, now)
	return err
}

func insertConnector(ctx context.Context, tx *sql.Tx, name, description string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO connectors(name, driver, dsn_template, description, owner_id, status, options_json, last_test_status, last_tested_at, etag, created_at, updated_at) VALUES (?, 'sqlite', ?, ?, 'system', 'active', ?, 'passed', ?, 1, ?, ?)`, name, fixtureConnectorDSN(name), description, `{"busy_timeout_ms":5000}`, now, now, now)
	return err
}

func insertReport(ctx context.Context, tx *sql.Tx, id, slug, title, description, connector, scope, name, status string, draftVersion int, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO reports(id, slug, title, description, owner_id, status, default_connector_name, component_scope, component_name, current_draft_version, etag, created_at, updated_at) VALUES (?, ?, ?, ?, 'system', ?, ?, ?, ?, ?, 1, ?, ?)`, id, slug, title, description, status, connector, scope, name, draftVersion, now, now)
	return err
}

func insertVersion(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, state, mode, authoredSQL, authoredDQL, hash, createdBy string, createdAt time.Time, publishedAt *time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_versions(report_id, version_no, state, authoring_mode, authored_sql, authored_dql, component_spec_json, spec_format_version, spec_hash, generated_dql, type_manifest_json, compile_status, datly_version, compiler_version, source_revision, notes, created_by, created_at, validated_at, published_at) VALUES (?, ?, ?, ?, ?, ?, '{}', 'studio.v1', ?, ?, '{}', 'valid', 'v1', 'studio.v1', 1, ?, ?, ?, ?, ?)`, reportID, versionNo, state, mode, authoredSQL, authoredDQL, hash, authoredDQL, "sample", createdBy, createdAt, publishedAt, publishedAt)
	return err
}

func insertView(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, viewID, name, role, sourceKind, sourceTable, connector string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_views(report_id, version_no, view_id, view_identity, name, role, connector_name, source_kind, source_table) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, reportID, versionNo, viewID, reportID+":"+viewID, name, role, connector, sourceKind, sourceTable)
	return err
}

func insertField(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, viewID, field, source, goType string, filterable, orderable, groupable, measurable bool) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_fields(report_id, version_no, view_id, field_name, source_column, go_type, nullable, ordinal, filterable, orderable, groupable, measurable, metadata_json) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, '{}')`, reportID, versionNo, viewID, field, source, goType, ordinal(field), boolToInt(filterable), boolToInt(orderable), boolToInt(groupable), boolToInt(measurable))
	return err
}

func insertParameter(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, name, sourceKind, sourceName, typeExpr string, required bool, ordinalNo int) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_parameters(report_id, version_no, parameter_id, parameter_identity, name, source_kind, source_name, type_expr, required, ordinal, metadata_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}')`, reportID, versionNo, name, reportID+":"+name, name, sourceKind, sourceName, typeExpr, boolToInt(required), ordinalNo)
	return err
}

func insertPredicate(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, parameter string, indexNo int, name, args string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_predicates(report_id, version_no, parameter_id, predicate_index, name, args_json) VALUES (?, ?, ?, ?, ?, ?)`, reportID, versionNo, parameter, indexNo, name, args)
	return err
}

func insertCube(ctx context.Context, tx *sql.Tx, reportID string, versionNo int) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_cube_configs(report_id, version_no, cube_enabled, dimensions_json, measures_json, compose_enabled) VALUES (?, ?, 1, '["order_date"]', '["order_count"]', 1)`, reportID, versionNo)
	return err
}

func insertMCPExposure(ctx context.Context, tx *sql.Tx, reportID string, versionNo int) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_mcp_exposures(report_id, version_no, exposure_id, route_id, route_method, route_path, kind, name, enabled, ordinal) VALUES (?, ?, 'exposure-orders-daily', 'route-orders-daily', 'GET', '/v1/studio/reports/orders-daily', 'tool', 'orders_daily', 1, 0)`, reportID, versionNo)
	return err
}

func insertPublication(ctx context.Context, tx *sql.Tx, reportID string, versionNo int, generation int64, revision, publishedBy string, publishedAt time.Time) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no, source_revision, status, report_count, build_manifest_json, requested_by, requested_at, activated_at) VALUES (?, ?, 'active', 1, '{}', ?, ?, ?)`, generation, revision, publishedBy, publishedAt, publishedAt); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO report_publications(report_id, active_version_no, desired_version_no, desired_generation, active_generation, publication_status, runtime_revision, spec_hash, published_by, published_at, activated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)`, reportID, versionNo, versionNo, generation, generation, revision, "published-orders-daily", publishedBy, publishedAt, publishedAt)
	return err
}

func insertACL(ctx context.Context, tx *sql.Tx, reportID, subjectType, subjectID string, view, run, edit, publish, dql bool) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO report_acl(report_id, subject_type, subject_id, can_view, can_run, can_edit, can_publish, can_use_dql) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, reportID, subjectType, subjectID, boolToInt(view), boolToInt(run), boolToInt(edit), boolToInt(publish), boolToInt(dql))
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func ordinal(field string) int {
	if strings.Contains(field, "date") || field == "region" {
		return 1
	}
	return 2
}
func fixtureConnectorDSN(name string) string {
	return "file:" + strings.ReplaceAll(name, " ", "_") + "?mode=memory&cache=shared"
}
