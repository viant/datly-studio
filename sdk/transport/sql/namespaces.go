package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

func (t *Transport) createNamespace(ctx context.Context, input, output any) error {
	var in sdk.CreateNamespaceInput
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if !validBusinessNamespace(in.Name) || strings.TrimSpace(in.Title) == "" {
		return invalid(errors.New("namespace name and title are required; name must use lowercase dot-separated segments"))
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if strings.TrimSpace(in.OwnerID) == "" {
		if !ok {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
		}
		in.OwnerID = principal.Subject
	}
	if ok && in.OwnerID != principal.Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "cannot create another principal's namespace"}
	}
	now := t.now()
	_, err := t.DB.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,description,status,etag,created_at,updated_at) VALUES(?,?,?,?,'active',1,?,?)`, in.OwnerID, in.Name, in.Title, nullable(in.Description), now, now)
	if err != nil {
		return classify(err, "namespace", in.Name)
	}
	return t.getNamespace(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) getNamespace(ctx context.Context, input, output any) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	query := `SELECT owner_id,name,title,description,status,etag,created_at,updated_at FROM namespaces WHERE name=? AND deleted_at IS NULL`
	query, args := namespaceReadScope(ctx, query, []any{in.Name})
	value, err := scanNamespace(t.DB.QueryRowContext(ctx, query, args...))
	if err != nil {
		return mapReadError(err, "namespace", in.Name)
	}
	return assign(output, value)
}

func (t *Transport) listNamespaces(ctx context.Context, input, output any) error {
	var in sdk.ListNamespacesInput
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
	query := `SELECT owner_id,name,title,description,status,etag,created_at,updated_at FROM namespaces WHERE deleted_at IS NULL`
	args := []any{}
	if value := strings.TrimSpace(in.Query); value != "" {
		like := "%" + strings.ToLower(value) + "%"
		query += ` AND (LOWER(name) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?)`
		args = append(args, like, like, like)
	}
	if in.Status != "" {
		query += ` AND status=?`
		args = append(args, in.Status)
	}
	query, args = namespaceReadScope(ctx, query, args)
	query += ` ORDER BY updated_at DESC,name ASC LIMIT ? OFFSET ?`
	args = append(args, limit, in.Offset)
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.NamespacePage{Limit: limit, Offset: in.Offset}
	for rows.Next() {
		value, scanErr := scanNamespace(rows)
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

func (t *Transport) updateNamespace(ctx context.Context, input, output any) error {
	var in struct {
		Name  string                   `json:"name"`
		Input sdk.UpdateNamespaceInput `json:"input"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if in.Input.ETag <= 0 {
		return invalid(errors.New("etag is required"))
	}
	current, err := t.namespaceValue(ctx, in.Name)
	if err != nil {
		return err
	}
	if in.Input.Title != nil {
		current.Title = strings.TrimSpace(*in.Input.Title)
	}
	if in.Input.Description != nil {
		current.Description = strings.TrimSpace(*in.Input.Description)
	}
	if in.Input.Status != nil {
		status := strings.TrimSpace(*in.Input.Status)
		if status != "active" && status != "archived" {
			return invalid(errors.New("namespace status must be active or archived"))
		}
		current.Status = status
	}
	if current.Title == "" {
		return invalid(errors.New("namespace title is required"))
	}
	result, err := t.DB.ExecContext(ctx, `UPDATE namespaces SET title=?,description=?,status=?,etag=etag+1,updated_at=? WHERE owner_id=? AND name=? AND etag=? AND deleted_at IS NULL`, current.Title, nullable(current.Description), current.Status, t.now(), current.OwnerID, current.Name, in.Input.ETag)
	if err != nil {
		return internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match"}
	}
	return t.getNamespace(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) deleteNamespace(ctx context.Context, input any) error {
	var in struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	current, err := t.namespaceValue(ctx, in.Name)
	if err != nil {
		return err
	}
	var used int
	if err = t.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM reports WHERE owner_id=? AND namespace=? AND deleted_at IS NULL`, current.OwnerID, current.Name).Scan(&used); err != nil {
		return internal(err)
	}
	if used > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace is referenced by a component"}
	}
	result, err := t.DB.ExecContext(ctx, `UPDATE namespaces SET status='archived',deleted_at=?,updated_at=?,etag=etag+1 WHERE owner_id=? AND name=? AND etag=? AND deleted_at IS NULL`, t.now(), t.now(), current.OwnerID, current.Name, in.ETag)
	if err != nil {
		return internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match"}
	}
	return nil
}

func (t *Transport) namespaceValue(ctx context.Context, name string) (*sdk.Namespace, error) {
	query := `SELECT owner_id,name,title,description,status,etag,created_at,updated_at FROM namespaces WHERE name=? AND deleted_at IS NULL`
	query, args := namespaceReadScope(ctx, query, []any{name})
	value, err := scanNamespace(t.DB.QueryRowContext(ctx, query, args...))
	if err != nil {
		return nil, mapReadError(err, "namespace", name)
	}
	return value, nil
}

func namespaceReadScope(ctx context.Context, query string, args []any) (string, []any) {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return query, args
	}
	query += ` AND (owner_id=? OR EXISTS (
SELECT 1 FROM reports namespace_report JOIN report_acl namespace_acl ON namespace_acl.report_id=namespace_report.id
WHERE namespace_report.owner_id=namespaces.owner_id AND namespace_report.namespace=namespaces.name
AND namespace_report.deleted_at IS NULL AND namespace_acl.subject_type='user'
AND namespace_acl.subject_id=? AND namespace_acl.can_view=TRUE))`
	return query, append(args, principal.Subject, principal.Subject)
}

func scanNamespace(scanner interface{ Scan(...any) error }) (*sdk.Namespace, error) {
	var value sdk.Namespace
	var description sql.NullString
	if err := scanner.Scan(&value.OwnerID, &value.Name, &value.Title, &description, &value.Status, &value.ETag, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, err
	}
	value.Description = description.String
	return &value, nil
}
