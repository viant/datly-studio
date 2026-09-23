// Package sqltransport is the canonical SQLite/MySQL-backed SDK transport.
// It speaks SDK DTOs only; Forge does not need to know about Studio control
// services or the generated component implementation.
package sqltransport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/authoring/readerbuilder"
	datlyreport "github.com/viant/datly/report"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/dql"
)

type PreviewExecutor interface {
	Execute(context.Context, string, int, sdk.PreviewInput) (*sdk.PreviewResult, error)
}

const previewExecutionTimeout = 30 * time.Second

type ViewTester interface {
	TestView(context.Context, string, int, string, sdk.ViewTestInput) (*sdk.ViewTestResult, error)
}

type RelationTester interface {
	TestRelation(context.Context, string, int, string, sdk.ViewTestInput) (*sdk.RelationTestResult, error)
}

type CubeComposeTester interface {
	TestCompose(context.Context, string, int, sdk.CubeComposeTestInput) (*sdk.CubeComposeTestResult, error)
}

type WarmupExecutor interface {
	Warmup(context.Context, string, int) (*sdk.WarmupResult, error)
}

type VersionValidator interface {
	Validate(context.Context, string, int) error
}

type RuntimeActivator interface {
	Reload(context.Context, int64) error
}

type RuntimeActivatorFunc func(context.Context, int64) error

func (f RuntimeActivatorFunc) Reload(ctx context.Context, generation int64) error {
	return f(ctx, generation)
}

type RuntimeHostProbe interface {
	ProbeRuntime(context.Context) (*sdk.RuntimeHost, error)
}

type RuntimeHostProbeFunc func(context.Context) (*sdk.RuntimeHost, error)

func (f RuntimeHostProbeFunc) ProbeRuntime(ctx context.Context) (*sdk.RuntimeHost, error) {
	return f(ctx)
}

// AuthorizationRequest describes one SDK operation without leaking storage or
// generated-component types to the authorization implementation.
type AuthorizationRequest struct {
	Operation     string
	ReportID      string
	ConnectorName string
	NamespaceName string
	OwnerID       string
	Permission    string
}

// Authorizer is mandatory for the in-process SQL transport. HTTP transports
// can adapt their verified request principal to this same contract.
type Authorizer interface {
	Authorize(context.Context, AuthorizationRequest) error
}

type AuthorizerFunc func(context.Context, AuthorizationRequest) error

func (f AuthorizerFunc) Authorize(ctx context.Context, request AuthorizationRequest) error {
	return f(ctx, request)
}

type Transport struct {
	DB             *sql.DB
	Now            func() time.Time
	Preview        PreviewExecutor
	ViewTester     ViewTester
	RelationTester RelationTester
	ComposeTester  CubeComposeTester
	Warmup         WarmupExecutor
	Validator      VersionValidator
	Activator      RuntimeActivator
	RuntimeProbe   RuntimeHostProbe
	Authorizer     Authorizer
	Probe          sdk.ConnectorProbe
	Catalog        sdk.CatalogExplorer
	SQLTester      sdk.ConnectorSQLTester
	Predicates     *predicatecatalog.Catalog
	publicationMu  sync.Mutex
	warmupMu       sync.Mutex
}

func (t *Transport) Invoke(ctx context.Context, operation string, input, output any) error {
	if t == nil || t.DB == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Studio database is unavailable"}
	}
	if err := t.authorize(ctx, operation, input); err != nil {
		return err
	}
	switch operation {
	case sdk.OperationVersionLoadDQL, sdk.OperationVersionLoadArchive:
		return t.loadDQL(ctx, operation, input, output)
	case sdk.OperationVersionDownload:
		return t.downloadComponent(ctx, input, output)
	case sdk.OperationConnectorCreate:
		return t.createConnector(ctx, input, output)
	case sdk.OperationConnectorGet:
		return t.getConnector(ctx, input, output)
	case sdk.OperationConnectorList:
		return t.listConnectors(ctx, input, output)
	case sdk.OperationConnectorUpdate:
		return t.updateConnector(ctx, input, output)
	case sdk.OperationConnectorActivate, sdk.OperationConnectorDisable:
		return t.setConnectorStatus(ctx, operation, input, output)
	case sdk.OperationConnectorDelete:
		return t.deleteConnector(ctx, input)
	case sdk.OperationConnectorTest:
		return t.testConnector(ctx, input, output)
	case sdk.OperationConnectorSchemas, sdk.OperationConnectorTables, sdk.OperationConnectorTable, sdk.OperationConnectorTestSQL:
		return t.connectorCatalog(ctx, operation, input, output)
	case sdk.OperationNamespaceCreate:
		return t.createNamespace(ctx, input, output)
	case sdk.OperationNamespaceGet:
		return t.getNamespace(ctx, input, output)
	case sdk.OperationNamespaceList:
		return t.listNamespaces(ctx, input, output)
	case sdk.OperationNamespaceUpdate:
		return t.updateNamespace(ctx, input, output)
	case sdk.OperationNamespaceDelete:
		return t.deleteNamespace(ctx, input)
	case sdk.OperationAuthorizationPredicateCreate:
		return t.createAuthorizationPredicate(ctx, input, output)
	case sdk.OperationAuthorizationPredicateGet:
		return t.getAuthorizationPredicate(ctx, input, output)
	case sdk.OperationAuthorizationPredicateList:
		return t.listAuthorizationPredicates(ctx, input, output)
	case sdk.OperationAuthorizationPredicateUpdate:
		return t.updateAuthorizationPredicate(ctx, input, output)
	case sdk.OperationAuthorizationPredicateDelete:
		return t.deleteAuthorizationPredicate(ctx, input)
	case sdk.OperationAuthorizationPredicateTypes:
		return t.authorizationPredicateTypes(output)
	case sdk.OperationReportCreate:
		return t.createReport(ctx, input, output)
	case sdk.OperationReportGet:
		return t.getReport(ctx, input, output)
	case sdk.OperationReportList:
		return t.listReports(ctx, input, output)
	case sdk.OperationReportUpdate:
		return t.updateReport(ctx, input, output)
	case sdk.OperationVersionCreate:
		return t.createVersion(ctx, input, output)
	case sdk.OperationVersionGet:
		return t.getVersion(ctx, input, output)
	case sdk.OperationVersionList:
		return t.listVersions(ctx, input, output)
	case sdk.OperationVersionApply:
		return t.applyVersionEdit(ctx, input, output)
	case sdk.OperationVersionValidate:
		return t.validateVersion(ctx, input, output)
	case sdk.OperationVersionExportDQL:
		return t.exportDQL(ctx, input, output)
	case sdk.OperationVersionInspect:
		return t.inspectVersion(ctx, input, output)
	case sdk.OperationVersionBuilder:
		return t.applyReaderBuilder(ctx, input, output)
	case sdk.OperationVersionTestView:
		return t.testVersionView(ctx, input, output)
	case sdk.OperationVersionTestRelation:
		return t.testVersionRelation(ctx, input, output)
	case sdk.OperationVersionTestCompose:
		return t.testVersionCompose(ctx, input, output)
	case sdk.OperationVersionWarmup:
		return t.warmupVersion(ctx, input, output)
	case sdk.OperationVersionWarmupGet:
		return t.getWarmupRun(ctx, input, output)
	case sdk.OperationVersionWarmupList:
		return t.listWarmupRuns(ctx, input, output)
	case sdk.OperationVersionDescriptor:
		return t.versionDescriptor(ctx, input, output)
	case sdk.OperationPreviewExecute:
		return t.preview(ctx, input, output)
	case sdk.OperationPublicationPublish:
		return t.publish(ctx, input, output, "publish")
	case sdk.OperationPublicationGet:
		var in struct {
			ReportID string `json:"reportId"`
		}
		if err := decode(input, &in); err != nil {
			return invalid(err)
		}
		return t.getPublication(ctx, in.ReportID, output)
	case sdk.OperationPublicationRemove:
		return t.unpublish(ctx, input, output)
	case sdk.OperationPublicationRollback:
		return t.publish(ctx, input, output, "rollback")
	case sdk.OperationPublicationEventsList:
		return t.listPublicationEvents(ctx, input, output)
	case sdk.OperationRuntimeStatus:
		return t.runtimeStatus(ctx, output)
	case sdk.OperationResourcesGet, sdk.OperationResourcesUpsertFile, sdk.OperationResourcesDeleteFile,
		sdk.OperationResourcesUpsertFolder, sdk.OperationResourcesDeleteFolder, sdk.OperationResourcesUpsertSkill, sdk.OperationResourcesDeleteSkill:
		return t.resources(ctx, operation, input, output)
	case sdk.OperationACLList, sdk.OperationACLUpsert, sdk.OperationACLDelete:
		return t.acl(ctx, operation, input, output)
	default:
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "unsupported SDK operation", Field: "operation"}
	}
}

func (t *Transport) connectorCatalog(ctx context.Context, operation string, input, output any) error {
	if operation != sdk.OperationConnectorTestSQL && t.Catalog == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "connector catalog explorer is not configured"}
	}
	if operation == sdk.OperationConnectorTestSQL && t.SQLTester == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "connector SQL tester is not configured"}
	}
	var envelope struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := decode(input, &envelope); err != nil {
		return invalid(err)
	}
	connector, err := t.getConnectorValue(ctx, envelope.Name)
	if err != nil {
		return err
	}
	if connector.Status != "active" {
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "connector must be active before browsing schema"}
	}
	switch operation {
	case sdk.OperationConnectorTestSQL:
		var request sdk.SQLTestInput
		if err = decode(envelope.Input, &request); err != nil {
			return invalid(err)
		}
		result, testErr := t.SQLTester.TestSQL(ctx, &connector.Connector, request)
		if testErr != nil {
			return testErr
		}
		return assign(output, result)
	case sdk.OperationConnectorSchemas:
		var request sdk.SchemaCatalogInput
		if err = decode(envelope.Input, &request); err != nil {
			return invalid(err)
		}
		result, exploreErr := t.Catalog.Schemas(ctx, &connector.Connector, request)
		if exploreErr != nil {
			return internal(exploreErr)
		}
		return assign(output, result)
	case sdk.OperationConnectorTables:
		var request sdk.TableCatalogInput
		if err = decode(envelope.Input, &request); err != nil {
			return invalid(err)
		}
		result, exploreErr := t.Catalog.Tables(ctx, &connector.Connector, request)
		if exploreErr != nil {
			return internal(exploreErr)
		}
		return assign(output, result)
	case sdk.OperationConnectorTable:
		var request sdk.TableDetailInput
		if err = decode(envelope.Input, &request); err != nil {
			return invalid(err)
		}
		result, exploreErr := t.Catalog.Table(ctx, &connector.Connector, request)
		if exploreErr != nil {
			return internal(exploreErr)
		}
		return assign(output, result)
	}
	return invalid(errors.New("unsupported connector catalog operation"))
}

func (t *Transport) authorize(ctx context.Context, operation string, input any) error {
	if t.Authorizer == nil {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio authorization is required"}
	}
	var identity struct {
		ReportID string `json:"reportId"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		OwnerID  string `json:"ownerId"`
	}
	if input != nil {
		if err := decode(input, &identity); err != nil {
			return invalid(err)
		}
	}
	permission := "view"
	switch operation {
	case sdk.OperationVersionLoadDQL, sdk.OperationVersionLoadArchive:
		permission = "dql"
	case sdk.OperationConnectorCreate, sdk.OperationConnectorUpdate, sdk.OperationConnectorActivate, sdk.OperationConnectorDisable, sdk.OperationConnectorDelete,
		sdk.OperationNamespaceCreate, sdk.OperationNamespaceUpdate, sdk.OperationNamespaceDelete,
		sdk.OperationReportCreate, sdk.OperationReportUpdate, sdk.OperationVersionCreate, sdk.OperationVersionApply, sdk.OperationVersionValidate, sdk.OperationVersionBuilder,
		sdk.OperationResourcesUpsertFile, sdk.OperationResourcesDeleteFile, sdk.OperationResourcesUpsertFolder, sdk.OperationResourcesDeleteFolder, sdk.OperationResourcesUpsertSkill, sdk.OperationResourcesDeleteSkill:
		permission = "edit"
	case sdk.OperationPreviewExecute, sdk.OperationVersionTestView, sdk.OperationVersionTestRelation, sdk.OperationVersionTestCompose:
		permission = "run"
	case sdk.OperationVersionExportDQL, sdk.OperationVersionDownload:
		permission = "dql"
	case sdk.OperationACLUpsert, sdk.OperationACLDelete,
		sdk.OperationAuthorizationPredicateCreate, sdk.OperationAuthorizationPredicateGet, sdk.OperationAuthorizationPredicateList, sdk.OperationAuthorizationPredicateUpdate, sdk.OperationAuthorizationPredicateDelete, sdk.OperationAuthorizationPredicateTypes:
		permission = "publish"
	case sdk.OperationPublicationPublish, sdk.OperationPublicationRemove, sdk.OperationPublicationRollback, sdk.OperationPublicationEventsList, sdk.OperationRuntimeStatus, sdk.OperationVersionWarmup, sdk.OperationVersionWarmupGet, sdk.OperationVersionWarmupList:
		permission = "publish"
	}
	reportID := identity.ReportID
	if reportID == "" && strings.HasPrefix(operation, "reports.") {
		reportID = identity.ID
	}
	connectorName := identity.Name
	namespaceName := ""
	if strings.HasPrefix(operation, "namespaces.") {
		namespaceName, connectorName = identity.Name, ""
	}
	if operation == sdk.OperationConnectorCreate {
		connectorName = ""
	}
	if operation == sdk.OperationReportCreate {
		reportID = ""
	}
	if err := t.Authorizer.Authorize(ctx, AuthorizationRequest{Operation: operation, ReportID: reportID, ConnectorName: connectorName, NamespaceName: namespaceName, OwnerID: identity.OwnerID, Permission: permission}); err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) {
			return err
		}
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio authorization denied", Cause: err}
	}
	return nil
}

func (t *Transport) now() time.Time {
	if t.Now != nil {
		return t.Now().UTC()
	}
	return time.Now().UTC()
}

func decode(input, target any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

type connectorCreate struct {
	Name, Driver, DSNTemplate, SecretRef, Description, OwnerID string
	Options                                                    json.RawMessage
}

func (t *Transport) createConnector(ctx context.Context, input, output any) error {
	var in connectorCreate
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Driver) == "" {
		return invalid(errors.New("name and driver are required"))
	}
	if strings.TrimSpace(in.OwnerID) == "" {
		principal, ok := sdk.PrincipalFromContext(ctx)
		if !ok {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
		}
		in.OwnerID = principal.Subject
	}
	now := t.now()
	options := string(in.Options)
	if options == "" {
		options = "{}"
	}
	_, err := t.DB.ExecContext(ctx, `INSERT INTO connectors(name, driver, dsn_template, secret_ref, description, owner_id, status, options_json, etag, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'draft', ?, 1, ?, ?)`, in.Name, in.Driver, nullable(in.DSNTemplate), nullable(in.SecretRef), nullable(in.Description), in.OwnerID, options, now, now)
	if err != nil {
		return classify(err, "connector", in.Name)
	}
	return t.getConnector(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) getConnector(ctx context.Context, input, output any) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	query := `SELECT name, driver, dsn_template, secret_ref, description, owner_id, status, options_json, last_test_status, last_test_error_code, last_tested_at, etag, created_at, updated_at FROM connectors WHERE name = ? AND deleted_at IS NULL`
	args := []any{in.Name}
	query, args = connectorReadScope(ctx, query, args)
	row := t.DB.QueryRowContext(ctx, query, args...)
	value, err := scanConnector(row)
	if err != nil {
		return mapReadError(err, "connector", in.Name)
	}
	return assign(output, value)
}

type connectorList struct {
	Query, Status, OwnerID, Driver string
	Limit, Offset                  int
}

func (t *Transport) listConnectors(ctx context.Context, input, output any) error {
	var in connectorList
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	query := `SELECT name, driver, dsn_template, secret_ref, description, owner_id, status, options_json, last_test_status, last_test_error_code, last_tested_at, etag, created_at, updated_at FROM connectors WHERE deleted_at IS NULL`
	args := []any{}
	if q := strings.TrimSpace(in.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query += ` AND (LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(driver) LIKE ? OR LOWER(owner_id) LIKE ?)`
		args = append(args, like, like, like, like)
	}
	if in.Status != "" {
		query += ` AND status = ?`
		args = append(args, in.Status)
	}
	if in.OwnerID != "" {
		query += ` AND owner_id = ?`
		args = append(args, in.OwnerID)
	}
	if in.Driver != "" {
		query += ` AND driver = ?`
		args = append(args, in.Driver)
	}
	query, args = connectorReadScope(ctx, query, args)
	query += ` ORDER BY updated_at DESC, name ASC LIMIT ? OFFSET ?`
	args = append(args, limit, in.Offset)
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.ConnectorPage{Limit: limit, Offset: in.Offset}
	for rows.Next() {
		value, err := scanConnector(rows)
		if err != nil {
			return internal(err)
		}
		page.Items = append(page.Items, value)
	}
	if err := rows.Err(); err != nil {
		return internal(err)
	}
	return assign(output, page)
}

type connectorUpdate struct {
	Name  string `json:"name"`
	Input struct {
		Driver, DSNTemplate, SecretRef, Description *string
		Options                                     *json.RawMessage
		ETag                                        int64
	} `json:"input"`
}

func (t *Transport) updateConnector(ctx context.Context, input, output any) error {
	var in connectorUpdate
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if in.Input.ETag <= 0 {
		return invalid(errors.New("etag is required"))
	}
	current, err := t.getConnectorValue(ctx, in.Name)
	if err != nil {
		return err
	}
	connectionChanged := false
	if in.Input.Driver != nil {
		connectionChanged = connectionChanged || *in.Input.Driver != current.Driver
		current.Driver = *in.Input.Driver
	}
	if in.Input.DSNTemplate != nil {
		connectionChanged = connectionChanged || *in.Input.DSNTemplate != current.DSNTemplate
		current.DSNTemplate = *in.Input.DSNTemplate
	}
	if in.Input.SecretRef != nil {
		connectionChanged = connectionChanged || *in.Input.SecretRef != current.SecretRef
		current.SecretRef = *in.Input.SecretRef
	}
	if in.Input.Description != nil {
		current.Description = *in.Input.Description
	}
	options := current.Options
	if in.Input.Options != nil {
		connectionChanged = connectionChanged || !bytes.Equal(*in.Input.Options, options)
		options = *in.Input.Options
	}
	status := current.Status
	if connectionChanged {
		status = "draft"
	}
	now := t.now()
	result, err := t.DB.ExecContext(ctx, `UPDATE connectors SET driver=?, dsn_template=?, secret_ref=?, description=?, options_json=?, status=?, last_test_status=CASE WHEN ? THEN NULL ELSE last_test_status END, last_test_error_code=CASE WHEN ? THEN NULL ELSE last_test_error_code END, last_tested_at=CASE WHEN ? THEN NULL ELSE last_tested_at END, etag=etag+1, updated_at=? WHERE name=? AND etag=? AND deleted_at IS NULL`, current.Driver, nullable(current.DSNTemplate), nullable(current.SecretRef), nullable(current.Description), string(options), status, connectionChanged, connectionChanged, connectionChanged, now, in.Name, in.Input.ETag)
	if err != nil {
		return internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector etag does not match"}
	}
	return t.getConnector(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) setConnectorStatus(ctx context.Context, operation string, input, output any) error {
	var in struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	status := "active"
	if operation == sdk.OperationConnectorDisable {
		status = "disabled"
	} else {
		current, err := t.getConnectorValue(ctx, in.Name)
		if err != nil {
			return err
		}
		if current.LastTestStatus != "passed" {
			return invalid(errors.New("connector must pass a connectivity test before activation"))
		}
	}
	result, err := t.DB.ExecContext(ctx, `UPDATE connectors SET status=?, etag=etag+1, updated_at=? WHERE name=? AND etag=? AND deleted_at IS NULL`, status, t.now(), in.Name, in.ETag)
	if err != nil {
		return internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector etag does not match"}
	}
	return t.getConnector(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) deleteConnector(ctx context.Context, input any) error {
	var in struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	var used int
	if err := t.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM reports WHERE default_connector_name=? AND deleted_at IS NULL`, in.Name).Scan(&used); err != nil {
		return internal(err)
	}
	if used > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector is referenced by a report"}
	}
	result, err := t.DB.ExecContext(ctx, `UPDATE connectors SET status='deleted', deleted_at=?, updated_at=?, etag=etag+1 WHERE name=? AND etag=? AND deleted_at IS NULL`, t.now(), t.now(), in.Name, in.ETag)
	if err != nil {
		return internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "connector not found"}
	}
	return nil
}

func (t *Transport) testConnector(ctx context.Context, input, output any) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getConnectorValue(ctx, in.Name)
	if err != nil {
		return err
	}
	if t.Probe == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "connector probe is not configured"}
	}
	result, probeErr := t.Probe.Probe(ctx, &value.Connector)
	if result == nil {
		result = &sdk.ConnectorTestResult{}
	}
	result.Name, result.TestedAt = value.Name, t.now()
	if probeErr != nil {
		result.Status = "failed"
		if result.Message == "" {
			result.Message = probeErr.Error()
		}
		if result.ErrorCode == "" {
			result.ErrorCode = "connectivity_failed"
		}
	} else if result.Status == "" {
		result.Status = "passed"
	}
	_, err = t.DB.ExecContext(ctx, `UPDATE connectors SET last_test_status=?,last_test_error_code=?,last_tested_at=?,updated_at=? WHERE name=? AND deleted_at IS NULL`, result.Status, nullable(result.ErrorCode), result.TestedAt, t.now(), value.Name)
	if err != nil {
		return internal(err)
	}
	if probeErr != nil {
		return assign(output, result)
	}
	return assign(output, result)
}

type connectorValue struct{ sdk.Connector }

func (t *Transport) getConnectorValue(ctx context.Context, name string) (*connectorValue, error) {
	var out sdk.Connector
	query := `SELECT name, driver, dsn_template, secret_ref, description, owner_id, status, options_json, last_test_status, last_test_error_code, last_tested_at, etag, created_at, updated_at FROM connectors WHERE name=? AND deleted_at IS NULL`
	query, args := connectorReadScope(ctx, query, []any{name})
	_, err := scanConnector(t.DB.QueryRowContext(ctx, query, args...), &out)
	if err != nil {
		return nil, mapReadError(err, "connector", name)
	}
	return &connectorValue{out}, nil
}
func scanConnector(scanner interface{ Scan(...any) error }, outputs ...*sdk.Connector) (*sdk.Connector, error) {
	var c sdk.Connector
	var dsn, secret, desc, options, test, code sql.NullString
	var tested sql.NullTime
	if err := scanner.Scan(&c.Name, &c.Driver, &dsn, &secret, &desc, &c.OwnerID, &c.Status, &options, &test, &code, &tested, &c.ETag, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.DSNTemplate, c.DSNConfigured, c.SecretRef, c.SecretConfigured, c.Description, c.LastTestStatus, c.LastTestErrorCode = dsn.String, strings.TrimSpace(dsn.String) != "", secret.String, strings.TrimSpace(secret.String) != "", desc.String, test.String, code.String
	if options.Valid {
		c.Options = json.RawMessage(options.String)
	}
	if tested.Valid {
		value := tested.Time
		c.LastTestedAt = &value
	}
	if len(outputs) > 0 {
		*outputs[0] = c
	}
	return &c, nil
}

type reportCreate struct{ ID, Namespace, Slug, Title, Description, OwnerID, DefaultConnectorName, ComponentScope, ComponentName string }

func (t *Transport) createReport(ctx context.Context, input, output any) error {
	var in reportCreate
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if !validReportSlug(in.Slug) || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.DefaultConnectorName) == "" {
		return invalid(errors.New("slug, title, and defaultConnectorName are required; slug must use lowercase letters, numbers, and dashes"))
	}
	if strings.TrimSpace(in.Namespace) == "" {
		in.Namespace = "general"
	}
	if !validBusinessNamespace(in.Namespace) {
		return invalid(errors.New("namespace must use lowercase letters, numbers, underscores, and optional dot-separated segments"))
	}
	principal, hasPrincipal := sdk.PrincipalFromContext(ctx)
	if strings.TrimSpace(in.OwnerID) == "" {
		if !hasPrincipal {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
		}
		in.OwnerID = principal.Subject
	}
	if hasPrincipal && in.OwnerID != principal.Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "cannot create another principal's report"}
	}
	if strings.TrimSpace(in.ID) == "" {
		var err error
		in.ID, err = generatedReportID()
		if err != nil {
			return internal(err)
		}
	}
	if hasPrincipal || strings.TrimSpace(in.ComponentScope) == "" {
		in.ComponentScope = dynamicComponentScope(in.OwnerID, in.ID)
	}
	if hasPrincipal || strings.TrimSpace(in.ComponentName) == "" {
		in.ComponentName = "reader"
	}
	connector, err := t.getConnectorValue(ctx, in.DefaultConnectorName)
	if err != nil {
		return err
	}
	if connector.Status != "active" {
		return invalid(errors.New("default connector must be active"))
	}
	now := t.now()
	if err = t.ensureReportNamespace(ctx, in.OwnerID, in.Namespace, now); err != nil {
		return err
	}
	_, err = t.DB.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,description,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,?,?,?,'draft',?,?,?,1,?,?)`, in.ID, in.Namespace, in.Slug, in.Title, nullable(in.Description), in.OwnerID, in.DefaultConnectorName, in.ComponentScope, in.ComponentName, now, now)
	if err != nil {
		return classify(err, "report", in.ID)
	}
	return t.getReport(ctx, struct {
		ID string `json:"id"`
	}{in.ID}, output)
}

func validBusinessNamespace(value string) bool {
	segments := strings.Split(strings.TrimSpace(value), ".")
	for _, segment := range segments {
		if segment == "" || len(segment) > 64 || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for _, character := range segment[1:] {
			if character != '_' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return false
			}
		}
	}
	return len(value) <= 200
}

func dynamicComponentScope(ownerID, reportID string) string {
	return "github.com/viant/datly-studio/dynamic/" + sdk.OwnerPackageSegment(ownerID) + "/" + strings.TrimSpace(reportID)
}

func validReportSlug(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 200 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func generatedReportID() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "r" + hex.EncodeToString(data), nil
}
func (t *Transport) getReport(ctx context.Context, input, output any) error {
	var in struct {
		ID string `json:"id"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getReportValue(ctx, in.ID)
	if err != nil {
		return err
	}
	return assign(output, value)
}

type reportList struct {
	Query, Namespace, Status, OwnerID, ConnectorName string
	Limit, Offset                                    int
}

func (t *Transport) listReports(ctx context.Context, input, output any) error {
	var in reportList
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	q := `SELECT id,namespace,slug,title,description,owner_id,status,default_connector_name,component_scope,component_name,current_draft_version,etag,created_at,updated_at FROM reports WHERE deleted_at IS NULL`
	args := []any{}
	if in.Query != "" {
		like := "%" + strings.ToLower(in.Query) + "%"
		q += ` AND (LOWER(namespace) LIKE ? OR LOWER(slug) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?)`
		args = append(args, like, like, like, like)
	}
	if in.Namespace != "" {
		q += ` AND namespace=?`
		args = append(args, in.Namespace)
	}
	if in.Status != "" {
		q += ` AND status=?`
		args = append(args, in.Status)
	}
	if in.OwnerID != "" {
		q += ` AND owner_id=?`
		args = append(args, in.OwnerID)
	}
	if in.ConnectorName != "" {
		q += ` AND default_connector_name=?`
		args = append(args, in.ConnectorName)
	}
	q, args = reportReadScope(ctx, q, args)
	q += ` ORDER BY updated_at DESC,id ASC LIMIT ? OFFSET ?`
	args = append(args, limit, in.Offset)
	rows, err := t.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.ReportPage{Limit: limit, Offset: in.Offset}
	for rows.Next() {
		v, err := scanReport(rows)
		if err != nil {
			return internal(err)
		}
		page.Items = append(page.Items, v)
	}
	return assign(output, page)
}

type reportUpdate struct {
	ID    string `json:"id"`
	Input struct {
		Namespace, Slug, Title, Description, Status, DefaultConnectorName, ComponentScope, ComponentName *string
		CurrentDraftVersion                                                                              *int
		ETag                                                                                             int64
	} `json:"input"`
}

func (t *Transport) updateReport(ctx context.Context, input, output any) error {
	var in reportUpdate
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if in.Input.ETag <= 0 {
		return invalid(errors.New("etag is required"))
	}
	current, err := t.getReportValue(ctx, in.ID)
	if err != nil {
		return err
	}
	if _, hasPrincipal := sdk.PrincipalFromContext(ctx); hasPrincipal && (in.Input.ComponentScope != nil || in.Input.ComponentName != nil) {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "component package identity is server managed"}
	}
	if in.Input.Slug != nil {
		current.Slug = *in.Input.Slug
	}
	if in.Input.Namespace != nil {
		if !validBusinessNamespace(*in.Input.Namespace) {
			return invalid(errors.New("namespace must use lowercase letters, numbers, underscores, and optional dot-separated segments"))
		}
		current.Namespace = *in.Input.Namespace
	}
	if in.Input.Title != nil {
		current.Title = *in.Input.Title
	}
	if in.Input.Description != nil {
		current.Description = *in.Input.Description
	}
	if in.Input.Status != nil {
		current.Status = *in.Input.Status
	}
	if in.Input.DefaultConnectorName != nil {
		requested := strings.TrimSpace(*in.Input.DefaultConnectorName)
		if requested == "" {
			return invalid(errors.New("default connector name is required"))
		}
		if requested != current.DefaultConnectorName {
			connector, connectorErr := t.getConnectorValue(ctx, requested)
			if connectorErr != nil {
				return connectorErr
			}
			if connector.Status != "active" {
				return invalid(errors.New("default connector must be active"))
			}
			var versioned bool
			if queryErr := t.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM report_versions WHERE report_id=?)`, in.ID).Scan(&versioned); queryErr != nil {
				return internal(queryErr)
			}
			if versioned {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "default connector is part of the versioned reader contract; change it through Reader Builder and create a validated version"}
			}
			current.DefaultConnectorName = requested
		}
	}
	if in.Input.ComponentScope != nil {
		current.ComponentScope = *in.Input.ComponentScope
	}
	if in.Input.ComponentName != nil {
		current.ComponentName = *in.Input.ComponentName
	}
	draft := current.CurrentDraftVersion
	if in.Input.CurrentDraftVersion != nil {
		draft = in.Input.CurrentDraftVersion
	}
	if err = t.requireActiveNamespace(ctx, current.OwnerID, current.Namespace); err != nil {
		return err
	}
	res, err := t.DB.ExecContext(ctx, `UPDATE reports SET namespace=?,slug=?,title=?,description=?,status=?,default_connector_name=?,component_scope=?,component_name=?,current_draft_version=?,etag=etag+1,updated_at=? WHERE id=? AND etag=? AND deleted_at IS NULL`, current.Namespace, current.Slug, current.Title, nullable(current.Description), current.Status, current.DefaultConnectorName, current.ComponentScope, current.ComponentName, draft, t.now(), in.ID, in.Input.ETag)
	if err != nil {
		return internal(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "report etag does not match"}
	}
	return t.getReport(ctx, struct {
		ID string `json:"id"`
	}{in.ID}, output)
}

func (t *Transport) ensureReportNamespace(ctx context.Context, ownerID, name string, now time.Time) error {
	err := t.requireActiveNamespace(ctx, ownerID, name)
	if err == nil {
		return nil
	}
	var sdkErr *sdk.Error
	if name != "general" || !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
		return err
	}
	_, insertErr := t.DB.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,description,status,etag,created_at,updated_at) VALUES(?,'general','General','Default namespace','active',1,?,?)`, ownerID, now, now)
	if insertErr != nil {
		if retryErr := t.requireActiveNamespace(ctx, ownerID, name); retryErr == nil {
			return nil
		}
		return classify(insertErr, "namespace", name)
	}
	return nil
}

func (t *Transport) requireActiveNamespace(ctx context.Context, ownerID, name string) error {
	var status string
	err := t.DB.QueryRowContext(ctx, `SELECT status FROM namespaces WHERE owner_id=? AND name=? AND deleted_at IS NULL`, ownerID, name).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "namespace not found"}
	}
	if err != nil {
		return internal(err)
	}
	if status != "active" {
		return invalid(errors.New("namespace must be active"))
	}
	return nil
}
func (t *Transport) getReportValue(ctx context.Context, id string) (*sdk.Report, error) {
	query := `SELECT id,namespace,slug,title,description,owner_id,status,default_connector_name,component_scope,component_name,current_draft_version,etag,created_at,updated_at FROM reports WHERE id=? AND deleted_at IS NULL`
	query, args := reportReadScope(ctx, query, []any{id})
	value, err := scanReport(t.DB.QueryRowContext(ctx, query, args...))
	if err != nil {
		return nil, mapReadError(err, "report", id)
	}
	return value, nil
}

// reportReadScope mirrors the Datly reader authorization predicate for SDK
// catalog reads. A trusted in-process caller may omit a principal; every
// HTTP gateway request must carry one after authentication.
func reportReadScope(ctx context.Context, query string, args []any) (string, []any) {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return query, args
	}
	query += ` AND (owner_id = ? OR EXISTS (
SELECT 1 FROM report_acl studio_sdk_acl
WHERE studio_sdk_acl.report_id = reports.id
  AND studio_sdk_acl.subject_type = 'user'
  AND studio_sdk_acl.subject_id = ?
  AND studio_sdk_acl.can_view = TRUE))`
	return query, append(args, principal.Subject, principal.Subject)
}

// connectorReadScope grants access to owned connectors and connectors used by
// a report for which the principal has view permission.
func connectorReadScope(ctx context.Context, query string, args []any) (string, []any) {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return query, args
	}
	query += ` AND (owner_id = ? OR EXISTS (
SELECT 1 FROM reports studio_sdk_report
JOIN report_acl studio_sdk_acl ON studio_sdk_acl.report_id = studio_sdk_report.id
WHERE studio_sdk_report.default_connector_name = connectors.name
  AND studio_sdk_report.deleted_at IS NULL
  AND studio_sdk_acl.subject_type = 'user'
  AND studio_sdk_acl.subject_id = ?
  AND studio_sdk_acl.can_view = TRUE))`
	return query, append(args, principal.Subject, principal.Subject)
}
func scanReport(scanner interface{ Scan(...any) error }) (*sdk.Report, error) {
	var r sdk.Report
	var desc sql.NullString
	var draft sql.NullInt64
	if err := scanner.Scan(&r.ID, &r.Namespace, &r.Slug, &r.Title, &desc, &r.OwnerID, &r.Status, &r.DefaultConnectorName, &r.ComponentScope, &r.ComponentName, &draft, &r.ETag, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Description = desc.String
	r.OwnerPackage = sdk.OwnerPackageSegment(r.OwnerID)
	if draft.Valid {
		v := int(draft.Int64)
		r.CurrentDraftVersion = &v
	}
	return &r, nil
}

type versionCreateRequest struct {
	ReportID string                 `json:"reportId"`
	Input    sdk.CreateVersionInput `json:"input"`
}

type versionIdentityRequest struct {
	ReportID  string `json:"reportId"`
	VersionNo int    `json:"versionNo"`
}

func (t *Transport) createVersion(ctx context.Context, input, output any) error {
	var in versionCreateRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" {
		return invalid(errors.New("reportId is required"))
	}
	principal, hasPrincipal := sdk.PrincipalFromContext(ctx)
	if strings.TrimSpace(in.Input.CreatedBy) == "" {
		if !hasPrincipal {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
		}
		in.Input.CreatedBy = principal.Subject
	}
	if hasPrincipal && in.Input.CreatedBy != principal.Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "cannot create a version for another principal"}
	}
	mode := strings.ToLower(strings.TrimSpace(in.Input.AuthoringMode))
	if mode == "" {
		mode = "structured"
	}
	if mode != "sql" && mode != "dql" && mode != "structured" {
		return invalid(errors.New("authoringMode must be sql, dql, or structured"))
	}
	var next int
	if err := t.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_no), 0) + 1 FROM report_versions WHERE report_id=?`, in.ReportID).Scan(&next); err != nil {
		return internal(err)
	}
	spec := in.Input.ComponentSpec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	generated := in.Input.AuthoredDQL
	if generated == "" {
		generated = in.Input.AuthoredSQL
	}
	specHash := hashVersion(in.ReportID, next, mode, in.Input.AuthoredSQL, in.Input.AuthoredDQL, spec)
	now := t.now()
	_, err := t.DB.ExecContext(ctx, `
INSERT INTO report_versions(report_id, version_no, state, authoring_mode, authored_sql, authored_dql,
 component_spec_json, spec_format_version, spec_hash, generated_dql, type_manifest_json,
 compile_status, datly_version, compiler_version, source_revision, notes, created_by, created_at)
VALUES (?, ?, 'draft', ?, ?, ?, ?, 'studio.v1', ?, ?, '{}', 'pending', 'v1', 'studio.v1', 1, ?, ?, ?)`,
		in.ReportID, next, mode, nullable(in.Input.AuthoredSQL), nullable(in.Input.AuthoredDQL), string(spec), specHash,
		nullable(generated), nullable(in.Input.Notes), in.Input.CreatedBy, now)
	if err != nil {
		return classify(err, "report version", fmt.Sprintf("%s/%d", in.ReportID, next))
	}
	return t.getVersion(ctx, versionIdentityRequest{ReportID: in.ReportID, VersionNo: next}, output)
}

func (t *Transport) getVersion(ctx context.Context, input, output any) error {
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	return assign(output, value)
}

type versionListRequest struct {
	ReportID string                `json:"reportId"`
	Input    sdk.ListVersionsInput `json:"input"`
}

func (t *Transport) listVersions(ctx context.Context, input, output any) error {
	var in versionListRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	limit := in.Input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := versionSelectSQL + ` WHERE report_id=?`
	args := []any{in.ReportID}
	if in.Input.State != "" {
		query += ` AND state=?`
		args = append(args, in.Input.State)
	}
	if in.Input.AuthoringMode != "" {
		query += ` AND authoring_mode=?`
		args = append(args, in.Input.AuthoringMode)
	}
	if in.Input.CompileStatus != "" {
		query += ` AND compile_status=?`
		args = append(args, in.Input.CompileStatus)
	}
	if in.Input.CreatedBy != "" {
		query += ` AND created_by=?`
		args = append(args, in.Input.CreatedBy)
	}
	query += ` ORDER BY version_no DESC LIMIT ? OFFSET ?`
	args = append(args, limit, maxZero(in.Input.Offset))
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.VersionPage{Limit: limit, Offset: maxZero(in.Input.Offset)}
	for rows.Next() {
		value, err := scanVersion(rows)
		if err != nil {
			return internal(err)
		}
		page.Items = append(page.Items, value)
	}
	if err := rows.Err(); err != nil {
		return internal(err)
	}
	return assign(output, page)
}

const versionSelectSQL = `SELECT report_id, version_no, state, authoring_mode, authored_sql, authored_dql,
 component_spec_json, spec_format_version, spec_hash, generated_dql, dql_export_limits_json,
 type_manifest_json, resource_manifest_json, component_descriptor_json, compile_status,
 compile_diagnostics_json, datly_version, compiler_version, source_revision, notes, created_by,
 created_at, validated_at, published_at FROM report_versions`

func (t *Transport) getVersionValue(ctx context.Context, reportID string, versionNo int) (*sdk.ReportVersion, error) {
	value, err := scanVersion(t.DB.QueryRowContext(ctx, versionSelectSQL+` WHERE report_id=? AND version_no=?`, reportID, versionNo))
	if err != nil {
		return nil, mapReadError(err, "report version", fmt.Sprintf("%s/%d", reportID, versionNo))
	}
	return value, nil
}

type versionEditRequest struct {
	ReportID  string          `json:"reportId"`
	VersionNo int             `json:"versionNo"`
	Command   sdk.EditCommand `json:"command"`
}

func (t *Transport) applyVersionEdit(ctx context.Context, input, output any) error {
	var in versionEditRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	current, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	if in.Command.ExpectedSourceRevision > 0 && in.Command.ExpectedSourceRevision != current.SourceRevision {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
	}
	if len(in.Command.Payload) == 0 {
		return invalid(errors.New("edit payload is required"))
	}
	spec := current.ComponentSpec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	var authoredSQL, authoredDQL string = current.AuthoredSQL, current.AuthoredDQL
	var generatedDQL = current.GeneratedDQL
	switch strings.ToLower(strings.TrimSpace(in.Command.Kind)) {
	case "replace_spec", "set_spec":
		var candidate any
		if err := json.Unmarshal(in.Command.Payload, &candidate); err != nil {
			return invalid(fmt.Errorf("component spec: %w", err))
		}
		spec = in.Command.Payload
	case "set_dql":
		var payload struct {
			AuthoredDQL  string `json:"authoredDql"`
			GeneratedDQL string `json:"generatedDql"`
		}
		if err := json.Unmarshal(in.Command.Payload, &payload); err != nil {
			return invalid(err)
		}
		if strings.TrimSpace(payload.AuthoredDQL) == "" {
			return invalid(errors.New("authoredDql is required"))
		}
		authoredDQL, generatedDQL = payload.AuthoredDQL, payload.GeneratedDQL
		if generatedDQL == "" {
			generatedDQL = authoredDQL
		}
	case "set_sql":
		var payload struct {
			AuthoredSQL string `json:"authoredSql"`
		}
		if err := json.Unmarshal(in.Command.Payload, &payload); err != nil {
			return invalid(err)
		}
		if strings.TrimSpace(payload.AuthoredSQL) == "" {
			return invalid(errors.New("authoredSql is required"))
		}
		authoredSQL, generatedDQL = payload.AuthoredSQL, payload.AuthoredSQL
	default:
		return invalid(fmt.Errorf("unsupported version edit kind %q", in.Command.Kind))
	}
	nextRevision := current.SourceRevision + 1
	if nextRevision <= 1 && current.SourceRevision == 0 {
		nextRevision = 2
	}
	hash := hashVersion(in.ReportID, in.VersionNo, current.AuthoringMode, authoredSQL, authoredDQL, spec)
	_, err = t.DB.ExecContext(ctx, `UPDATE report_versions SET authored_sql=?, authored_dql=?, component_spec_json=?, spec_hash=?, generated_dql=?, compile_status='pending', compile_diagnostics_json='[]', source_revision=? WHERE report_id=? AND version_no=? AND source_revision=?`, nullable(authoredSQL), nullable(authoredDQL), string(spec), hash, nullable(generatedDQL), nextRevision, in.ReportID, in.VersionNo, current.SourceRevision)
	if err != nil {
		return internal(err)
	}
	updated, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	return assign(output, &sdk.EditResult{Version: updated})
}

func scanVersion(scanner interface{ Scan(...any) error }) (*sdk.ReportVersion, error) {
	var value sdk.ReportVersion
	var authoredSQL, authoredDQL, spec, limits, types, resources, descriptor, diagnostics, generated, notes sql.NullString
	if err := scanner.Scan(&value.ReportID, &value.VersionNo, &value.State, &value.AuthoringMode, &authoredSQL, &authoredDQL,
		&spec, &value.SpecFormatVersion, &value.SpecHash, &generated, &limits, &types, &resources, &descriptor,
		&value.CompileStatus, &diagnostics, &value.DatlyVersion, &value.CompilerVersion, &value.SourceRevision,
		&notes, &value.CreatedBy, &value.CreatedAt, &value.ValidatedAt, &value.PublishedAt); err != nil {
		return nil, err
	}
	value.AuthoredSQL = rawJSONOrString(authoredSQL)
	value.AuthoredDQL = rawJSONOrString(authoredDQL)
	value.GeneratedDQL = rawJSONOrString(generated)
	value.Notes = notes.String
	value.ComponentSpec = rawJSON(spec)
	value.DQLExportLimits = rawJSON(limits)
	value.TypeManifest = rawJSON(types)
	value.ResourceManifest = rawJSON(resources)
	value.ComponentDescriptor = rawJSON(descriptor)
	value.CompileDiagnostics = rawJSON(diagnostics)
	return &value, nil
}

func (t *Transport) validateVersion(ctx context.Context, input, output any) error {
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	valid := strings.TrimSpace(value.AuthoredDQL) != "" || strings.TrimSpace(value.AuthoredSQL) != ""
	var diagnostics []sdk.Diagnostic
	if !valid {
		diagnostics = append(diagnostics, sdk.Diagnostic{Severity: "error", Code: "empty_source", Message: "authored SQL or DQL is required"})
	} else if t.Validator != nil {
		if validateErr := t.Validator.Validate(ctx, in.ReportID, in.VersionNo); validateErr != nil {
			valid = false
			var compileErr *transcribe.CompileError
			if errors.As(validateErr, &compileErr) && len(compileErr.Diagnostics) > 0 {
				for _, item := range compileErr.Diagnostics {
					if item == nil {
						continue
					}
					diagnostics = append(diagnostics, sdk.Diagnostic{Severity: string(item.Severity), Code: item.Code, Message: item.Message, Hint: item.Hint, Line: item.Span.Start.Line, Column: item.Span.Start.Char})
				}
			} else {
				diagnostics = append(diagnostics, sdk.Diagnostic{Severity: "error", Code: "runtime_contract", Message: validateErr.Error()})
			}
		}
	}
	status := "valid"
	if !valid {
		status = "invalid"
	}
	diagnosticJSON, err := json.Marshal(diagnostics)
	if err != nil {
		return internal(err)
	}
	if _, err := t.DB.ExecContext(ctx, `UPDATE report_versions SET compile_status=?, compile_diagnostics_json=?, validated_at=? WHERE report_id=? AND version_no=?`, status, string(diagnosticJSON), t.now(), in.ReportID, in.VersionNo); err != nil {
		return internal(err)
	}
	value.CompileStatus = status
	value.ValidatedAt = ptrTime(t.now())
	result := &sdk.ValidationResult{Valid: valid, Version: value, Diagnostics: diagnostics}
	return assign(output, result)
}

func (t *Transport) exportDQL(ctx context.Context, input, output any) error {
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	dql := value.GeneratedDQL
	if dql == "" {
		dql = value.AuthoredDQL
	}
	if dql == "" {
		dql = value.AuthoredSQL
	}
	return assign(output, &sdk.DQLExport{DQL: dql, Complete: dql != ""})
}

func (t *Transport) versionDescriptor(ctx context.Context, input, output any) error {
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	return assign(output, &sdk.ComponentDescriptor{Component: value.ComponentSpec, Types: value.TypeManifest, Resources: value.ResourceManifest})
}

func (t *Transport) inspectVersion(ctx context.Context, input, output any) error {
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	version, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	response, err := t.runReaderBuilder(ctx, in.ReportID, version, readerbuilder.Operation{Type: readerbuilder.OperationInspect})
	if err != nil {
		return err
	}
	capabilities, err := t.reportCapabilities(ctx, in.ReportID)
	if err != nil {
		return err
	}
	return assign(output, readerInspection(version, response, capabilities))
}

type readerBuilderRequest struct {
	ReportID  string                   `json:"reportId"`
	VersionNo int                      `json:"versionNo"`
	Command   sdk.ReaderBuilderCommand `json:"command"`
}

func (t *Transport) applyReaderBuilder(ctx context.Context, input, output any) error {
	var in readerBuilderRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if len(in.Command.Operation) == 0 {
		return invalid(errors.New("reader builder operation is required"))
	}
	version, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	if in.Command.ExpectedSourceRevision > 0 && in.Command.ExpectedSourceRevision != version.SourceRevision {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
	}
	var operation readerbuilder.Operation
	if err = json.Unmarshal(in.Command.Operation, &operation); err != nil {
		return invalid(fmt.Errorf("reader builder operation: %w", err))
	}
	response, err := t.runReaderBuilder(ctx, in.ReportID, version, operation)
	if err != nil {
		return err
	}
	capabilities, err := t.reportCapabilities(ctx, in.ReportID)
	if err != nil {
		return err
	}
	inspection := readerInspection(version, response, capabilities)
	if !response.Applied || operation.Type == readerbuilder.OperationInspect {
		return assign(output, &sdk.ReaderBuilderResult{Applied: response.Applied, Inspection: inspection})
	}
	if err = t.validateMCPToolNames(ctx, in.ReportID, response.Structure); err != nil {
		return err
	}
	connector := ""
	if response.Structure != nil && response.Structure.Component != nil && response.Structure.Component.Settings != nil {
		connector = strings.TrimSpace(response.Structure.Component.Settings.DefaultConnector)
	}
	editedVersion, err := t.persistReaderBuilderDQL(ctx, in.ReportID, version, response.DQL, connector)
	if err != nil {
		return err
	}
	inspection.Version = editedVersion
	if operation.Type == readerbuilder.OperationSetPackage {
		report, reportErr := t.getReportValue(ctx, in.ReportID)
		if reportErr != nil {
			return reportErr
		}
		desiredScope := dynamicComponentScope(report.OwnerID, report.ID)
		if _, reportErr = t.DB.ExecContext(ctx, `UPDATE reports SET component_scope=?,component_name='reader',etag=etag+1,updated_at=? WHERE id=? AND deleted_at IS NULL`, desiredScope, t.now(), report.ID); reportErr != nil {
			return internal(reportErr)
		}
	}
	return assign(output, &sdk.ReaderBuilderResult{Applied: true, Inspection: inspection})
}

func (t *Transport) persistReaderBuilderDQL(ctx context.Context, reportID string, current *sdk.ReportVersion, dql, connector string) (*sdk.ReportVersion, error) {
	if current == nil || strings.TrimSpace(dql) == "" {
		return nil, invalid(errors.New("reader builder DQL is required"))
	}
	nextRevision := current.SourceRevision + 1
	if nextRevision <= 1 && current.SourceRevision == 0 {
		nextRevision = 2
	}
	spec := current.ComponentSpec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	hash := hashVersion(reportID, current.VersionNo, current.AuthoringMode, current.AuthoredSQL, dql, spec)
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, internal(err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE report_versions SET authored_dql=?, component_spec_json=?, spec_hash=?, generated_dql=?, compile_status='pending', compile_diagnostics_json='[]', source_revision=? WHERE report_id=? AND version_no=? AND source_revision=?`, nullable(dql), string(spec), hash, nullable(dql), nextRevision, reportID, current.VersionNo, current.SourceRevision)
	if err != nil {
		return nil, internal(err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
	}
	if connector != "" {
		if _, err = tx.ExecContext(ctx, `UPDATE reports SET default_connector_name=?,etag=etag+CASE WHEN default_connector_name<>? THEN 1 ELSE 0 END,updated_at=? WHERE id=? AND deleted_at IS NULL`, connector, connector, t.now(), reportID); err != nil {
			return nil, internal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, internal(err)
	}
	return t.getVersionValue(ctx, reportID, current.VersionNo)
}

func (t *Transport) validateMCPToolNames(ctx context.Context, reportID string, structure *readerbuilder.Structure) error {
	report, err := t.getReportValue(ctx, reportID)
	if err != nil {
		return err
	}
	names := mcpToolNames(structure)
	prefix := report.OwnerPackage + "."
	for _, name := range names {
		if !canonicalMCPToolName(name) {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: fmt.Sprintf("MCP tool %q must start with a letter, use only letters, numbers, dots, dashes, or underscores, and leave room for generated Cube names", name)}
		}
		if !strings.HasPrefix(name, prefix) {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: fmt.Sprintf("MCP tool %q must use the owner prefix %q", name, prefix)}
		}
	}
	if len(names) == 0 {
		return nil
	}
	rows, err := t.DB.QueryContext(ctx, `SELECT v.generated_dql, v.authored_dql
FROM reports r
JOIN report_versions v ON v.report_id=r.id
LEFT JOIN report_publications p ON p.report_id=r.id
WHERE r.id<>? AND r.deleted_at IS NULL
  AND (v.version_no=(SELECT MAX(v2.version_no) FROM report_versions v2 WHERE v2.report_id=r.id)
    OR v.version_no=p.active_version_no)`, reportID)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	requested := map[string]bool{}
	for _, name := range names {
		key := strings.ToLower(name)
		if requested[key] {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("MCP tool name %q is duplicated in this component", name)}
		}
		requested[key] = true
	}
	for rows.Next() {
		var generated, authored sql.NullString
		if err = rows.Scan(&generated, &authored); err != nil {
			return internal(err)
		}
		source := generated.String
		if strings.TrimSpace(source) == "" {
			source = authored.String
		}
		prepared := dql.PrepareSource(source)
		if prepared == nil || prepared.Directives == nil || prepared.Directives.MCP == nil {
			continue
		}
		name := strings.TrimSpace(prepared.Directives.MCP.Name)
		var settings *spec.ReportSettings
		if prepared.Directives.Settings != nil {
			settings = prepared.Directives.Settings.Report
		}
		for _, candidate := range mcpToolFamily(name, settings) {
			if requested[strings.ToLower(candidate)] {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("MCP tool name %q is already used by another reader", candidate)}
			}
		}
	}
	if err = rows.Err(); err != nil {
		return internal(err)
	}
	return nil
}

func mcpToolNames(structure *readerbuilder.Structure) []string {
	if structure == nil || structure.Component == nil {
		return nil
	}
	var result []string
	for _, route := range structure.Component.Routes {
		if route == nil {
			continue
		}
		for _, exposure := range route.MCP {
			if exposure != nil && (exposure.Kind == "" || exposure.Kind == "tool") && strings.TrimSpace(exposure.Name) != "" {
				var settings *spec.ReportSettings
				if structure.Component.Settings != nil {
					settings = structure.Component.Settings.Report
				}
				result = append(result, mcpToolFamily(strings.TrimSpace(exposure.Name), settings)...)
			}
		}
	}
	return result
}

func mcpToolFamily(base string, settings *spec.ReportSettings) []string {
	result := []string{base}
	if settings == nil || !settings.Enabled {
		return result
	}
	if settings.MCPTool == nil || *settings.MCPTool {
		result = append(result, base+"Cube")
	}
	if settings.Compose != nil && settings.Compose.Enabled && (settings.Compose.MCPTool == nil || *settings.Compose.MCPTool) {
		result = append(result, base+"CubeCompose")
	}
	return result
}

func canonicalMCPToolName(name string) bool {
	if len(name) == 0 || len(name)+len("CubeCompose") > 128 || name[0] < 'A' || name[0] > 'Z' && (name[0] < 'a' || name[0] > 'z') {
		return false
	}
	for _, char := range name {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '.' || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

type versionViewTestRequest struct {
	ReportID  string            `json:"reportId"`
	VersionNo int               `json:"versionNo"`
	View      string            `json:"view"`
	Input     sdk.ViewTestInput `json:"input"`
}

type versionRelationTestRequest struct {
	ReportID  string            `json:"reportId"`
	VersionNo int               `json:"versionNo"`
	Relation  string            `json:"relation"`
	Input     sdk.ViewTestInput `json:"input"`
}

type versionComposeTestRequest struct {
	ReportID  string                   `json:"reportId"`
	VersionNo int                      `json:"versionNo"`
	Input     sdk.CubeComposeTestInput `json:"input"`
}

func (t *Transport) testVersionCompose(ctx context.Context, input, output any) error {
	if t.ComposeTester == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic cube composition tester is not configured"}
	}
	var in versionComposeTestRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 || len(in.Input.Cubes) == 0 || strings.TrimSpace(in.Input.SQL) == "" {
		return invalid(errors.New("reportId, versionNo, cubes, and SQL are required"))
	}
	result, err := t.ComposeTester.TestCompose(ctx, in.ReportID, in.VersionNo, in.Input)
	if err != nil {
		return err
	}
	return assign(output, result)
}

func (t *Transport) testVersionView(ctx context.Context, input, output any) error {
	if t.ViewTester == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic view tester is not configured"}
	}
	var in versionViewTestRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 || strings.TrimSpace(in.View) == "" {
		return invalid(errors.New("reportId, versionNo, and view are required"))
	}
	executionCtx, cancel := context.WithTimeout(ctx, previewExecutionTimeout)
	defer cancel()
	result, err := t.ViewTester.TestView(executionCtx, in.ReportID, in.VersionNo, in.View, in.Input)
	if err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) {
			return err
		}
		return internal(err)
	}
	return assign(output, result)
}

func (t *Transport) testVersionRelation(ctx context.Context, input, output any) error {
	if t.RelationTester == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic relation tester is not configured"}
	}
	var in versionRelationTestRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 || strings.TrimSpace(in.Relation) == "" {
		return invalid(errors.New("reportId, versionNo, and relation are required"))
	}
	executionCtx, cancel := context.WithTimeout(ctx, previewExecutionTimeout)
	defer cancel()
	result, err := t.RelationTester.TestRelation(executionCtx, in.ReportID, in.VersionNo, in.Relation, in.Input)
	if err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) {
			return err
		}
		return internal(err)
	}
	return assign(output, result)
}

func (t *Transport) warmupVersion(ctx context.Context, input, output any) error {
	if t.Warmup == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic warmup executor is not configured"}
	}
	var in versionIdentityRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 {
		return invalid(errors.New("reportId and versionNo are required"))
	}
	return t.startWarmupRun(ctx, in, output)
}

func (t *Transport) runReaderBuilder(ctx context.Context, reportID string, version *sdk.ReportVersion, operation readerbuilder.Operation) (*readerbuilder.Response, error) {
	report, err := t.getReportValue(ctx, reportID)
	if err != nil {
		return nil, err
	}
	connectors, err := t.availableConnectorNames(ctx)
	if err != nil {
		return nil, err
	}
	if operation.Type == readerbuilder.OperationCreateReader && operation.Reader != nil {
		copy := *operation.Reader
		if strings.TrimSpace(copy.Package) == "" {
			copy.Package = strings.TrimSuffix(report.ComponentScope, "/") + "/" + strings.Trim(report.ComponentName, "/")
		}
		if strings.TrimSpace(copy.Connector) == "" {
			copy.Connector = report.DefaultConnectorName
		}
		operation.Reader = &copy
	}
	if operation.Type == readerbuilder.OperationSetPackage && operation.Package != nil {
		copy := *operation.Package
		copy.Path = dynamicComponentScope(report.OwnerID, report.ID) + "/reader"
		operation.Package = &copy
	}
	types, err := t.Predicates.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	service := readerbuilder.New(readerbuilder.Config{
		Types:               types,
		Scope:               report.ComponentScope,
		Name:                report.ComponentName,
		AvailableConnectors: connectors,
	})
	response := service.Apply(ctx, readerbuilder.Request{DQL: versionDQL(version), Operation: operation})
	return &response, nil
}

// availableConnectorNames returns the active connector catalog visible to the
// current Studio principal. Reader Builder validates authored connector names
// against this catalog; the report default is not a special second registry.
func (t *Transport) availableConnectorNames(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM connectors WHERE deleted_at IS NULL AND status = 'active'`
	query, args := connectorReadScope(ctx, query, nil)
	query += ` ORDER BY name`
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, internal(err)
		}
		result = append(result, name)
	}
	if err = rows.Err(); err != nil {
		return nil, internal(err)
	}
	return result, nil
}

func versionDQL(version *sdk.ReportVersion) string {
	if version == nil {
		return ""
	}
	for _, source := range []string{version.GeneratedDQL, version.AuthoredDQL, version.AuthoredSQL} {
		if strings.TrimSpace(source) != "" {
			return source
		}
	}
	return ""
}

func readerInspection(version *sdk.ReportVersion, response *readerbuilder.Response, capabilities sdk.ReportCapabilities) *sdk.ReaderInspection {
	version = redactVersionDQL(version, capabilities.CanUseDQL)
	if response == nil {
		return &sdk.ReaderInspection{Version: version, Capabilities: capabilities}
	}
	structure, _ := json.Marshal(response.Structure)
	result := &sdk.ReaderInspection{
		Version: version, Structure: structure, Diagnostics: readerDiagnostics(response.Diagnostics), Capabilities: capabilities,
	}
	if capabilities.CanUseDQL {
		result.DQL = response.DQL
	}
	return result
}

func redactVersionDQL(version *sdk.ReportVersion, allowed bool) *sdk.ReportVersion {
	if version == nil || allowed {
		return version
	}
	copy := *version
	copy.AuthoredDQL = ""
	copy.GeneratedDQL = ""
	return &copy
}

func readerDiagnostics(diagnostics []*transcribe.Diagnostic) []sdk.Diagnostic {
	result := make([]sdk.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic == nil {
			continue
		}
		result = append(result, sdk.Diagnostic{Severity: string(diagnostic.Severity), Code: diagnostic.Code, Message: diagnostic.Message, Hint: diagnostic.Hint, Line: diagnostic.Span.Start.Line, Column: diagnostic.Span.Start.Char})
	}
	return result
}

func hashVersion(reportID string, versionNo int, mode, authoredSQL, authoredDQL string, spec []byte) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%s:%s:%s", reportID, versionNo, mode, authoredSQL, authoredDQL, spec)))
	return fmt.Sprintf("%x", sum[:])
}

func rawJSON(value sql.NullString) json.RawMessage {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return json.RawMessage(value.String)
}

func rawJSONOrString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func maxZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
func ptrTime(value time.Time) *time.Time { return &value }

type publishRequest struct {
	ReportID  string           `json:"reportId"`
	VersionNo int              `json:"versionNo"`
	Input     sdk.PublishInput `json:"input"`
}

func (t *Transport) publish(ctx context.Context, input, output any, operation string) (returnErr error) {
	t.publicationMu.Lock()
	defer t.publicationMu.Unlock()
	var in publishRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if !validPublicationOperation(operation) || operation == "unpublish" {
		return invalid(errors.New("invalid publication operation"))
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 {
		return invalid(errors.New("reportId and versionNo are required"))
	}
	if principal, ok := sdk.PrincipalFromContext(ctx); ok {
		if in.Input.RequestedBy != "" && in.Input.RequestedBy != principal.Subject {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "publisher must match the Studio principal"}
		}
		in.Input.RequestedBy = principal.Subject
	}
	if strings.TrimSpace(in.Input.RequestedBy) == "" {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	version, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	ownerID, err := t.publicationEventOwner(ctx, in.ReportID)
	if err != nil {
		return err
	}
	versionNo := in.VersionNo
	event := publicationEventRecord{ReportID: in.ReportID, OwnerID: ownerID, Operation: operation, VersionNo: &versionNo, RequestedBy: in.Input.RequestedBy, Reason: boundedPublicationText(in.Input.Reason)}
	defer func() {
		if returnErr != nil {
			t.recordPublicationFailure(ctx, event, returnErr)
		}
	}()
	if in.Input.ExpectedSourceRevision > 0 && in.Input.ExpectedSourceRevision != version.SourceRevision {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
	}
	if version.CompileStatus != "valid" {
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "version must validate before publication"}
	}
	if t.Activator == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic runtime activator is not configured"}
	}
	generation, previous, hasPrevious, now, err := t.stagePublication(ctx, in, version)
	if err != nil {
		return err
	}
	event.GenerationNo = &generation
	if err = t.Activator.Reload(ctx, generation); err != nil {
		t.restoreFailedPublication(ctx, in.ReportID, generation, previous, hasPrevious, err)
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic runtime reload failed: " + err.Error(), Cause: err}
	}
	event.Status = "succeeded"
	event.OccurredAt = now
	if err = t.activatePublication(ctx, in.ReportID, in.VersionNo, generation, now, event); err != nil {
		return t.compensateActivationFailure(ctx, in.ReportID, generation, previous, hasPrevious, err)
	}
	return t.getPublication(ctx, in.ReportID, output)
}

func (t *Transport) stagePublication(ctx context.Context, in publishRequest, version *sdk.ReportVersion) (int64, publicationState, bool, time.Time, error) {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	defer func() { _ = tx.Rollback() }()
	now := t.now()
	if err = ensureNoStagedGeneration(ctx, tx, now); err != nil {
		return 0, publicationState{}, false, time.Time{}, err
	}
	var generation int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(generation_no), 0) + 1 FROM runtime_generations`).Scan(&generation); err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	runtimeRevision := fmt.Sprintf("%s:%d:%d", in.ReportID, in.VersionNo, generation)
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no, source_revision, status, report_count, build_manifest_json, requested_by, requested_at) VALUES (?, ?, 'building', 0, '{}', ?, ?)`, generation, runtimeRevision, in.Input.RequestedBy, now); err != nil {
		return 0, publicationState{}, false, time.Time{}, classify(err, "runtime generation", fmt.Sprint(generation))
	}
	previous, hasPrevious, err := publicationSnapshot(ctx, tx, in.ReportID)
	if err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	if hasPrevious && (previous.status == "pending" || previous.status == "unpublishing") {
		return 0, publicationState{}, false, time.Time{}, &sdk.Error{Code: sdk.ErrorConflict, Message: "another publication transition is already staged"}
	}
	if !hasPrevious {
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_publications(report_id, active_version_no, desired_version_no, desired_generation, active_generation, publication_status, runtime_revision, spec_hash, published_by, published_at) VALUES (?, ?, ?, ?, NULL, 'pending', ?, ?, ?, ?)`, in.ReportID, in.VersionNo, in.VersionNo, generation, runtimeRevision, version.SpecHash, in.Input.RequestedBy, now); err != nil {
			return 0, publicationState{}, false, time.Time{}, internal(err)
		}
	} else if _, err := tx.ExecContext(ctx, `UPDATE report_publications SET desired_version_no=?, desired_generation=?, publication_status='pending', runtime_revision=?, spec_hash=?, published_by=?, published_at=?, failure_json=NULL WHERE report_id=?`, in.VersionNo, generation, runtimeRevision, version.SpecHash, in.Input.RequestedBy, now, in.ReportID); err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	if err := tx.Commit(); err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	return generation, previous, hasPrevious, now, nil
}

const stagedGenerationLease = 5 * time.Minute

func ensureNoStagedGeneration(ctx context.Context, tx *sql.Tx, now time.Time) error {
	cutoff := now.Add(-stagedGenerationLease)
	failure, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "staged_generation_expired", Message: "staged runtime generation expired before activation"}})
	if _, err := tx.ExecContext(ctx, `UPDATE report_publications
SET publication_status=CASE WHEN active_generation IS NULL THEN 'failed' ELSE 'active' END,
    desired_version_no=CASE WHEN active_generation IS NULL THEN desired_version_no ELSE active_version_no END,
    desired_generation=CASE WHEN active_generation IS NULL THEN desired_generation ELSE active_generation END,
    runtime_revision=CASE WHEN active_generation IS NULL THEN runtime_revision ELSE (SELECT source_revision FROM runtime_generations WHERE generation_no=active_generation) END,
    spec_hash=CASE WHEN active_generation IS NULL THEN spec_hash ELSE (SELECT spec_hash FROM report_versions WHERE report_id=report_publications.report_id AND version_no=active_version_no) END,
    failure_json=?
WHERE publication_status IN ('pending','unpublishing')
  AND desired_generation IN (SELECT generation_no FROM runtime_generations WHERE status='building' AND requested_at<?)`, string(failure), cutoff); err != nil {
		return internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE runtime_generations SET status='failed',diagnostics_json=?,retired_at=? WHERE status='building' AND requested_at<?`, string(failure), now, cutoff); err != nil {
		return internal(err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM runtime_generations WHERE status='building'`).Scan(&count); err != nil {
		return internal(err)
	}
	if count > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "another runtime generation is already building"}
	}
	return nil
}

func (t *Transport) activatePublication(ctx context.Context, reportID string, versionNo int, generation int64, now time.Time, event publicationEventRecord) error {
	activated, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer func() { _ = activated.Rollback() }()
	var reportCount int
	var staged int
	if err = activated.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_publications WHERE report_id=? AND publication_status='pending' AND desired_generation=? AND desired_version_no=?`, reportID, generation, versionNo).Scan(&staged); err != nil {
		return internal(err)
	}
	if staged != 1 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "staged publication changed before activation"}
	}
	if _, err = activated.ExecContext(ctx, `UPDATE report_publications SET active_version_no=desired_version_no,active_generation=?,publication_status='active',activated_at=?,failure_json=NULL WHERE report_id=? AND publication_status='pending' AND desired_generation=?`, generation, now, reportID, generation); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE report_publications SET active_generation=? WHERE report_id<>? AND publication_status='active'`, generation, reportID); err != nil {
		return internal(err)
	}
	if err = activated.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_publications WHERE publication_status='active'`).Scan(&reportCount); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE runtime_generations SET status='active',report_count=?,activated_at=? WHERE generation_no=? AND status='building'`, reportCount, now, generation); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE runtime_generations SET status='retired',retired_at=? WHERE status='active' AND generation_no<>?`, now, generation); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE report_versions SET state='superseded' WHERE report_id=? AND version_no<>? AND state='published'`, reportID, versionNo); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE report_versions SET state='published',published_at=? WHERE report_id=? AND version_no=?`, now, reportID, versionNo); err != nil {
		return internal(err)
	}
	if _, err = activated.ExecContext(ctx, `UPDATE reports SET status='active',updated_at=? WHERE id=?`, now, reportID); err != nil {
		return internal(err)
	}
	if err = t.appendPublicationEventTx(ctx, activated, event); err != nil {
		return internal(err)
	}
	if err = activated.Commit(); err != nil {
		return internal(err)
	}
	return nil
}

type publicationState struct {
	activeVersion, desiredGeneration  int64
	desiredVersion                    sql.NullInt64
	activeGeneration                  sql.NullInt64
	status, runtimeRevision, specHash string
	publishedBy                       string
	publishedAt, activatedAt          time.Time
}

func publicationSnapshot(ctx context.Context, tx *sql.Tx, reportID string) (publicationState, bool, error) {
	var value publicationState
	var activatedAt sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at FROM report_publications WHERE report_id=?`, reportID).Scan(&value.activeVersion, &value.desiredVersion, &value.desiredGeneration, &value.activeGeneration, &value.status, &value.runtimeRevision, &value.specHash, &value.publishedBy, &value.publishedAt, &activatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	if activatedAt.Valid {
		value.activatedAt = activatedAt.Time
	}
	return value, true, nil
}

func (t *Transport) restoreFailedPublication(ctx context.Context, reportID string, generation int64, previous publicationState, hasPrevious bool, cause error) {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	failure, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "runtime_reload", Message: cause.Error()}})
	_, _ = tx.ExecContext(ctx, `UPDATE runtime_generations SET status='failed',diagnostics_json=? WHERE generation_no=?`, string(failure), generation)
	if hasPrevious {
		active := any(nil)
		if previous.activeGeneration.Valid {
			active = previous.activeGeneration.Int64
		}
		desired := any(nil)
		if previous.desiredVersion.Valid {
			desired = previous.desiredVersion.Int64
		}
		_, _ = tx.ExecContext(ctx, `UPDATE report_publications SET active_version_no=?,desired_version_no=?,desired_generation=?,active_generation=?,publication_status=?,runtime_revision=?,spec_hash=?,published_by=?,published_at=?,activated_at=?,failure_json=? WHERE report_id=?`, previous.activeVersion, desired, previous.desiredGeneration, active, previous.status, previous.runtimeRevision, previous.specHash, previous.publishedBy, previous.publishedAt, nullableTime(previous.activatedAt), string(failure), reportID)
	} else {
		_, _ = tx.ExecContext(ctx, `UPDATE report_publications SET publication_status='failed',active_generation=NULL,failure_json=? WHERE report_id=?`, string(failure), reportID)
	}
	_ = tx.Commit()
}

func (t *Transport) compensateActivationFailure(ctx context.Context, reportID string, generation int64, previous publicationState, hasPrevious bool, cause error) error {
	rollbackGeneration := int64(0)
	if hasPrevious && previous.activeGeneration.Valid {
		rollbackGeneration = previous.activeGeneration.Int64
	}
	rollbackErr := t.Activator.Reload(context.WithoutCancel(ctx), rollbackGeneration)
	combined := cause
	if rollbackErr != nil {
		combined = errors.Join(cause, fmt.Errorf("runtime compensation failed: %w", rollbackErr))
	}
	t.restoreFailedPublication(context.WithoutCancel(ctx), reportID, generation, previous, hasPrevious, combined)
	return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "runtime activation persistence failed: " + combined.Error(), Cause: combined}
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

type unpublishRequest struct {
	ReportID string             `json:"reportId"`
	Input    sdk.UnpublishInput `json:"input"`
}

func (t *Transport) unpublish(ctx context.Context, input, output any) (returnErr error) {
	t.publicationMu.Lock()
	defer t.publicationMu.Unlock()
	var in unpublishRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" {
		return invalid(errors.New("reportId is required"))
	}
	if principal, ok := sdk.PrincipalFromContext(ctx); ok {
		if in.Input.RequestedBy != "" && in.Input.RequestedBy != principal.Subject {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "publisher must match the Studio principal"}
		}
		in.Input.RequestedBy = principal.Subject
	}
	if strings.TrimSpace(in.Input.RequestedBy) == "" {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	ownerID, err := t.publicationEventOwner(ctx, in.ReportID)
	if err != nil {
		return err
	}
	event := publicationEventRecord{ReportID: in.ReportID, OwnerID: ownerID, Operation: "unpublish", RequestedBy: in.Input.RequestedBy, Reason: boundedPublicationText(in.Input.Reason)}
	defer func() {
		if returnErr != nil {
			t.recordPublicationFailure(ctx, event, returnErr)
		}
	}()
	if t.Activator == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic runtime activator is not configured"}
	}
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer func() { _ = tx.Rollback() }()
	previous, found, err := publicationSnapshot(ctx, tx, in.ReportID)
	if err != nil {
		return internal(err)
	}
	if !found {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "publication not found"}
	}
	if !previous.activeGeneration.Valid || previous.status != "active" {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "publication is not active"}
	}
	activeVersion := int(previous.activeVersion)
	event.VersionNo = &activeVersion
	now := t.now()
	if err = ensureNoStagedGeneration(ctx, tx, now); err != nil {
		return err
	}
	var generation int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(generation_no), 0) + 1 FROM runtime_generations`).Scan(&generation); err != nil {
		return internal(err)
	}
	revision := fmt.Sprintf("unpublish:%s:%d", in.ReportID, generation)
	if _, err = tx.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(?,?,'building',0,'{}',?,?)`, generation, revision, in.Input.RequestedBy, now); err != nil {
		return classify(err, "runtime generation", fmt.Sprint(generation))
	}
	if _, err = tx.ExecContext(ctx, `UPDATE report_publications SET desired_version_no=NULL,desired_generation=?,publication_status='unpublishing',failure_json=NULL WHERE report_id=?`, generation, in.ReportID); err != nil {
		return internal(err)
	}
	if err = tx.Commit(); err != nil {
		return internal(err)
	}
	event.GenerationNo = &generation
	if err = t.Activator.Reload(ctx, generation); err != nil {
		t.restoreFailedPublication(ctx, in.ReportID, generation, previous, true, err)
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic runtime reload failed: " + err.Error(), Cause: err}
	}
	event.Status = "succeeded"
	event.OccurredAt = now
	if err = t.activateUnpublish(ctx, in.ReportID, generation, now, event); err != nil {
		return t.compensateActivationFailure(ctx, in.ReportID, generation, previous, true, err)
	}
	value := &sdk.Publication{ReportID: in.ReportID, ActiveVersionNo: int(previous.activeVersion), DesiredGeneration: generation, Status: "unpublished", PublishedAt: ptrTime(now)}
	return assign(output, value)
}

func (t *Transport) activateUnpublish(ctx context.Context, reportID string, generation int64, now time.Time, event publicationEventRecord) error {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer func() { _ = tx.Rollback() }()
	var staged int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_publications WHERE report_id=? AND publication_status='unpublishing' AND desired_generation=?`, reportID, generation).Scan(&staged); err != nil {
		return internal(err)
	}
	if staged != 1 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "staged unpublish changed before activation"}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM report_publications WHERE report_id=?`, reportID); err != nil {
		return internal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE report_publications SET active_generation=? WHERE publication_status='active'`, generation); err != nil {
		return internal(err)
	}
	var reportCount int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_publications WHERE publication_status='active'`).Scan(&reportCount); err != nil {
		return internal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE runtime_generations SET status='active',report_count=?,activated_at=? WHERE generation_no=? AND status='building'`, reportCount, now, generation); err != nil {
		return internal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE runtime_generations SET status='retired',retired_at=? WHERE status='active' AND generation_no<>?`, now, generation); err != nil {
		return internal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE report_versions SET state='superseded' WHERE report_id=? AND state='published'`, reportID); err != nil {
		return internal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE reports SET status='disabled',updated_at=? WHERE id=?`, now, reportID); err != nil {
		return internal(err)
	}
	if err = t.appendPublicationEventTx(ctx, tx, event); err != nil {
		return internal(err)
	}
	if err = tx.Commit(); err != nil {
		return internal(err)
	}
	return nil
}

func (t *Transport) getPublication(ctx context.Context, reportID string, output any) error {
	value, err := scanPublication(t.DB.QueryRowContext(ctx, `SELECT report_id, active_version_no, desired_version_no, desired_generation, active_generation, publication_status, runtime_revision, spec_hash, published_at FROM report_publications WHERE report_id=?`, reportID))
	if err != nil {
		return mapReadError(err, "publication", reportID)
	}
	return assign(output, value)
}

func scanPublication(scanner interface{ Scan(...any) error }) (*sdk.Publication, error) {
	var value sdk.Publication
	var activeGeneration sql.NullInt64
	var desiredVersion sql.NullInt64
	var publishedAt sql.NullTime
	if err := scanner.Scan(&value.ReportID, &value.ActiveVersionNo, &desiredVersion, &value.DesiredGeneration, &activeGeneration, &value.Status, &value.RuntimeRevision, &value.SpecHash, &publishedAt); err != nil {
		return nil, err
	}
	if activeGeneration.Valid {
		generation := activeGeneration.Int64
		value.ActiveGeneration = &generation
	}
	if desiredVersion.Valid {
		version := int(desiredVersion.Int64)
		value.DesiredVersionNo = &version
	}
	if publishedAt.Valid {
		published := publishedAt.Time
		value.PublishedAt = &published
	}
	return &value, nil
}

func (t *Transport) runtimeStatus(ctx context.Context, output any) error {
	var value sdk.RuntimeStatus
	var activatedAt sql.NullTime
	var diagnosticsJSON sql.NullString
	err := t.DB.QueryRowContext(ctx, `SELECT generation_no, status, report_count, activated_at, diagnostics_json FROM runtime_generations WHERE status='active' ORDER BY generation_no DESC LIMIT 1`).Scan(&value.ActiveGeneration, &value.Status, &value.ReportCount, &activatedAt, &diagnosticsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		value.Status = "idle"
		return assign(output, &value)
	}
	if err != nil {
		return internal(err)
	}
	if activatedAt.Valid {
		at := activatedAt.Time
		value.ActivatedAt = &at
	}
	if diagnosticsJSON.Valid && strings.TrimSpace(diagnosticsJSON.String) != "" {
		_ = json.Unmarshal([]byte(diagnosticsJSON.String), &value.Diagnostics)
	}
	readers, err := t.runtimeReaders(ctx, value.ActiveGeneration)
	if err != nil {
		return internal(err)
	}
	value.Readers = readers
	checkedAt := time.Now().UTC()
	if t.Now != nil {
		checkedAt = t.Now().UTC()
	}
	if t.RuntimeProbe == nil {
		value.Host = &sdk.RuntimeHost{Status: "unknown", CheckedAt: checkedAt}
	} else if host, probeErr := t.RuntimeProbe.ProbeRuntime(ctx); probeErr != nil || host == nil {
		value.Host = &sdk.RuntimeHost{Status: "unavailable", CheckedAt: checkedAt}
	} else {
		if host.CheckedAt.IsZero() {
			host.CheckedAt = checkedAt
		}
		value.Host = host
	}
	return assign(output, &value)
}

func (t *Transport) runtimeReaders(ctx context.Context, generation int64) ([]sdk.RuntimeReader, error) {
	query := `SELECT r.id,r.title,r.namespace,r.owner_id,r.default_connector_name,r.component_name,p.active_version_no,p.publication_status,p.runtime_revision,p.activated_at
FROM report_publications p JOIN reports r ON r.id=p.report_id
WHERE p.active_generation=? AND p.publication_status='active' AND r.deleted_at IS NULL`
	args := []any{generation}
	principal, scoped := sdk.PrincipalFromContext(ctx)
	if scoped {
		query += ` AND (r.owner_id=? OR EXISTS(SELECT 1 FROM report_acl acl WHERE acl.report_id=r.id AND acl.subject_type='user' AND acl.subject_id=? AND acl.can_publish=TRUE))`
		args = append(args, principal.Subject, principal.Subject)
	}
	query += ` ORDER BY r.namespace,r.title,r.id`
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var result []sdk.RuntimeReader
	for rows.Next() {
		var item sdk.RuntimeReader
		var ownerID string
		var revision sql.NullString
		var activated sql.NullTime
		if err = rows.Scan(&item.ReportID, &item.Title, &item.Namespace, &ownerID, &item.ConnectorName, &item.ComponentName, &item.VersionNo, &item.Status, &revision, &activated); err != nil {
			rows.Close()
			return nil, err
		}
		item.OwnerPackage = sdk.OwnerPackageSegment(ownerID)
		if revision.Valid {
			item.RuntimeRevision = revision.String
		}
		if activated.Valid {
			at := activated.Time
			item.ActivatedAt = &at
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(result) == 0 {
		return result, nil
	}
	index := make(map[string]int, len(result))
	for i := range result {
		index[result[i].ReportID] = i
	}
	exposureQuery := `SELECT e.report_id,e.kind,e.name,e.route_method,e.route_path,e.description,e.mime_type,e.enabled
FROM report_mcp_exposures e
JOIN report_publications p ON p.report_id=e.report_id AND p.active_version_no=e.version_no
JOIN reports r ON r.id=e.report_id
WHERE p.active_generation=? AND p.publication_status='active' AND r.deleted_at IS NULL`
	exposureArgs := []any{generation}
	if scoped {
		exposureQuery += ` AND (r.owner_id=? OR EXISTS(SELECT 1 FROM report_acl acl WHERE acl.report_id=r.id AND acl.subject_type='user' AND acl.subject_id=? AND acl.can_publish=TRUE))`
		exposureArgs = append(exposureArgs, principal.Subject, principal.Subject)
	}
	exposureQuery += ` ORDER BY e.report_id,e.ordinal,e.exposure_id`
	exposures, err := t.DB.QueryContext(ctx, exposureQuery, exposureArgs...)
	if err != nil {
		return nil, err
	}
	for exposures.Next() {
		var reportID string
		var item sdk.RuntimeMCPExposure
		var description, mimeType sql.NullString
		if err = exposures.Scan(&reportID, &item.Kind, &item.Name, &item.Method, &item.Path, &description, &mimeType, &item.Enabled); err != nil {
			return nil, err
		}
		item.Description = description.String
		item.MIMEType = mimeType.String
		if position, ok := index[reportID]; ok {
			item.Component = result[position].ComponentName
			result[position].MCPExposures = append(result[position].MCPExposures, item)
		}
	}
	if err = exposures.Err(); err != nil {
		exposures.Close()
		return nil, err
	}
	exposures.Close()
	for position := range result {
		version, loadErr := t.getVersionValue(ctx, result[position].ReportID, result[position].VersionNo)
		if loadErr != nil {
			return nil, loadErr
		}
		inspected, inspectErr := t.runReaderBuilder(ctx, result[position].ReportID, version, readerbuilder.Operation{Type: readerbuilder.OperationInspect})
		if inspectErr != nil {
			return nil, inspectErr
		}
		if inspected.Structure == nil || inspected.Structure.Component == nil {
			continue
		}
		seenTools := map[string]bool{}
		for _, exposure := range result[position].MCPExposures {
			seenTools[strings.ToLower(exposure.Name)] = true
		}
		for _, route := range inspected.Structure.Component.Routes {
			if route == nil {
				continue
			}
			for _, exposure := range route.MCP {
				if exposure == nil || exposure.Kind != "" && exposure.Kind != "tool" || strings.TrimSpace(exposure.Name) == "" || seenTools[strings.ToLower(exposure.Name)] {
					continue
				}
				seenTools[strings.ToLower(exposure.Name)] = true
				result[position].MCPExposures = append(result[position].MCPExposures, sdk.RuntimeMCPExposure{Kind: "tool", Name: exposure.Name, Component: inspected.Structure.Component.Name, Method: route.Method, Path: route.Path, Description: exposure.Description, Enabled: true})
			}
		}
		for _, projected := range datlyreport.ProjectedComponents(inspected.Structure.Component) {
			if seenTools[strings.ToLower(projected.Name)] {
				continue
			}
			seenTools[strings.ToLower(projected.Name)] = true
			result[position].MCPExposures = append(result[position].MCPExposures, sdk.RuntimeMCPExposure{Kind: "tool", Name: projected.Name, Component: projected.Name, Method: projected.Method, Path: projected.Path, Description: projected.Description, Enabled: projected.MCPEnabled})
		}
	}

	resourceQuery := `SELECT f.report_id,f.namespace,f.root_path,f.uri_prefix
FROM report_resource_folders f
JOIN report_publications p ON p.report_id=f.report_id AND p.active_version_no=f.version_no
JOIN reports r ON r.id=f.report_id
WHERE p.active_generation=? AND p.publication_status='active' AND r.deleted_at IS NULL`
	resourceArgs := []any{generation}
	if scoped {
		resourceQuery += ` AND (r.owner_id=? OR EXISTS(SELECT 1 FROM report_acl acl WHERE acl.report_id=r.id AND acl.subject_type='user' AND acl.subject_id=? AND acl.can_publish=TRUE))`
		resourceArgs = append(resourceArgs, principal.Subject, principal.Subject)
	}
	resourceQuery += ` ORDER BY f.report_id,f.ordinal,f.folder_id`
	resources, err := t.DB.QueryContext(ctx, resourceQuery, resourceArgs...)
	if err != nil {
		return nil, err
	}
	for resources.Next() {
		var reportID string
		var item sdk.RuntimeMCPResource
		if err = resources.Scan(&reportID, &item.Namespace, &item.RootPath, &item.URIPrefix); err != nil {
			resources.Close()
			return nil, err
		}
		if position, ok := index[reportID]; ok {
			result[position].MCPResources = append(result[position].MCPResources, item)
		}
	}
	if err = resources.Err(); err != nil {
		resources.Close()
		return nil, err
	}
	resources.Close()

	skillQuery := `SELECT s.report_id,s.skill_id,s.skill_root,f.uri_prefix
FROM report_skill_roots s
JOIN report_resource_folders f ON f.report_id=s.report_id AND f.version_no=s.version_no AND f.folder_id=s.folder_id
JOIN report_publications p ON p.report_id=s.report_id AND p.active_version_no=s.version_no
JOIN reports r ON r.id=s.report_id
WHERE p.active_generation=? AND p.publication_status='active' AND r.deleted_at IS NULL`
	skillArgs := []any{generation}
	if scoped {
		skillQuery += ` AND (r.owner_id=? OR EXISTS(SELECT 1 FROM report_acl acl WHERE acl.report_id=r.id AND acl.subject_type='user' AND acl.subject_id=? AND acl.can_publish=TRUE))`
		skillArgs = append(skillArgs, principal.Subject, principal.Subject)
	}
	skillQuery += ` ORDER BY s.report_id,s.ordinal,s.skill_id`
	skills, err := t.DB.QueryContext(ctx, skillQuery, skillArgs...)
	if err != nil {
		return nil, err
	}
	defer skills.Close()
	for skills.Next() {
		var reportID string
		var item sdk.RuntimeSkill
		if err = skills.Scan(&reportID, &item.SkillID, &item.SkillRoot, &item.URIPrefix); err != nil {
			return nil, err
		}
		if position, ok := index[reportID]; ok {
			result[position].Skills = append(result[position].Skills, item)
		}
	}
	return result, skills.Err()
}

type previewRequest struct {
	ReportID  string           `json:"reportId"`
	VersionNo int              `json:"versionNo"`
	Input     sdk.PreviewInput `json:"input"`
}

func (t *Transport) preview(ctx context.Context, input, output any) error {
	if t.Preview == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "preview executor is not configured"}
	}
	var in previewRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 {
		return invalid(errors.New("reportId and versionNo are required"))
	}
	executionCtx, cancel := context.WithTimeout(ctx, previewExecutionTimeout)
	defer cancel()
	result, err := t.Preview.Execute(executionCtx, in.ReportID, in.VersionNo, in.Input)
	if err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) {
			return err
		}
		return internal(err)
	}
	return assign(output, result)
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func assign(output, value any) error {
	if output == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
func invalid(err error) error {
	return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: err.Error(), Cause: err}
}
func internal(err error) error {
	return &sdk.Error{Code: sdk.ErrorInternal, Message: err.Error(), Cause: err}
}
func classify(err error, kind, id string) error {
	if strings.Contains(strings.ToLower(err.Error()), "constraint") || strings.Contains(strings.ToLower(err.Error()), "unique") {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("%s %q already exists", kind, id), Cause: err}
	}
	return internal(err)
}
func mapReadError(err error, kind, id string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: fmt.Sprintf("%s %q not found", kind, id), Cause: err}
	}
	return internal(err)
}
