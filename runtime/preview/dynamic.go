// Package preview executes a versioned dynamic Datly reader for Studio preview.
package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly/typecatalog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/bindly"
	bindresource "github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	studiors "github.com/viant/datly-studio/runtime/resources"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	mcpresource "github.com/viant/datly/mcp/resource"
	"github.com/viant/datly/report"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/sqlx"
	xhandler "github.com/viant/xdatly/handler"
	"golang.org/x/mod/modfile"
)

// Dynamic executes dynamic reader versions using Datly's runtime-contract
// materialization. It is an SDK preview executor, not a SQL escape hatch.
type Dynamic struct {
	Types       *typecatalog.Catalog
	StudioDB    *sql.DB
	RootDir     string
	Secrets     connectorsecret.Resolver
	Definitions *DefinitionReader
	Connectors  *ConnectorReader
}

// Validate materializes the same runtime contract and reader execution used by
// preview without issuing a business query. Connector, type, resource, route,
// predicate, MCP, and SQL-reader wiring must all be ready before publication.
func (d Dynamic) Validate(ctx context.Context, reportID string, versionNo int) error {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return err
	}
	sources, err := openSources(ctx, definition, definition.DQL)
	if err != nil {
		return err
	}
	defer sources.Close()
	contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, d.RootDir, &transcribe.Source{Types: d.Types,
		Scope: definition.Scope, Name: definition.Name, Text: definition.DQL, Connector: definition.Connector,
		Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections),
	})
	if err != nil {
		return fmt.Errorf("compile runtime contract: %w", err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{
		Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType,
		Types: contract.Types, Resources: contract.Resources,
	})
	if err != nil {
		return fmt.Errorf("build runtime artifact: %w", err)
	}
	if artifact.ReaderCompilation() == nil {
		return errors.New("reader compilation is unavailable")
	}
	if _, err = artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sources.SQL}); err != nil {
		return fmt.Errorf("initialize runtime reader: %w", err)
	}
	if _, err = (mcpresource.Publisher{Resources: definition.ResourceStore}).Compile(ctx, definition.ResourceFolders); err != nil {
		return fmt.Errorf("compile MCP resources and skills: %w", err)
	}
	return nil
}

// TestSQL compiles browser-authored SQL into a bounded transient Datly reader.
// It intentionally reuses the normal runtime-contract and typed reader path.
func (d Dynamic) TestSQL(ctx context.Context, connector *sdk.Connector, request sdk.SQLTestInput) (*sdk.SQLTestResult, error) {
	if connector == nil || strings.TrimSpace(connector.Name) == "" || strings.TrimSpace(connector.DSNTemplate) == "" {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "active connector is required"}
	}
	sqlText := strings.TrimSpace(request.SQL)
	if sqlText == "" {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "SQL is required"}
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	token := safeViewToken(connector.Name)
	modulePath, err := previewModulePath(d.RootDir)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "resolve Studio module for SQL test", Cause: err}
	}
	resolvedDSN, err := connectorsecret.Resolve(ctx, d.Secrets, connector.DSNTemplate, connector.SecretRef)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "resolve connector secret", Cause: err}
	}
	definition := &definition{
		Scope: modulePath + "/dynamic/sqltest/" + token,
		Name:  "preview", Connector: connector.Name, Driver: connector.Driver, DSN: resolvedDSN, Options: connector.Options, Secrets: d.Secrets,
	}
	dql := fmt.Sprintf(`#package('%s')

#setting($_ = $connector('%s'))
#setting($_ = $route('/v1/studio/sqltest/%s', 'GET'))
#setting($_ = $case_format('lc'))

#define($_ = $Rows<[]*SQLTestRow>(output/view))

SELECT result.*,
       type(result, 'SQLTestRow'),
       set_limit(result, %d)
FROM (%s) result`, definition.Scope, connector.Name, token, limit, sqlText)
	preview, err := d.execute(ctx, definition, dql, sdk.PreviewInput{Limit: limit})
	if err != nil {
		return nil, err
	}
	return &sdk.SQLTestResult{Data: preview.Data, Duration: preview.Duration}, nil
}

func previewModulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	modulePath := strings.TrimSpace(modfile.ModulePath(data))
	if modulePath == "" {
		return "", errors.New("go.mod has no module path")
	}
	return modulePath, nil
}

func (d Dynamic) Execute(ctx context.Context, reportID string, versionNo int, request sdk.PreviewInput) (*sdk.PreviewResult, error) {
	if d.StudioDB == nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Studio database is unavailable"}
	}
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(definition.DQL) == "" {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader version has no DQL source"}
	}
	return d.execute(ctx, definition, definition.DQL, request)
}

func (d Dynamic) TestView(ctx context.Context, reportID string, versionNo int, view string, request sdk.ViewTestInput) (*sdk.ViewTestResult, error) {
	if d.StudioDB == nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Studio database is unavailable"}
	}
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	dql, err := testViewDQL(ctx, definition, view)
	if err != nil {
		return nil, err
	}
	preview, err := d.execute(ctx, definition, dql, sdk.PreviewInput{Input: request.Input, Limit: request.Limit})
	if err != nil {
		return nil, err
	}
	return &sdk.ViewTestResult{View: view, Data: preview.Data, Duration: preview.Duration, Diagnostics: preview.Diagnostics, Evidence: preview.Evidence}, nil
}

func (d Dynamic) TestRelation(ctx context.Context, reportID string, versionNo int, relationName string, request sdk.ViewTestInput) (*sdk.RelationTestResult, error) {
	if d.StudioDB == nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Studio database is unavailable"}
	}
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	var result *sdk.RelationTestResult
	preview, err := d.executeObserved(ctx, definition, definition.DQL, sdk.PreviewInput{Input: request.Input, Limit: request.Limit}, func(data any, contract *transcribe.RuntimeContract) error {
		resolved, resolveErr := resolveRelationPath(contract.Component.RootView, relationName)
		if resolveErr != nil {
			return resolveErr
		}
		rows := rootTypedRows(data, contract.Component)
		for _, ancestor := range resolved.path[:len(resolved.path)-1] {
			rows = attachedTypedRows(rows, ancestor)
		}
		parentRows := len(rows)
		matched, attached := 0, 0
		for _, row := range rows {
			children := relationTypedRows(row, resolved.relation)
			if len(children) > 0 {
				matched++
				attached += len(children)
			}
		}
		result = &sdk.RelationTestResult{Relation: resolved.relation.Name, ParentView: viewEvidenceName(resolved.parent), ChildView: viewEvidenceName(resolved.relation.View), Kind: string(resolved.relation.Kind), Cardinality: string(resolved.relation.Cardinality), ParentRows: parentRows, MatchedParents: matched, UnmatchedParents: parentRows - matched, AttachedChildren: attached}
		for _, key := range resolved.relation.On {
			if key != nil {
				result.Keys = append(result.Keys, sdk.RelationKeyEvidence{ParentColumn: key.ParentColumn, ChildColumn: key.ChildColumn})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "relation test evidence was not produced"}
	}
	result.Data, result.Duration, result.Evidence, result.Diagnostics = preview.Data, preview.Duration, preview.Evidence, preview.Diagnostics
	return result, nil
}

// TestCompose derives and invokes Datly's real cube-compose component from the
// persisted reader contract. Frame SQL substitution and budgets remain owned by
// Datly's report package.
func (d Dynamic) TestCompose(ctx context.Context, reportID string, versionNo int, request sdk.CubeComposeTestInput) (*sdk.CubeComposeTestResult, error) {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	sources, err := openSources(ctx, definition, definition.DQL)
	if err != nil {
		return nil, err
	}
	defer sources.Close()
	contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, d.RootDir, &transcribe.Source{Types: d.Types,
		Scope: definition.Scope, Name: definition.Name, Text: definition.DQL, Connector: definition.Connector,
		Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections),
	})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "compile cube source: " + err.Error(), Cause: err}
	}
	if contract.Component == nil || contract.Component.Settings == nil || contract.Component.Settings.Report == nil || contract.Component.Settings.Report.Compose == nil || !contract.Component.Settings.Report.Compose.Enabled {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader version has no enabled cube composition"}
	}
	compilation, err := report.NewProjectCompiler(report.ProjectConfig{Types: contract.Types}).CompileArtifacts([]bootstrap.ArtifactInput{{
		Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType,
		DirectViewField: outputViewField(contract.Component), Types: contract.Types, Resources: contract.Resources,
	}})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "derive cube composition: " + err.Error(), Cause: err}
	}
	sourceSQL := sources.SQL
	registered, err := compilation.RuntimeComponents(ctx, report.RuntimeConfigureFunc(func(_ context.Context, artifact *report.ComponentArtifact) (report.RuntimeCapabilities, error) {
		if artifact.IsReport() {
			return report.RuntimeCapabilities{}, nil
		}
		compiled := artifact.ReaderCompilation()
		if compiled == nil {
			return report.RuntimeCapabilities{}, errors.New("cube source reader compilation is unavailable")
		}
		reader, buildErr := compiled.NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sourceSQL})
		return report.RuntimeCapabilities{Reader: reader}, buildErr
	}))
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "configure cube composition runtime", Cause: err}
	}
	runtime, err := druntime.NewRuntime(registered)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize cube composition runtime", Cause: err}
	}
	defer runtime.Shutdown(ctx)
	var composeArtifact *report.ComponentArtifact
	for _, artifact := range compilation.Artifacts() {
		component := artifact.Component()
		if !artifact.IsReport() || component == nil {
			continue
		}
		for _, route := range component.Routes {
			if route != nil && strings.HasSuffix(strings.TrimSuffix(route.Path, "/"), "/cube/compose") {
				composeArtifact = artifact
				break
			}
		}
	}
	if composeArtifact == nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "derived cube composition component is unavailable"}
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "encode cube composition input", Cause: err}
	}
	input := reflect.New(composeArtifact.InputType()).Interface()
	if err = json.Unmarshal(payload, input); err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "cube composition input does not match derived contract", Cause: err}
	}
	component := composeArtifact.Component()
	if len(component.Routes) == 0 || component.Routes[0] == nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "cube composition route is unavailable"}
	}
	started := time.Now()
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: dexec.ComponentTarget{
		Component: component.Key, Route: spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path},
	}, Input: input})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "execute cube composition: " + err.Error(), Cause: err}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "encode cube composition result", Cause: err}
	}
	return &sdk.CubeComposeTestResult{Data: encoded, Duration: time.Since(started)}, nil
}

func outputViewField(component *spec.Component) string {
	if component == nil {
		return ""
	}
	for _, parameter := range component.Parameters {
		if parameter != nil && strings.EqualFold(parameter.Source.Kind, "output") && strings.EqualFold(parameter.Source.Name, "view") {
			return parameter.Name
		}
	}
	return ""
}

// Warmup invokes Datly's ReaderWarmer against the persisted reader component.
// It is server-owned: callers choose only a version, never cache paths, cases,
// connector credentials, or a background execution context.
func (d Dynamic) Warmup(ctx context.Context, reportID string, versionNo int) (*sdk.WarmupResult, error) {
	definition, err := d.definition(ctx, reportID, versionNo)
	if err != nil {
		return nil, err
	}
	sources, err := openSources(ctx, definition, definition.DQL)
	if err != nil {
		return nil, err
	}
	defer sources.Close()
	contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, d.RootDir, &transcribe.Source{Types: d.Types, Scope: definition.Scope, Name: definition.Name, Text: definition.DQL, Connector: definition.Connector, Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections)})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "compile dynamic reader: " + err.Error(), Cause: err}
	}
	if contract.Component == nil || contract.Component.RootView == nil || contract.Component.CacheWarmup() == nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader version has no authored cache warmup policy"}
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "build dynamic reader artifact", Cause: err}
	}
	reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sources.SQL})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize dynamic reader", Cause: err}
	}
	registered, err := artifact.Registration(druntime.RegisteredComponent{Reader: reader})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "register dynamic warmup reader", Cause: err}
	}
	runtime, err := druntime.NewRuntime([]*druntime.RegisteredComponent{registered})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize dynamic warmup runtime", Cause: err}
	}
	defer runtime.Shutdown(ctx)
	if len(contract.Component.Routes) == 0 || contract.Component.Routes[0] == nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader version has no warmup route"}
	}
	route := contract.Component.Routes[0]
	target := dexec.ComponentTarget{Component: contract.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
	operation, err := runtime.NewWarmup(target)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic reader does not support cache warmup", Cause: err}
	}
	planned, err := operation.PlannedCases()
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "plan dynamic cache warmup: " + err.Error(), Cause: err}
	}
	settings := contract.Component.CacheWarmup()
	cache := contract.Component.Settings.Cache
	provider := cache.Provider
	if strings.TrimSpace(provider) == "" {
		provider = "afs"
	}
	result := &sdk.WarmupResult{Status: "running", PlannedCases: planned, Target: sdk.WarmupTarget{View: contract.Component.RootView.Name, CacheName: cache.Name, CacheProvider: provider, ConnectorName: settings.Connector, IndexColumn: settings.IndexColumn, IndexParameter: settings.IndexParameter}}
	if settings.MaxCases != nil {
		value := *settings.MaxCases
		result.MaxCases = &value
	}
	if settings.Limit != nil {
		value := *settings.Limit
		result.RowLimit = &value
	}
	started := time.Now()
	entries, err := operation.Run(ctx)
	result.Entries = entries
	result.Duration = time.Since(started)
	if err != nil {
		return result, &sdk.Error{Code: sdk.ErrorInternal, Message: "execute dynamic cache warmup: " + err.Error(), Cause: err}
	}
	result.Status = "completed"
	result.CompletedCases = planned
	return result, nil
}

func (d Dynamic) execute(ctx context.Context, definition *definition, dql string, request sdk.PreviewInput) (*sdk.PreviewResult, error) {
	return d.executeObserved(ctx, definition, dql, request, nil)
}

func (d Dynamic) executeObserved(ctx context.Context, definition *definition, dql string, request sdk.PreviewInput, observe func(any, *transcribe.RuntimeContract) error) (*sdk.PreviewResult, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	sources, err := openSources(ctx, definition, dql)
	if err != nil {
		return nil, err
	}
	defer sources.Close()
	contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, d.RootDir, &transcribe.Source{Types: d.Types,
		Scope: definition.Scope, Name: definition.Name, Text: dql, Connector: definition.Connector,
		Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections),
	})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "compile dynamic reader: " + err.Error(), Cause: err}
	}
	input := reflect.New(contract.InputType).Interface()
	if len(request.Input) != 0 {
		if err = json.Unmarshal(request.Input, input); err != nil {
			return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "preview input does not match reader contract", Cause: err}
		}
		markSuppliedFields(input, request.Input)
	}
	if credential, ok := sdk.VerifiedCredentialFromContext(ctx); ok {
		hydrateVerifiedClaims(input, credential.Claims)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "build dynamic reader artifact", Cause: err}
	}
	reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sources.SQL})
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize dynamic reader", Cause: err}
	}
	injector, err := bindly.NewInjector()
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize dynamic reader binder", Cause: err}
	}
	binder := previewBinder{delegate: rhandler.NewBinder(injector, input), input: input}
	started := time.Now()
	data, err := reader.Read(ctx, input, binder, sqlx.ParameterResolver(artifact.Input.Resolver(input)))
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "execute dynamic reader: " + err.Error(), Cause: err}
	}
	data, returnedRows, truncated := limitPreviewRows(data, limit)
	if observe != nil {
		if err = observe(data, contract); err != nil {
			return nil, err
		}
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "encode dynamic reader preview", Cause: err}
	}
	for len(encoded) > maxPreviewBytes && returnedRows > 1 {
		data, returnedRows, _ = limitPreviewRows(data, (returnedRows+1)/2)
		truncated = true
		encoded, err = json.Marshal(data)
		if err != nil {
			return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "encode bounded dynamic reader preview", Cause: err}
		}
	}
	if len(encoded) > maxPreviewBytes {
		return nil, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "preview result exceeds the encoded response budget"}
	}
	evidence := sdk.ExecutionEvidence{ReportID: definition.ReportID, VersionNo: definition.VersionNo, SourceRevision: definition.SourceRevision, Connector: definition.Connector, Limit: limit, ReturnedRows: returnedRows, EncodedBytes: len(encoded), Truncated: truncated}
	return &sdk.PreviewResult{Data: encoded, Duration: time.Since(started), Evidence: evidence}, nil
}

const maxPreviewBytes = 2 << 20

func limitPreviewRows(data any, limit int) (any, int, bool) {
	if limit < 0 {
		limit = 0
	}
	value := reflect.ValueOf(data)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return data, 0, false
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return data, 0, false
	}
	if value.Kind() == reflect.Slice {
		count := value.Len()
		if count > limit {
			return value.Slice(0, limit).Interface(), limit, true
		}
		return data, count, false
	}
	if value.Kind() != reflect.Struct {
		return data, 0, false
	}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		if field.Kind() != reflect.Slice || !field.CanSet() {
			continue
		}
		count := field.Len()
		if count > limit {
			field.Set(field.Slice(0, limit))
			return data, limit, true
		}
		return data, count, false
	}
	return data, 0, false
}

type resolvedRelation struct {
	parent   *spec.View
	relation *spec.Relation
	path     []*spec.Relation
}

func resolveRelationPath(root *spec.View, identity string) (*resolvedRelation, error) {
	var matches []*resolvedRelation
	var walk func(*spec.View, []*spec.Relation)
	walk = func(parent *spec.View, path []*spec.Relation) {
		if parent == nil {
			return
		}
		for _, relation := range parent.Relations {
			if relation == nil || relation.View == nil {
				continue
			}
			next := append(append([]*spec.Relation(nil), path...), relation)
			if relationIdentityMatches(identity, parent, relation) {
				matches = append(matches, &resolvedRelation{parent: parent, relation: relation, path: next})
			}
			walk(relation.View, next)
		}
	}
	walk(root, nil)
	if len(matches) == 0 {
		return nil, &sdk.Error{Code: sdk.ErrorNotFound, Message: fmt.Sprintf("relation %q was not found", identity)}
	}
	if len(matches) > 1 {
		return nil, &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("relation %q is ambiguous; use parent->child identity", identity)}
	}
	return matches[0], nil
}

func relationIdentityMatches(identity string, parent *spec.View, relation *spec.Relation) bool {
	parts := strings.Split(identity, "->")
	if len(parts) == 2 {
		return matchesIdentity(parts[0], parent.Name, parent.Namespace) && matchesIdentity(parts[1], relation.Name, relation.Holder, relation.View.Name, relation.View.Namespace)
	}
	return matchesIdentity(identity, relation.Name, relation.Holder, relation.View.Name, relation.View.Namespace)
}

func rootTypedRows(data any, component *spec.Component) []reflect.Value {
	candidates := []string{}
	if component != nil {
		for _, parameter := range component.Parameters {
			if parameter != nil && strings.EqualFold(parameter.Source.Kind, "output") && strings.EqualFold(parameter.Source.Name, "view") {
				candidates = append(candidates, parameter.Name)
			}
		}
		if component.RootView != nil {
			candidates = append(candidates, component.RootView.Name, component.RootView.Namespace)
		}
	}
	return typedRows(reflect.ValueOf(data), candidates...)
}

func attachedTypedRows(parents []reflect.Value, relation *spec.Relation) []reflect.Value {
	var result []reflect.Value
	for _, parent := range parents {
		result = append(result, relationTypedRows(parent, relation)...)
	}
	return result
}

func relationTypedRows(parent reflect.Value, relation *spec.Relation) []reflect.Value {
	if relation == nil || relation.View == nil {
		return nil
	}
	return typedRows(parent, relation.Holder, relation.Name, relation.View.Name, relation.View.Namespace)
}

func typedRows(value reflect.Value, candidates ...string) []reflect.Value {
	value = dereferenceValue(value)
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		result := make([]reflect.Value, 0, value.Len())
		for index := 0; index < value.Len(); index++ {
			result = append(result, value.Index(index))
		}
		return result
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	for _, candidate := range candidates {
		identity := normalizedIdentity(candidate)
		if identity == "" {
			continue
		}
		for index := 0; index < value.NumField(); index++ {
			fieldType := value.Type().Field(index)
			if normalizedIdentity(fieldType.Name) != identity {
				continue
			}
			field := dereferenceValue(value.Field(index))
			if !field.IsValid() {
				return nil
			}
			if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				result := make([]reflect.Value, 0, field.Len())
				for item := 0; item < field.Len(); item++ {
					result = append(result, field.Index(item))
				}
				return result
			}
			if field.Kind() == reflect.Struct {
				return []reflect.Value{field}
			}
			return nil
		}
	}
	return nil
}

func dereferenceValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func matchesIdentity(value string, candidates ...string) bool {
	want := normalizedIdentity(value)
	for _, candidate := range candidates {
		if want != "" && want == normalizedIdentity(candidate) {
			return true
		}
	}
	return false
}

func normalizedIdentity(value string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result.WriteRune(character)
		}
	}
	return result.String()
}

func viewEvidenceName(view *spec.View) string {
	if view == nil {
		return ""
	}
	if strings.EqualFold(view.Name, "reader") && strings.TrimSpace(view.Namespace) != "" {
		return view.Namespace
	}
	if strings.TrimSpace(view.Name) != "" {
		return view.Name
	}
	return view.Namespace
}

func hydrateVerifiedClaims(input, claims any) {
	target := reflect.ValueOf(input)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return
	}
	target = target.Elem()
	field := target.FieldByName("Jwt")
	value := reflect.ValueOf(claims)
	if field.IsValid() && field.CanSet() && value.IsValid() && value.Type().AssignableTo(field.Type()) {
		field.Set(value)
	}
}

func testViewDQL(ctx context.Context, definition *definition, viewName string) (string, error) {
	service := readerbuilder.New(readerbuilder.Config{Scope: definition.Scope, Name: definition.Name, AvailableConnectors: []string{definition.Connector}})
	response := service.Apply(ctx, readerbuilder.Request{DQL: definition.DQL, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if response.Structure == nil {
		return "", &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader view structure is unavailable"}
	}
	packagePath := strings.TrimSpace(response.Structure.Component.TypeContext.PackagePath)
	if packagePath == "" {
		return "", &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reader component package is unavailable"}
	}
	var sourceSpanStart, sourceSpanEnd int
	found := false
	for _, view := range response.Structure.Views {
		if strings.EqualFold(view.Name, viewName) {
			sourceSpanStart, sourceSpanEnd, found = view.SourceSpan.Start, view.SourceSpan.End, true
			break
		}
	}
	if !found || sourceSpanStart < 0 || sourceSpanEnd > len(definition.DQL) || sourceSpanStart >= sourceSpanEnd {
		return "", &sdk.Error{Code: sdk.ErrorNotFound, Message: "reader view not found"}
	}
	var declarations []string
	for _, declaration := range response.Structure.Declarations {
		if declaration.Parameter == nil || declaration.Parameter.EmitOutput || strings.EqualFold(declaration.Parameter.Source.Kind, "output") {
			continue
		}
		if declaration.Span.Start >= 0 && declaration.Span.End <= len(definition.DQL) && declaration.Span.Start < declaration.Span.End {
			declarations = append(declarations, definition.DQL[declaration.Span.Start:declaration.Span.End])
		}
	}
	imports := ""
	for _, line := range strings.Split(definition.DQL, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#import(") {
			imports += line + "\n"
		}
	}
	token := safeViewToken(viewName)
	typeName := "Preview" + strings.ToUpper(token[:1]) + token[1:]
	inner := definition.DQL[sourceSpanStart:sourceSpanEnd]
	return fmt.Sprintf("#package('%s/viewtest/%s')\n%s#setting($_ = $connector('%s'))\n#setting($_ = $route('/v1/studio/viewtest/%s','GET'))\n%s#define($_ = $Rows<[]*%s>(output/view))\nSELECT tested.*, type(tested, '%s')\nFROM (%s) tested", packagePath, token, imports, definition.Connector, token, strings.Join(declarations, "\n")+"\n", typeName, typeName, inner), nil
}

func safeViewToken(value string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(value) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' {
			result.WriteRune(character)
		}
	}
	if result.Len() == 0 {
		return "view"
	}
	return result.String()
}

func markSuppliedFields(input any, raw json.RawMessage) {
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return
	}
	target := reflect.ValueOf(input)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return
	}
	target = target.Elem()
	marker := target.FieldByName("Has")
	if !marker.IsValid() || marker.Kind() != reflect.Pointer {
		return
	}
	if marker.IsNil() {
		marker.Set(reflect.New(marker.Type().Elem()))
	}
	for index := 0; index < target.NumField(); index++ {
		field := target.Type().Field(index)
		if field.Name == "Has" {
			continue
		}
		for key := range values {
			if strings.EqualFold(key, field.Name) {
				if supplied := marker.Elem().FieldByName(field.Name); supplied.IsValid() && supplied.CanSet() && supplied.Kind() == reflect.Bool {
					supplied.SetBool(true)
				}
			}
		}
	}
}

type previewBinder struct {
	delegate *rhandler.Binder
	input    any
}

func (b previewBinder) Bind(ctx context.Context, target any) error {
	return b.delegate.Bind(ctx, target)
}

func (b previewBinder) Lookup(ctx context.Context, key xhandler.ValueKey) (any, bool, error) {
	if key == xhandler.InputKey {
		return b.input, true, nil
	}
	return b.delegate.Lookup(ctx, key)
}

type definition struct {
	Scope, Name, Connector, Driver, DSN, SecretRef, DQL string
	ReportID                                            string
	VersionNo                                           int
	SourceRevision                                      int64
	SpecHash                                            string
	Secrets                                             connectorsecret.Resolver
	Options                                             json.RawMessage
	Connectors                                          []connectorDefinition
	Resources                                           *bindresource.Store
	ResourceStore                                       *bindresource.Store
	ResourceFolders                                     []mcpresource.Folder
}

type connectorDefinition struct {
	Name, Driver, DSN, SecretRef string
	Options                      json.RawMessage
}

func (d Dynamic) definition(ctx context.Context, reportID string, versionNo int) (*definition, error) {
	var result definition
	result.ReportID, result.VersionNo = reportID, versionNo
	result.Secrets = d.Secrets
	reader := d.Definitions
	if reader == nil {
		var setupErr error
		reader, setupErr = NewDefinitionReader(d.StudioDB)
		if setupErr != nil {
			return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize reader preview definition", Cause: setupErr}
		}
		defer reader.Close(context.Background())
	}
	row, err := reader.Get(ctx, reportID, versionNo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &sdk.Error{Code: sdk.ErrorNotFound, Message: "reader version not found"}
	}
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "load reader preview definition", Cause: err}
	}
	result.Scope, result.Name, result.Connector = row.ComponentScope, row.ComponentName, row.DefaultConnectorName
	result.Driver, result.DSN, result.SecretRef = row.Driver, *row.DsnTemplate, row.SecretRef
	result.Options = append(json.RawMessage(nil), row.OptionsJson...)
	result.SourceRevision, result.SpecHash = row.SourceRevision, row.SpecHash
	result.DQL = row.GeneratedDql
	if strings.TrimSpace(result.DQL) == "" {
		result.DQL = row.AuthoredDql
	}
	result.DSN, err = connectorsecret.Resolve(ctx, d.Secrets, result.DSN, result.SecretRef)
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "resolve reader connector secret", Cause: err}
	}
	if strings.TrimSpace(result.DSN) == "" {
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "reader connector has no DSN"}
	}
	loaded, loadErr := studiors.Load(ctx, d.StudioDB, []studiors.Version{{ReportID: reportID, VersionNo: versionNo}})
	if loadErr != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "load reader resources", Cause: loadErr}
	}
	result.Resources = loaded.ByVersion[studiors.Version{ReportID: reportID, VersionNo: versionNo}]
	result.ResourceStore, result.ResourceFolders = loaded.Store, loaded.Folders
	result.Connectors, err = d.activeConnectors(ctx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (d Dynamic) activeConnectors(ctx context.Context) ([]connectorDefinition, error) {
	reader := d.Connectors
	if reader == nil {
		var err error
		reader, err = NewConnectorReader(d.StudioDB)
		if err != nil {
			return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "initialize active Studio connectors", Cause: err}
		}
		defer reader.Close(context.Background())
	}
	var result []connectorDefinition
	var err error
	if principal, ok := sdk.PrincipalFromContext(ctx); ok {
		result, err = reader.ListForPrincipal(ctx, principal.Subject)
	} else {
		result, err = reader.ListAll(ctx)
	}
	if err != nil {
		return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "load active Studio connectors", Cause: err}
	}
	return result, nil
}

type sources struct {
	SQL         *dsql.SQLComponent
	Connections column.Connections
	opened      []*sql.DB
}

func (s *sources) Close() {
	for _, db := range s.opened {
		_ = db.Close()
	}
}

// openSources derives connector usage from Datly's compiled view metadata and
// binds exact Studio connector names to both transcription and execution. No
// connector aliases or per-view Studio registry are maintained here.
func openSources(ctx context.Context, definition *definition, dql string) (*sources, error) {
	available := map[string]connectorDefinition{}
	for _, item := range definition.Connectors {
		available[item.Name] = item
	}
	if _, ok := available[definition.Connector]; !ok && strings.TrimSpace(definition.DSN) != "" {
		available[definition.Connector] = connectorDefinition{Name: definition.Connector, Driver: definition.Driver, DSN: definition.DSN, SecretRef: definition.SecretRef, Options: definition.Options}
	}
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	required := requiredConnectors(ctx, definition, dql, names)
	result := &sources{Connections: column.Connections{}}
	for _, name := range required {
		item, ok := available[name]
		if !ok {
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("Datly reader requires active Studio connector %q", name)}
		}
		resolvedDSN, err := connectorsecret.Resolve(ctx, definition.Secrets, item.DSN, item.SecretRef)
		if err != nil {
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("resolve Studio connector %q secret", name), Cause: err}
		}
		if strings.TrimSpace(resolvedDSN) == "" {
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("Studio connector %q has no DSN", name)}
		}
		db, err := sql.Open(normalizeDriver(item.Driver), resolvedDSN)
		if err != nil {
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("open Studio connector %q", name), Cause: err}
		}
		if err = connectorinit.Configure(ctx, db, item.Driver, item.Options); err != nil {
			_ = db.Close()
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("configure Studio connector %q", name), Cause: err}
		}
		result.opened = append(result.opened, db)
		result.Connections[name] = db
		if name == definition.Connector {
			result.SQL = &dsql.SQLComponent{DB: db}
		}
	}
	if result.SQL == nil {
		result.Close()
		return nil, &sdk.Error{Code: sdk.ErrorUnavailable, Message: fmt.Sprintf("default Studio connector %q is not active", definition.Connector)}
	}
	for name, db := range result.Connections {
		if err := result.SQL.RegisterConnector(name, db); err != nil {
			result.Close()
			return nil, &sdk.Error{Code: sdk.ErrorInternal, Message: "register Datly SQL connector", Cause: err}
		}
	}
	return result, nil
}

func requiredConnectors(ctx context.Context, definition *definition, dql string, available []string) []string {
	required := map[string]bool{definition.Connector: true}
	inspection := readerbuilder.New(readerbuilder.Config{Scope: definition.Scope, Name: definition.Name, AvailableConnectors: available}).Apply(ctx, readerbuilder.Request{DQL: dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if inspection.Structure != nil {
		for _, occurrence := range inspection.Structure.Functions {
			if strings.EqualFold(occurrence.Name, "use_connector") && len(occurrence.Args) > 1 {
				required[trimDQLString(occurrence.Args[1])] = true
			}
		}
		if component := inspection.Structure.Component; component != nil {
			if component.Settings != nil && strings.TrimSpace(component.Settings.DefaultConnector) != "" {
				required[component.Settings.DefaultConnector] = true
			}
			collectViewConnectors(component.RootView, required)
			for _, view := range component.Views {
				collectViewConnectors(view, required)
			}
		}
	}
	result := make([]string, 0, len(required))
	for name := range required {
		if strings.TrimSpace(name) != "" {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func collectViewConnectors(view *spec.View, target map[string]bool) {
	if view == nil {
		return
	}
	if view.Source != nil && view.Source.Bindings != nil && strings.TrimSpace(view.Source.Bindings.Connector) != "" {
		target[view.Source.Bindings.Connector] = true
	}
	for _, relation := range view.Relations {
		if relation != nil {
			collectViewConnectors(relation.View, target)
		}
	}
}

func trimDQLString(value string) string {
	return strings.Trim(strings.TrimSpace(value), "'\"")
}

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite3":
		return "sqlite"
	case "viant/bigquery":
		return "bigquery"
	case "pg", "pgx":
		return "postgres"
	default:
		return strings.TrimSpace(driver)
	}
}
