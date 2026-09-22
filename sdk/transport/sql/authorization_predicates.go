package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/viant/datly-studio/sdk"
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
	scopeJSON, err := authorizationPredicateScopeJSON(in.Alias, in.Columns)
	if err != nil {
		return invalid(err)
	}
	if _, err := t.DB.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,description,package_path,type_name,sql_scope_json,owner_id,status,etag,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'active',1,?,?)`, in.Name, in.Title, nullable(in.Description), in.PackagePath, in.TypeName, nullable(scopeJSON), principal.Subject, now, now); err != nil {
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
	query := `SELECT name,title,description,package_path,type_name,sql_scope_json,owner_id,status,etag,created_at,updated_at FROM authorization_predicates WHERE deleted_at IS NULL`
	args := []any{}
	if value := strings.TrimSpace(in.Query); value != "" {
		like := "%" + strings.ToLower(value) + "%"
		query += ` AND (LOWER(name) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(package_path) LIKE ? OR LOWER(type_name) LIKE ?)`
		args = append(args, like, like, like, like, like)
	}
	if in.Status != "" {
		query += ` AND status=?`
		args = append(args, in.Status)
	}
	query += ` ORDER BY updated_at DESC,name ASC LIMIT ? OFFSET ?`
	args = append(args, limit, in.Offset)
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.AuthorizationPredicatePage{Limit: limit, Offset: in.Offset}
	for rows.Next() {
		value, scanErr := t.scanAuthorizationPredicate(rows)
		if scanErr != nil {
			return internal(scanErr)
		}
		page.Items = append(page.Items, value)
	}
	if err = rows.Err(); err != nil {
		return internal(err)
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
	current, err := t.authorizationPredicateValue(ctx, in.Name)
	if err != nil {
		return err
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
	result, err := t.DB.ExecContext(ctx, `UPDATE authorization_predicates SET title=?,description=?,package_path=?,type_name=?,sql_scope_json=?,status=?,etag=etag+1,updated_at=? WHERE name=? AND etag=? AND deleted_at IS NULL`, current.Title, nullable(current.Description), current.PackagePath, current.TypeName, nullable(scopeJSON), current.Status, t.now(), current.Name, in.Input.ETag)
	if err != nil {
		return classify(err, "authorization predicate", in.Name)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "authorization predicate etag does not match"}
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
	result, err := t.DB.ExecContext(ctx, `UPDATE authorization_predicates SET status='disabled',deleted_at=?,updated_at=?,etag=etag+1 WHERE name=? AND etag=? AND deleted_at IS NULL`, t.now(), t.now(), in.Name, in.ETag)
	if err != nil {
		return internal(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "authorization predicate etag does not match"}
	}
	return nil
}

func (t *Transport) authorizationPredicateValue(ctx context.Context, name string) (*sdk.AuthorizationPredicate, error) {
	value, err := t.scanAuthorizationPredicate(t.DB.QueryRowContext(ctx, `SELECT name,title,description,package_path,type_name,sql_scope_json,owner_id,status,etag,created_at,updated_at FROM authorization_predicates WHERE name=? AND deleted_at IS NULL`, name))
	if err != nil {
		return nil, mapReadError(err, "authorization predicate", name)
	}
	return value, nil
}
func (t *Transport) scanAuthorizationPredicate(scanner interface{ Scan(...any) error }) (*sdk.AuthorizationPredicate, error) {
	var value sdk.AuthorizationPredicate
	var description, scopeJSON sql.NullString
	if err := scanner.Scan(&value.Name, &value.Title, &description, &value.PackagePath, &value.TypeName, &scopeJSON, &value.OwnerID, &value.Status, &value.ETag, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, err
	}
	value.Description = description.String
	if scopeJSON.Valid && strings.TrimSpace(scopeJSON.String) != "" {
		var metadata sdk.AuthorizationPredicateSQLMetadata
		if err := json.Unmarshal([]byte(scopeJSON.String), &metadata); err != nil {
			return nil, err
		}
		value.Alias, value.Columns = metadata.Alias, metadata.Columns
	}
	value.Linked = t.Predicates.Contains(value.PackagePath, value.TypeName)
	return &value, nil
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
