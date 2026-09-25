package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/viant/datly-studio/sdk"
	storedwriter "github.com/viant/datly-studio/studio/authorization_predicates/store_write"
)

func (t *Transport) createAuthorizationPredicate(ctx context.Context, input, output any) error {
	var in sdk.CreateAuthorizationPredicateInput
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	in.Name, in.Title, in.PackagePath, in.TypeName = strings.TrimSpace(in.Name), strings.TrimSpace(in.Title), strings.TrimSpace(in.PackagePath), strings.TrimSpace(in.TypeName)
	if !validPredicateName(in.Name) || in.Title == "" || !t.Predicates.Contains(in.PackagePath, in.TypeName) {
		return invalid(errors.New("authorization predicate requires a canonical name and configured package/type link"))
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	now := t.now()
	initialETag := 1
	scopeJSON, err := authorizationPredicateScopeJSON(in.Alias, in.Columns)
	if err != nil {
		return invalid(err)
	}
	if _, err := t.authorizationPredicateValue(ctx, in.Name); err == nil {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "authorization predicate already exists"}
	} else {
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
			return err
		}
	}
	row := &storedwriter.StoredAuthorizationPredicate{Name: in.Name, Title: in.Title,
		Description: predicateOptionalString(in.Description), PackagePath: in.PackagePath, TypeName: in.TypeName,
		SqlScopeJson: predicateOptionalString(scopeJSON), OwnerId: principal.Subject, Status: "active",
		Etag: &initialETag, CreatedAt: &now, UpdatedAt: &now,
		Has: &storedwriter.StoredAuthorizationPredicateHas{Name: true, Title: true, Description: true,
			PackagePath: true, TypeName: true, SqlScopeJson: true, OwnerId: true, Status: true,
			Etag: true, CreatedAt: true, UpdatedAt: true}}
	if err := t.writeAuthorizationPredicate(ctx, "post", row); err != nil {
		if _, readErr := t.authorizationPredicateValue(ctx, in.Name); readErr == nil {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "authorization predicate already exists", Cause: err}
		}
		return classify(err, "authorization predicate", in.Name)
	}
	return t.getAuthorizationPredicate(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) getAuthorizationPredicate(ctx context.Context, input, output any) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.authorizationPredicateValue(ctx, in.Name)
	if err != nil {
		return err
	}
	return assign(output, value)
}

func (t *Transport) listAuthorizationPredicates(ctx context.Context, input, output any) error {
	var in sdk.ListAuthorizationPredicatesInput
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
	rows, err := t.readAuthorizationPredicateRows(ctx, "", false, in.Query, in.Status, limit, in.Offset)
	if err != nil {
		return internal(err)
	}
	page := &sdk.AuthorizationPredicatePage{Limit: limit, Offset: in.Offset}
	for _, row := range rows {
		value, conversionErr := t.authorizationPredicateFromRow(row)
		if conversionErr != nil {
			return internal(conversionErr)
		}
		page.Items = append(page.Items, value)
	}
	return assign(output, page)
}

func (t *Transport) updateAuthorizationPredicate(ctx context.Context, input, output any) error {
	var in struct {
		Name  string                                `json:"name"`
		Input sdk.UpdateAuthorizationPredicateInput `json:"input"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if in.Input.ETag <= 0 {
		return invalid(errors.New("etag is required"))
	}
	if in.Input.ETag > math.MaxInt {
		return invalid(errors.New("etag is out of range"))
	}
	current, err := t.authorizationPredicateValue(ctx, in.Name)
	if err != nil {
		return err
	}
	if current.ETag != in.Input.ETag {
		return predicateETagConflict()
	}
	if in.Input.Title != nil {
		current.Title = strings.TrimSpace(*in.Input.Title)
	}
	if in.Input.Description != nil {
		current.Description = strings.TrimSpace(*in.Input.Description)
	}
	if in.Input.PackagePath != nil {
		current.PackagePath = strings.TrimSpace(*in.Input.PackagePath)
	}
	if in.Input.TypeName != nil {
		current.TypeName = strings.TrimSpace(*in.Input.TypeName)
	}
	if in.Input.Status != nil {
		current.Status = strings.TrimSpace(*in.Input.Status)
	}
	if in.Input.Alias != nil {
		current.Alias = strings.TrimSpace(*in.Input.Alias)
	}
	if in.Input.Columns != nil {
		current.Columns = *in.Input.Columns
	}
	if current.Title == "" || (current.Status != "active" && current.Status != "disabled") || current.Status == "active" && !t.Predicates.Contains(current.PackagePath, current.TypeName) {
		return invalid(errors.New("active authorization predicate requires a title and configured package/type link"))
	}
	scopeJSON, err := authorizationPredicateScopeJSON(current.Alias, current.Columns)
	if err != nil {
		return invalid(err)
	}
	etag, now := int(in.Input.ETag), t.now()
	row := &storedwriter.StoredAuthorizationPredicate{Name: current.Name, Title: current.Title,
		Description: predicateOptionalString(current.Description), PackagePath: current.PackagePath,
		TypeName: current.TypeName, SqlScopeJson: predicateOptionalString(scopeJSON), Status: current.Status,
		Etag: &etag, UpdatedAt: &now,
		Has: &storedwriter.StoredAuthorizationPredicateHas{Name: true, Title: true, Description: true,
			PackagePath: true, TypeName: true, SqlScopeJson: true, Status: true, Etag: true, UpdatedAt: true}}
	if err := t.writeAuthorizationPredicate(ctx, "put", row); err != nil {
		return t.classifyPredicateUpdateError(ctx, in.Name, in.Input.ETag, err)
	}
	return t.getAuthorizationPredicate(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) deleteAuthorizationPredicate(ctx context.Context, input any) error {
	var in struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if in.ETag > math.MaxInt {
		return predicateETagConflict()
	}
	current, err := t.authorizationPredicateValue(ctx, in.Name)
	if err != nil {
		var sdkErr *sdk.Error
		if errors.As(err, &sdkErr) && sdkErr.Code == sdk.ErrorNotFound {
			return predicateETagConflict()
		}
		return err
	}
	if current.ETag != in.ETag {
		return predicateETagConflict()
	}
	etag, now := int(in.ETag), t.now()
	row := &storedwriter.StoredAuthorizationPredicate{Name: in.Name, Status: "disabled", DeletedAt: &now, UpdatedAt: &now, Etag: &etag,
		Has: &storedwriter.StoredAuthorizationPredicateHas{Name: true, Status: true, DeletedAt: true, UpdatedAt: true, Etag: true}}
	if err := t.writeAuthorizationPredicate(ctx, "put", row); err != nil {
		return t.classifyPredicateUpdateError(ctx, in.Name, in.ETag, err)
	}
	return nil
}

func predicateOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func predicateETagConflict() error {
	return &sdk.Error{Code: sdk.ErrorConflict, Message: "authorization predicate etag does not match"}
}

func (t *Transport) classifyPredicateUpdateError(ctx context.Context, name string, expected int64, writeErr error) error {
	current, readErr := t.authorizationPredicateValue(ctx, name)
	if readErr == nil && current.ETag != expected {
		return predicateETagConflict()
	}
	var sdkErr *sdk.Error
	if errors.As(readErr, &sdkErr) && sdkErr.Code == sdk.ErrorNotFound {
		return predicateETagConflict()
	}
	return classify(writeErr, "authorization predicate", name)
}

func (t *Transport) authorizationPredicateValue(ctx context.Context, name string) (*sdk.AuthorizationPredicate, error) {
	rows, err := t.readAuthorizationPredicateRows(ctx, name, true, "", "", 1, 0)
	if err != nil {
		return nil, mapReadError(err, "authorization predicate", name)
	}
	if len(rows) == 0 {
		return nil, mapReadError(sql.ErrNoRows, "authorization predicate", name)
	}
	if len(rows) != 1 || rows[0] == nil || rows[0].Name != name {
		return nil, internal(errors.New("authorization predicate reader returned an ambiguous identity"))
	}
	value, err := t.authorizationPredicateFromRow(rows[0])
	if err != nil {
		return nil, internal(err)
	}
	return value, nil
}

func authorizationPredicateScopeJSON(alias string, columns []string) (string, error) {
	alias = strings.TrimSpace(alias)
	clean := make([]string, 0, len(columns))
	for _, column := range columns {
		if column = strings.TrimSpace(column); column != "" {
			clean = append(clean, column)
		}
	}
	if alias == "" && len(clean) == 0 {
		return "", nil
	}
	payload, err := json.Marshal(sdk.AuthorizationPredicateSQLMetadata{Alias: alias, Columns: clean})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func validPredicateName(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for index, char := range part {
			if index == 0 && (char < 'a' || char > 'z') {
				return false
			}
			if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' {
				continue
			}
			return false
		}
	}
	return true
}

func (t *Transport) authorizationPredicateTypes(output any) error {
	page := &sdk.AuthorizationPredicateTypePage{}
	for _, descriptor := range t.Predicates.Descriptors() {
		page.Items = append(page.Items, &sdk.AuthorizationPredicateType{Alias: descriptor.Alias, PackagePath: descriptor.Package, TypeName: descriptor.TypeName})
	}
	return assign(output, page)
}
