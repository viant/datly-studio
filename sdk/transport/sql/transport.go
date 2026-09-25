// Package sqltransport is the canonical SQLite/MySQL-backed SDK transport.
// It speaks SDK DTOs only; Forge does not need to know about Studio control
// services or the generated component implementation.
package sqltransport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viant/datly-studio/internal/reportcapability"
	"github.com/viant/datly-studio/sdk"
	connectorconfig "github.com/viant/datly-studio/studio/connectors/store_config"
	connectorinsert "github.com/viant/datly-studio/studio/connectors/store_insert"
	connectorstatus "github.com/viant/datly-studio/studio/connectors/store_status"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	versionedit "github.com/viant/datly-studio/studio/report_versions/store_edit"
	versioninsert "github.com/viant/datly-studio/studio/report_versions/store_insert"
	versionvalidation "github.com/viant/datly-studio/studio/report_versions/store_validation"
	reportconfig "github.com/viant/datly-studio/studio/reports/store_config"
	reportinsert "github.com/viant/datly-studio/studio/reports/store_insert"
	"github.com/viant/datly/authoring/readerbuilder"
	datlyreport "github.com/viant/datly/report"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/dql"
	xhandler "github.com/viant/xdatly/handler"
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
	DB                       *sql.DB
	Now                      func() time.Time
	Preview                  PreviewExecutor
	ViewTester               ViewTester
	RelationTester           RelationTester
	ComposeTester            CubeComposeTester
	Warmup                   WarmupExecutor
	Validator                VersionValidator
	Activator                RuntimeActivator
	RuntimeProbe             RuntimeHostProbe
	Authorizer               Authorizer
	SystemCredentialProvider sdk.SystemCredentialProvider
	Probe                    sdk.ConnectorProbe
	Catalog                  sdk.CatalogExplorer
	SQLTester                sdk.ConnectorSQLTester
	Predicates               *predicatecatalog.Catalog
	predicateReaderMu        sync.Mutex
	predicateReader          *authorizationPredicateReader
	capabilityReaderMu       sync.Mutex
	capabilityReader         *reportcapability.Reader
	aclReaderMu              sync.Mutex
	aclReader                *aclStoreReader
	publicationReaderMu      sync.Mutex
	publicationReader        *publicationEventStoreReader
	publicationMu            sync.Mutex
	warmupMu                 sync.Mutex
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
	err := t.writeConnectorInsertRow(ctx, &connectorinsert.StoredConnector{
		Name: in.Name, Driver: in.Driver, DsnTemplate: namespaceOptionalDescription(in.DSNTemplate),
		SecretRef: namespaceOptionalDescription(in.SecretRef), Description: namespaceOptionalDescription(in.Description),
		OwnerId: in.OwnerID, Status: "draft", OptionsJson: json.RawMessage(options), Etag: 1,
		CreatedAt: now, UpdatedAt: now,
		Has: &connectorinsert.StoredConnectorHas{Name: true, Driver: true, DsnTemplate: true,
			SecretRef: true, Description: true, OwnerId: true, Status: true, OptionsJson: true,
			Etag: true, CreatedAt: true, UpdatedAt: true},
	})
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
	value, err := t.getConnectorValue(ctx, in.Name)
	if err != nil {
		return err
	}
	return assign(output, &value.Connector)
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
	items, err := t.readConnectorCatalog(ctx, connectorCatalogRequest{Query: in.Query, Status: in.Status,
		OwnerID: in.OwnerID, Driver: in.Driver, Limit: limit, Offset: in.Offset})
	if err != nil {
		return internal(err)
	}
	page := &sdk.ConnectorPage{Items: items, Limit: limit, Offset: in.Offset}
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
	if in.Input.Driver != nil {
		current.Driver = *in.Input.Driver
	}
	if in.Input.DSNTemplate != nil {
		current.DSNTemplate = *in.Input.DSNTemplate
	}
	if in.Input.SecretRef != nil {
		current.SecretRef = *in.Input.SecretRef
	}
	if in.Input.Description != nil {
		current.Description = *in.Input.Description
	}
	options := current.Options
	if in.Input.Options != nil {
		options = *in.Input.Options
	}
	now := t.now()
	etag := in.Input.ETag
	err = t.writeConnectorConfig(ctx, &connectorconfig.StoredConnector{
		Name: in.Name, Driver: current.Driver, DsnTemplate: namespaceOptionalDescription(current.DSNTemplate),
		SecretRef: namespaceOptionalDescription(current.SecretRef), Description: namespaceOptionalDescription(current.Description),
		OptionsJson: options, Etag: &etag, UpdatedAt: &now,
		Has: &connectorconfig.StoredConnectorHas{Name: true, Driver: true, DsnTemplate: true,
			SecretRef: true, Description: true, OptionsJson: true, Etag: true, UpdatedAt: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector etag does not match"}
		}
		return internal(err)
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
	statusOperation := "activate"
	if operation == sdk.OperationConnectorDisable {
		statusOperation = "disable"
	} else {
		current, err := t.getConnectorValue(ctx, in.Name)
		if err != nil {
			return err
		}
		if current.LastTestStatus != "passed" {
			return invalid(errors.New("connector must pass a connectivity test before activation"))
		}
	}
	now, etag := t.now(), in.ETag
	err := t.writeConnectorStatus(ctx, statusOperation, &connectorstatus.StoredConnector{Name: in.Name, Etag: &etag, UpdatedAt: &now,
		Has: &connectorstatus.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true}})
	if err != nil {
		if errors.Is(err, connectorstatus.ErrProbeRequired) {
			return invalid(err)
		}
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector etag does not match"}
		}
		return internal(err)
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
	used, err := t.connectorUsage(ctx, in.Name)
	if err != nil {
		return internal(err)
	}
	if used > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector is referenced by a report"}
	}
	now, etag := t.now(), in.ETag
	err = t.writeConnectorStatus(ctx, "delete", &connectorstatus.StoredConnector{Name: in.Name, Etag: &etag, UpdatedAt: &now, DeletedAt: &now,
		Has: &connectorstatus.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true, DeletedAt: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "connector not found"}
		}
		return internal(err)
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
	now, etag := t.now(), value.ETag
	status, code, testedAt := result.Status, namespaceOptionalDescription(result.ErrorCode), result.TestedAt
	err = t.writeConnectorStatus(ctx, "probe", &connectorstatus.StoredConnector{Name: value.Name, Etag: &etag, UpdatedAt: &now,
		LastTestStatus: &status, LastTestErrorCode: code, LastTestedAt: &testedAt,
		Has: &connectorstatus.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true,
			LastTestStatus: true, LastTestErrorCode: true, LastTestedAt: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "connector changed during connectivity test"}
		}
		return internal(err)
	}
	if probeErr != nil {
		return assign(output, result)
	}
	return assign(output, result)
}

type connectorValue struct{ sdk.Connector }

func (t *Transport) getConnectorValue(ctx context.Context, name string) (*connectorValue, error) {
	if name == "" {
		return nil, mapReadError(sql.ErrNoRows, "connector", name)
	}
	items, err := t.readConnectorCatalog(ctx, connectorCatalogRequest{Name: name, Limit: 2})
	if err != nil {
		return nil, internal(err)
	}
	if len(items) == 0 {
		return nil, mapReadError(sql.ErrNoRows, "connector", name)
	}
	if len(items) != 1 {
		return nil, internal(errors.New("connector catalog returned ambiguous rows"))
	}
	return &connectorValue{*items[0]}, nil
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
	err = t.writeReportInsert(ctx, &reportinsert.StoredReport{
		Id: in.ID, Namespace: in.Namespace, Slug: in.Slug, Title: in.Title,
		Description: namespaceOptionalDescription(in.Description), OwnerId: in.OwnerID,
		Status: "draft", DefaultConnectorName: in.DefaultConnectorName,
		ComponentScope: in.ComponentScope, ComponentName: in.ComponentName,
		Etag: 1, CreatedAt: now, UpdatedAt: now,
		Has: &reportinsert.StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true,
			Description: true, OwnerId: true, Status: true, DefaultConnectorName: true,
			ComponentScope: true, ComponentName: true, Etag: true, CreatedAt: true, UpdatedAt: true},
	})
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
	items, err := t.readReportCatalog(ctx, reportCatalogRequest{Query: in.Query, Namespace: in.Namespace,
		Status: in.Status, OwnerID: in.OwnerID, ConnectorName: in.ConnectorName,
		Limit: limit, Offset: in.Offset})
	if err != nil {
		return internal(err)
	}
	page := &sdk.ReportPage{Items: items, Limit: limit, Offset: in.Offset}
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
			nextVersion, versionErr := t.nextVersionNo(ctx, in.ID)
			if versionErr != nil {
				return internal(versionErr)
			}
			if nextVersion > 1 {
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
	now := t.now()
	etag := in.Input.ETag
	err = t.writeReportConfig(ctx, nil, &reportconfig.StoredReport{
		Id: in.ID, Namespace: current.Namespace, Slug: current.Slug, Title: current.Title,
		Description: namespaceOptionalDescription(current.Description), OwnerId: current.OwnerID,
		Status: current.Status, DefaultConnectorName: current.DefaultConnectorName,
		ComponentScope: current.ComponentScope, ComponentName: current.ComponentName,
		CurrentDraftVersion: draft, Etag: &etag, UpdatedAt: &now,
		Has: &reportconfig.StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true,
			Description: true, OwnerId: true, Status: true, DefaultConnectorName: true,
			ComponentScope: true, ComponentName: true, CurrentDraftVersion: true, Etag: true, UpdatedAt: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "report etag does not match"}
		}
		return internal(err)
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
	insertErr := t.insertDefaultNamespace(ctx, ownerID, now)
	if insertErr != nil {
		if retryErr := t.requireActiveNamespace(ctx, ownerID, name); retryErr == nil {
			return nil
		}
		return classify(insertErr, "namespace", name)
	}
	return nil
}

func (t *Transport) requireActiveNamespace(ctx context.Context, ownerID, name string) error {
	status, err := t.ownedNamespaceStatus(ctx, ownerID, name)
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
	if id == "" {
		return nil, mapReadError(sql.ErrNoRows, "report", id)
	}
	items, err := t.readReportCatalog(ctx, reportCatalogRequest{ID: id, Limit: 2})
	if err != nil {
		return nil, internal(err)
	}
	if len(items) == 0 {
		return nil, mapReadError(sql.ErrNoRows, "report", id)
	}
	if len(items) != 1 {
		return nil, internal(errors.New("report catalog returned ambiguous rows"))
	}
	return items[0], nil
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
	next, err := t.nextVersionNo(ctx, in.ReportID)
	if err != nil {
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
	err = t.writeVersionInsert(ctx, &versioninsert.StoredVersion{
		ReportId: in.ReportID, VersionNo: next, State: "draft", AuthoringMode: mode,
		AuthoredSql:       namespaceOptionalDescription(in.Input.AuthoredSQL),
		AuthoredDql:       namespaceOptionalDescription(in.Input.AuthoredDQL),
		ComponentSpecJson: spec, SpecFormatVersion: "studio.v1", SpecHash: specHash,
		GeneratedDql: namespaceOptionalDescription(generated), TypeManifestJson: json.RawMessage(`{}`),
		CompileStatus: "pending", DatlyVersion: "v1", CompilerVersion: "studio.v1",
		SourceRevision: 1, Notes: namespaceOptionalDescription(in.Input.Notes),
		CreatedBy: in.Input.CreatedBy, CreatedAt: now,
		Has: &versioninsert.StoredVersionHas{ReportId: true, VersionNo: true, State: true,
			AuthoringMode: true, AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true,
			SpecFormatVersion: true, SpecHash: true, GeneratedDql: true, TypeManifestJson: true,
			CompileStatus: true, DatlyVersion: true, CompilerVersion: true, SourceRevision: true,
			Notes: true, CreatedBy: true, CreatedAt: true},
	})
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
	if strings.TrimSpace(in.ReportID) == "" {
		return invalid(errors.New("reportId is required"))
	}
	limit := in.Input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	items, err := t.readVersionCatalog(ctx, versionCatalogRequest{ReportID: in.ReportID,
		State: in.Input.State, AuthoringMode: in.Input.AuthoringMode,
		CompileStatus: in.Input.CompileStatus, CreatedBy: in.Input.CreatedBy,
		Limit: limit, Offset: maxZero(in.Input.Offset)})
	if err != nil {
		return internal(err)
	}
	page := &sdk.VersionPage{Items: items, Limit: limit, Offset: maxZero(in.Input.Offset)}
	return assign(output, page)
}

func (t *Transport) getVersionValue(ctx context.Context, reportID string, versionNo int) (*sdk.ReportVersion, error) {
	if reportID == "" || versionNo <= 0 {
		return nil, mapReadError(sql.ErrNoRows, "report version", fmt.Sprintf("%s/%d", reportID, versionNo))
	}
	items, err := t.readVersionCatalog(ctx, versionCatalogRequest{ReportID: reportID, VersionNo: versionNo, Limit: 2})
	if err != nil {
		return nil, internal(err)
	}
	if len(items) == 0 {
		return nil, mapReadError(sql.ErrNoRows, "report version", fmt.Sprintf("%s/%d", reportID, versionNo))
	}
	if len(items) != 1 {
		return nil, internal(errors.New("version catalog returned ambiguous rows"))
	}
	return items[0], nil
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
	hash := hashVersion(in.ReportID, in.VersionNo, current.AuthoringMode, authoredSQL, authoredDQL, spec)
	expected := current.SourceRevision
	err = t.writeVersionEdit(ctx, nil, &versionedit.StoredVersion{
		ReportId: in.ReportID, VersionNo: in.VersionNo,
		AuthoredSql:       namespaceOptionalDescription(authoredSQL),
		AuthoredDql:       namespaceOptionalDescription(authoredDQL),
		ComponentSpecJson: spec, SpecHash: hash,
		GeneratedDql:  namespaceOptionalDescription(generatedDQL),
		CompileStatus: "pending", CompileDiagnosticsJson: json.RawMessage(`[]`),
		SourceRevision: &expected,
		Has: &versionedit.StoredVersionHas{ReportId: true, VersionNo: true,
			AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true,
			SpecHash: true, GeneratedDql: true, CompileStatus: true,
			CompileDiagnosticsJson: true, SourceRevision: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
		}
		return internal(err)
	}
	updated, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	return assign(output, &sdk.EditResult{Version: updated})
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
	now := t.now()
	expected := value.SourceRevision
	err = t.writeVersionValidation(ctx, &versionvalidation.StoredVersion{
		ReportId: in.ReportID, VersionNo: in.VersionNo, CompileStatus: status,
		CompileDiagnosticsJson: diagnosticJSON, ValidatedAt: &now,
		SourceRevision: &expected,
		Has: &versionvalidation.StoredVersionHas{ReportId: true, VersionNo: true,
			CompileStatus: true, CompileDiagnosticsJson: true, ValidatedAt: true,
			SourceRevision: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision changed during validation"}
		}
		return internal(err)
	}
	value.CompileStatus = status
	value.CompileDiagnostics = append(json.RawMessage(nil), diagnosticJSON...)
	value.ValidatedAt = &now
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
	editedVersion, err := t.persistReaderBuilderDQL(ctx, in.ReportID, version, response.DQL, connector,
		operation.Type == readerbuilder.OperationSetPackage)
	if err != nil {
		return err
	}
	inspection.Version = editedVersion
	return assign(output, &sdk.ReaderBuilderResult{Applied: true, Inspection: inspection})
}

func (t *Transport) persistReaderBuilderDQL(ctx context.Context, reportID string, current *sdk.ReportVersion, dql, connector string, setPackage bool) (*sdk.ReportVersion, error) {
	if current == nil || strings.TrimSpace(dql) == "" {
		return nil, invalid(errors.New("reader builder DQL is required"))
	}
	spec := current.ComponentSpec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	hash := hashVersion(reportID, current.VersionNo, current.AuthoringMode, current.AuthoredSQL, dql, spec)
	var report *sdk.Report
	var err error
	if connector != "" || setPackage {
		report, err = t.getReportValue(ctx, reportID)
		if err != nil {
			return nil, err
		}
	}
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, internal(err)
	}
	defer tx.Rollback()
	expected := current.SourceRevision
	err = t.writeVersionEdit(ctx, tx, &versionedit.StoredVersion{
		ReportId: reportID, VersionNo: current.VersionNo,
		AuthoredSql: namespaceOptionalDescription(current.AuthoredSQL),
		AuthoredDql: namespaceOptionalDescription(dql), ComponentSpecJson: spec,
		SpecHash: hash, GeneratedDql: namespaceOptionalDescription(dql),
		CompileStatus: "pending", CompileDiagnosticsJson: json.RawMessage(`[]`),
		SourceRevision: &expected,
		Has: &versionedit.StoredVersionHas{ReportId: true, VersionNo: true,
			AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true,
			SpecHash: true, GeneratedDql: true, CompileStatus: true,
			CompileDiagnosticsJson: true, SourceRevision: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return nil, &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match"}
		}
		return nil, internal(err)
	}
	if report != nil && (setPackage || connector != "" && connector != report.DefaultConnectorName) {
		etag := report.ETag
		now := t.now()
		defaultConnector := report.DefaultConnectorName
		if connector != "" {
			defaultConnector = connector
		}
		componentScope, componentName := report.ComponentScope, report.ComponentName
		if setPackage {
			componentScope = dynamicComponentScope(report.OwnerID, report.ID)
			componentName = "reader"
		}
		err = t.writeReportConfig(ctx, tx, &reportconfig.StoredReport{
			Id: report.ID, Namespace: report.Namespace, Slug: report.Slug, Title: report.Title,
			Description: namespaceOptionalDescription(report.Description), OwnerId: report.OwnerID,
			Status: report.Status, DefaultConnectorName: defaultConnector,
			ComponentScope: componentScope, ComponentName: componentName,
			CurrentDraftVersion: report.CurrentDraftVersion, Etag: &etag, UpdatedAt: &now,
			Has: &reportconfig.StoredReportHas{Id: true, Namespace: true, Slug: true,
				Title: true, Description: true, OwnerId: true, Status: true,
				DefaultConnectorName: true, ComponentScope: true, ComponentName: true,
				CurrentDraftVersion: true, Etag: true, UpdatedAt: true},
		})
		if err != nil {
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				return nil, &sdk.Error{Code: sdk.ErrorConflict, Message: "report etag does not match"}
			}
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
	sources, err := t.readMCPNameSources(ctx, reportID)
	if err != nil {
		return internal(err)
	}
	requested := map[string]bool{}
	for _, name := range names {
		key := strings.ToLower(name)
		if requested[key] {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("MCP tool name %q is duplicated in this component", name)}
		}
		requested[key] = true
	}
	for _, row := range sources {
		source := versionOptionalString(row.GeneratedDql)
		if strings.TrimSpace(source) == "" {
			source = versionOptionalString(row.AuthoredDql)
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
	var result []string
	const pageSize = 500
	for offset := 0; ; {
		items, err := t.readConnectorCatalog(ctx, connectorCatalogRequest{Status: "active", Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, internal(err)
		}
		for _, item := range items {
			result = append(result, item.Name)
		}
		if len(items) < pageSize {
			break
		}
		offset += len(items)
	}
	sort.Strings(result)
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
	generation, err := t.nextGenerationNo(ctx, tx)
	if err != nil {
		return 0, publicationState{}, false, time.Time{}, internal(err)
	}
	runtimeRevision := fmt.Sprintf("%s:%d:%d", in.ReportID, in.VersionNo, generation)
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no, source_revision, status, report_count, build_manifest_json, requested_by, requested_at) VALUES (?, ?, 'building', 0, '{}', ?, ?)`, generation, runtimeRevision, in.Input.RequestedBy, now); err != nil {
		return 0, publicationState{}, false, time.Time{}, classify(err, "runtime generation", fmt.Sprint(generation))
	}
	previous, hasPrevious, err := t.publicationSnapshot(ctx, tx, in.ReportID)
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

func (t *Transport) publicationSnapshot(ctx context.Context, tx *sql.Tx, reportID string) (publicationState, bool, error) {
	var value publicationState
	row, err := t.readPublicationRow(ctx, tx, reportID)
	if errors.Is(err, sql.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	value.activeVersion = int64(row.ActiveVersionNo)
	value.desiredGeneration = row.DesiredGeneration
	if row.DesiredVersionNo != nil {
		value.desiredVersion = sql.NullInt64{Int64: int64(*row.DesiredVersionNo), Valid: true}
	}
	if row.ActiveGeneration != nil {
		value.activeGeneration = sql.NullInt64{Int64: *row.ActiveGeneration, Valid: true}
	}
	value.status, value.specHash, value.publishedBy = row.PublicationStatus, row.SpecHash, row.PublishedBy
	if row.RuntimeRevision != nil {
		value.runtimeRevision = *row.RuntimeRevision
	}
	if row.PublishedAt != nil {
		value.publishedAt = *row.PublishedAt
	}
	if row.ActivatedAt != nil {
		value.activatedAt = *row.ActivatedAt
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
	previous, found, err := t.publicationSnapshot(ctx, tx, in.ReportID)
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
	generation, err := t.nextGenerationNo(ctx, tx)
	if err != nil {
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
	if reportID == "" {
		return mapReadError(sql.ErrNoRows, "publication", reportID)
	}
	value, err := t.readPublicationStatus(ctx, reportID)
	if err != nil {
		return mapReadError(err, "publication", reportID)
	}
	return assign(output, value)
}

func (t *Transport) runtimeStatus(ctx context.Context, output any) error {
	var value sdk.RuntimeStatus
	active, err := t.readActiveGeneration(ctx)
	if err != nil {
		return internal(err)
	}
	if active == nil {
		value.Status = "idle"
		return assign(output, &value)
	}
	value.ActiveGeneration, value.Status, value.ReportCount = active.GenerationNo, active.Status, active.ReportCount
	value.ActivatedAt = active.ActivatedAt
	if len(active.DiagnosticsJson) > 0 {
		_ = json.Unmarshal(active.DiagnosticsJson, &value.Diagnostics)
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
	result, err := t.readRuntimeReaderCatalog(ctx, generation)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}
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

	return result, nil
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
