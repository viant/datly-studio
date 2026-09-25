package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	insert "github.com/viant/datly-studio/studio/namespaces/store_insert"
	stored "github.com/viant/datly-studio/studio/namespaces/store_write"
	xhandler "github.com/viant/xdatly/handler"
)

func namespaceOptionalDescription(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

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
	etag := int64(1)
	err := t.insertNamespaceRow(ctx, &insert.StoredNamespace{OwnerId: in.OwnerID, Name: in.Name,
		Title: in.Title, Description: namespaceOptionalDescription(in.Description), Status: "active",
		Etag: &etag, CreatedAt: &now, UpdatedAt: &now,
		Has: &insert.StoredNamespaceHas{OwnerId: true, Name: true, Title: true, Description: true,
			Status: true, Etag: true, CreatedAt: true, UpdatedAt: true}})
	if err != nil {
		return classify(err, "namespace", in.Name)
	}
	return t.getNamespace(ctx, struct {
		Name string `json:"name"`
	}{in.Name}, output)
}

func (t *Transport) insertNamespaceRow(ctx context.Context, row *insert.StoredNamespace) error {
	return t.writeNamespaceInsertRow(ctx, row)
}

func (t *Transport) insertDefaultNamespace(ctx context.Context, ownerID string, now time.Time) error {
	etag, description := int64(1), "Default namespace"
	return t.insertNamespaceRow(ctx, &insert.StoredNamespace{OwnerId: ownerID, Name: "general",
		Title: "General", Description: &description, Status: "active",
		Etag: &etag, CreatedAt: &now, UpdatedAt: &now,
		Has: &insert.StoredNamespaceHas{OwnerId: true, Name: true, Title: true, Description: true,
			Status: true, Etag: true, CreatedAt: true, UpdatedAt: true}})
}

func (t *Transport) getNamespace(ctx context.Context, input, output any) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	value, err := t.namespaceValue(ctx, in.Name)
	if err != nil {
		return err
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
	items, err := t.readNamespaces(ctx, "", in.Query, in.Status, limit, in.Offset)
	if err != nil {
		return internal(err)
	}
	page := &sdk.NamespacePage{Items: items, Limit: limit, Offset: in.Offset}
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
	etag, now := in.Input.ETag, t.now()
	err = t.writeNamespaceRow(ctx, &stored.StoredNamespace{OwnerId: current.OwnerID, Name: current.Name,
		Title: current.Title, Description: namespaceOptionalDescription(current.Description), Status: current.Status,
		Etag: &etag, UpdatedAt: &now,
		Has: &stored.StoredNamespaceHas{OwnerId: true, Name: true, Title: true,
			Description: true, Status: true, Etag: true, UpdatedAt: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match"}
		}
		return internal(err)
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
	used, err := t.namespaceUsage(ctx, current.OwnerID, current.Name)
	if err != nil {
		return internal(err)
	}
	if used > 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace is referenced by a component"}
	}
	now := t.now()
	etag := in.ETag
	err = t.writeNamespaceRow(ctx, &stored.StoredNamespace{OwnerId: current.OwnerID, Name: current.Name,
		Status: "archived", DeletedAt: &now, UpdatedAt: &now, Etag: &etag,
		Has: &stored.StoredNamespaceHas{OwnerId: true, Name: true, Status: true,
			DeletedAt: true, UpdatedAt: true, Etag: true}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "namespace etag does not match"}
		}
		return internal(err)
	}
	return nil
}

func (t *Transport) namespaceValue(ctx context.Context, name string) (*sdk.Namespace, error) {
	if name == "" {
		return nil, mapReadError(sql.ErrNoRows, "namespace", name)
	}
	items, err := t.readNamespaces(ctx, name, "", "", 1, 0)
	if err != nil {
		return nil, internal(err)
	}
	if len(items) == 0 {
		return nil, mapReadError(sql.ErrNoRows, "namespace", name)
	}
	if items[0].Name != name {
		return nil, internal(errors.New("namespace reader returned an ambiguous or mismatched row"))
	}
	return items[0], nil
}
