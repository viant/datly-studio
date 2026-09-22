package sdk

import (
	"context"
	"time"
)

const (
	OperationAuthorizationPredicateCreate = "authorization_predicates.create"
	OperationAuthorizationPredicateGet    = "authorization_predicates.get"
	OperationAuthorizationPredicateList   = "authorization_predicates.list"
	OperationAuthorizationPredicateUpdate = "authorization_predicates.update"
	OperationAuthorizationPredicateDelete = "authorization_predicates.delete"
	OperationAuthorizationPredicateTypes  = "authorization_predicates.types"
)

type AuthorizationPredicate struct {
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	PackagePath string    `json:"packagePath"`
	TypeName    string    `json:"typeName"`
	Alias       string    `json:"alias,omitempty"`
	Columns     []string  `json:"columns,omitempty"`
	OwnerID     string    `json:"ownerId"`
	Status      string    `json:"status"`
	Linked      bool      `json:"linked"`
	ETag        int64     `json:"etag"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AuthorizationPredicateSQLMetadata struct {
	Alias   string   `json:"alias"`
	Columns []string `json:"columns"`
}

type CreateAuthorizationPredicateInput struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	PackagePath string   `json:"packagePath"`
	TypeName    string   `json:"typeName"`
	Alias       string   `json:"alias,omitempty"`
	Columns     []string `json:"columns,omitempty"`
}
type UpdateAuthorizationPredicateInput struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	PackagePath *string   `json:"packagePath,omitempty"`
	TypeName    *string   `json:"typeName,omitempty"`
	Alias       *string   `json:"alias,omitempty"`
	Columns     *[]string `json:"columns,omitempty"`
	Status      *string   `json:"status,omitempty"`
	ETag        int64     `json:"etag"`
}
type ListAuthorizationPredicatesInput struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
type AuthorizationPredicatePage struct {
	Items  []*AuthorizationPredicate `json:"items"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

type AuthorizationPredicateType struct {
	Alias       string `json:"alias,omitempty"`
	PackagePath string `json:"packagePath"`
	TypeName    string `json:"typeName"`
}

type AuthorizationPredicateTypePage struct {
	Items []*AuthorizationPredicateType `json:"items"`
}

type AuthorizationPredicateService interface {
	Create(context.Context, CreateAuthorizationPredicateInput) (*AuthorizationPredicate, error)
	Get(context.Context, string) (*AuthorizationPredicate, error)
	List(context.Context, ListAuthorizationPredicatesInput) (*AuthorizationPredicatePage, error)
	Update(context.Context, string, UpdateAuthorizationPredicateInput) (*AuthorizationPredicate, error)
	Delete(context.Context, string, int64) error
	Types(context.Context) (*AuthorizationPredicateTypePage, error)
}

type authorizationPredicateClient struct{ transport Transport }

func (c authorizationPredicateClient) Create(ctx context.Context, input CreateAuthorizationPredicateInput) (*AuthorizationPredicate, error) {
	return invoke[AuthorizationPredicate](ctx, c.transport, OperationAuthorizationPredicateCreate, input)
}
func (c authorizationPredicateClient) Get(ctx context.Context, name string) (*AuthorizationPredicate, error) {
	return invoke[AuthorizationPredicate](ctx, c.transport, OperationAuthorizationPredicateGet, struct {
		Name string `json:"name"`
	}{name})
}
func (c authorizationPredicateClient) List(ctx context.Context, input ListAuthorizationPredicatesInput) (*AuthorizationPredicatePage, error) {
	return invoke[AuthorizationPredicatePage](ctx, c.transport, OperationAuthorizationPredicateList, input)
}
func (c authorizationPredicateClient) Update(ctx context.Context, name string, input UpdateAuthorizationPredicateInput) (*AuthorizationPredicate, error) {
	return invoke[AuthorizationPredicate](ctx, c.transport, OperationAuthorizationPredicateUpdate, struct {
		Name  string                            `json:"name"`
		Input UpdateAuthorizationPredicateInput `json:"input"`
	}{name, input})
}
func (c authorizationPredicateClient) Delete(ctx context.Context, name string, etag int64) error {
	return invokeEmpty(ctx, c.transport, OperationAuthorizationPredicateDelete, struct {
		Name string `json:"name"`
		ETag int64  `json:"etag"`
	}{name, etag})
}
func (c authorizationPredicateClient) Types(ctx context.Context) (*AuthorizationPredicateTypePage, error) {
	return invoke[AuthorizationPredicateTypePage](ctx, c.transport, OperationAuthorizationPredicateTypes, nil)
}
