package sdk

import (
	"context"
	"time"
)

const (
	OperationNamespaceCreate = "namespaces.create"
	OperationNamespaceGet    = "namespaces.get"
	OperationNamespaceList   = "namespaces.list"
	OperationNamespaceUpdate = "namespaces.update"
	OperationNamespaceDelete = "namespaces.delete"
)

type Namespace struct {
	OwnerID     string    `json:"ownerId"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	ETag        int64     `json:"etag"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateNamespaceInput struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"ownerId,omitempty"`
}

type UpdateNamespaceInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	ETag        int64   `json:"etag"`
}

type ListNamespacesInput struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type NamespacePage struct {
	Items  []*Namespace `json:"items"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type NamespaceService interface {
	Create(context.Context, CreateNamespaceInput) (*Namespace, error)
	Get(context.Context, string) (*Namespace, error)
	List(context.Context, ListNamespacesInput) (*NamespacePage, error)
	Update(context.Context, string, UpdateNamespaceInput) (*Namespace, error)
	Delete(context.Context, string, int64) error
}

type namespaceClient struct{ transport Transport }

func (c namespaceClient) Create(ctx context.Context, input CreateNamespaceInput) (*Namespace, error) {
	return invoke[Namespace](ctx, c.transport, OperationNamespaceCreate, input)
}
func (c namespaceClient) Get(ctx context.Context, name string) (*Namespace, error) {
	return invoke[Namespace](ctx, c.transport, OperationNamespaceGet, struct {
		Name string `json:"name"`
	}{name})
}
func (c namespaceClient) List(ctx context.Context, input ListNamespacesInput) (*NamespacePage, error) {
	return invoke[NamespacePage](ctx, c.transport, OperationNamespaceList, input)
}
func (c namespaceClient) Update(ctx context.Context, name string, input UpdateNamespaceInput) (*Namespace, error) {
	return invoke[Namespace](ctx, c.transport, OperationNamespaceUpdate, struct {
		Name  string               `json:"name"`
		Input UpdateNamespaceInput `json:"input"`
	}{name, input})
}
func (c namespaceClient) Delete(ctx context.Context, name string, etag int64) error {
	return invokeEmpty(ctx, c.transport, OperationNamespaceDelete, struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}{name, etag})
}
