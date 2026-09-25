package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/sdk"
	storedreader "github.com/viant/datly-studio/studio/authorization_predicates/store_read"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

type authorizationPredicateReader struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

// readAuthorizationPredicateRows invokes the generated, server-owned reader.
// Invoke has already checked the SDK authorization before reaching this method.
func (t *Transport) readAuthorizationPredicateRows(ctx context.Context, name string, byName bool, query, status string, limit, offset int) ([]*storedreader.StoredAuthorizationPredicate, error) {
	reader, err := t.authorizationPredicateReader()
	if err != nil {
		return nil, err
	}
	input := &storedreader.Input{}
	if byName {
		input.SetName(name)
	}
	pattern := ""
	if value := strings.TrimSpace(query); value != "" {
		pattern = "%" + strings.ToLower(value) + "%"
	}
	input.SetSearchPattern(pattern)
	input.SetStatus(status)
	input.SetOrderBy("updated_at DESC, name ASC")
	input.SetLimit(limit)
	input.SetOffset(offset)
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedreader.Output)
	if !ok {
		return nil, fmt.Errorf("authorization predicate reader returned %T", value)
	}
	if len(output.AuthorizationPredicates) > limit {
		return nil, fmt.Errorf("authorization predicate reader exceeded page limit")
	}
	return output.AuthorizationPredicates, nil
}

func (t *Transport) authorizationPredicateReader() (*authorizationPredicateReader, error) {
	t.predicateReaderMu.Lock()
	defer t.predicateReaderMu.Unlock()
	if t.predicateReader != nil {
		return t.predicateReader, nil
	}
	resources := resource.New()
	if err := resources.Register(storedreader.AuthorizationPredicateDatlyResourceNamespace, storedreader.AuthorizationPredicateDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	holder := reflect.TypeOf(storedreader.AuthorizationPredicateComponent{})
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, fmt.Errorf("authorization predicate store reader has no component contract")
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return nil, fmt.Errorf("authorization predicate store reader metadata: %w", err)
	}
	if !present {
		return nil, fmt.Errorf("authorization predicate store reader has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: "store_read", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(storedreader.Input{}), reflect.TypeOf(storedreader.Output{}))
	if err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(storedreader.Input{}), OutputType: reflect.TypeOf(storedreader.Output{}), Resources: resources})
	if err != nil {
		return nil, err
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		return nil, err
	}
	registration, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	t.predicateReader = &authorizationPredicateReader{runtime: runtime, target: target}
	return t.predicateReader, nil
}

// Close releases lazily hosted Datly components after the serving host has
// drained requests. The database remains caller-owned.
func (t *Transport) Close(ctx context.Context) error {
	t.predicateReaderMu.Lock()
	var result error
	if t.predicateReader != nil {
		result = errors.Join(result, t.predicateReader.runtime.Shutdown(ctx))
		t.predicateReader = nil
	}
	t.predicateReaderMu.Unlock()
	t.capabilityReaderMu.Lock()
	if t.capabilityReader != nil {
		result = errors.Join(result, t.capabilityReader.Close(ctx))
		t.capabilityReader = nil
	}
	t.capabilityReaderMu.Unlock()
	t.aclReaderMu.Lock()
	if t.aclReader != nil {
		result = errors.Join(result, t.aclReader.runtime.Shutdown(ctx))
		t.aclReader = nil
	}
	t.aclReaderMu.Unlock()
	t.publicationReaderMu.Lock()
	if t.publicationReader != nil {
		result = errors.Join(result, t.publicationReader.runtime.Shutdown(ctx))
		t.publicationReader = nil
	}
	t.publicationReaderMu.Unlock()
	return result
}

func (t *Transport) authorizationPredicateFromRow(row *storedreader.StoredAuthorizationPredicate) (*sdk.AuthorizationPredicate, error) {
	if row == nil || row.CreatedAt == nil || row.UpdatedAt == nil {
		return nil, fmt.Errorf("authorization predicate reader returned an incomplete row")
	}
	value := &sdk.AuthorizationPredicate{
		Name: row.Name, Title: row.Title, PackagePath: row.PackagePath, TypeName: row.TypeName,
		OwnerID: row.OwnerId, Status: row.Status, ETag: row.Etag,
		CreatedAt: *row.CreatedAt, UpdatedAt: *row.UpdatedAt,
	}
	if row.Description != nil {
		value.Description = *row.Description
	}
	if row.SqlScopeJson != nil && strings.TrimSpace(*row.SqlScopeJson) != "" {
		var metadata sdk.AuthorizationPredicateSQLMetadata
		if err := json.Unmarshal([]byte(*row.SqlScopeJson), &metadata); err != nil {
			return nil, err
		}
		value.Alias, value.Columns = metadata.Alias, metadata.Columns
	}
	value.Linked = t.Predicates.Contains(value.PackagePath, value.TypeName)
	return value, nil
}
