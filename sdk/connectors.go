package sdk

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OperationConnectorCreate   = "connectors.create"
	OperationConnectorGet      = "connectors.get"
	OperationConnectorList     = "connectors.list"
	OperationConnectorUpdate   = "connectors.update"
	OperationConnectorTest     = "connectors.test"
	OperationConnectorActivate = "connectors.activate"
	OperationConnectorDisable  = "connectors.disable"
	OperationConnectorDelete   = "connectors.delete"
)

type Connector struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	// DSNTemplate is server-only connection material. It is intentionally
	// omitted from JSON DTOs returned to Forge or other SDK HTTP consumers.
	DSNTemplate       string          `json:"-"`
	DSNConfigured     bool            `json:"dsnConfigured"`
	SecretRef         string          `json:"-"`
	SecretConfigured  bool            `json:"secretConfigured"`
	Description       string          `json:"description,omitempty"`
	OwnerID           string          `json:"ownerId"`
	Status            string          `json:"status"`
	Options           json.RawMessage `json:"options,omitempty"`
	LastTestStatus    string          `json:"lastTestStatus,omitempty"`
	LastTestErrorCode string          `json:"lastTestErrorCode,omitempty"`
	LastTestedAt      *time.Time      `json:"lastTestedAt,omitempty"`
	ETag              int64           `json:"etag"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type CreateConnectorInput struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	DSNTemplate string `json:"dsnTemplate,omitempty"`
	SecretRef   string `json:"secretRef,omitempty"`
	Description string `json:"description,omitempty"`
	// OwnerID is accepted only for trusted in-process callers. The SDK HTTP
	// host derives ownership from the verified principal when it is omitted.
	OwnerID string          `json:"ownerId,omitempty"`
	Options json.RawMessage `json:"options,omitempty"`
}

type UpdateConnectorInput struct {
	Driver      *string          `json:"driver,omitempty"`
	DSNTemplate *string          `json:"dsnTemplate,omitempty"`
	SecretRef   *string          `json:"secretRef,omitempty"`
	Description *string          `json:"description,omitempty"`
	Options     *json.RawMessage `json:"options,omitempty"`
	ETag        int64            `json:"etag"`
}

type ListConnectorsInput struct {
	Query   string   `json:"query,omitempty"`
	Status  string   `json:"status,omitempty"`
	OwnerID string   `json:"ownerId,omitempty"`
	Driver  string   `json:"driver,omitempty"`
	Fields  []string `json:"fields,omitempty"`
	OrderBy string   `json:"orderBy,omitempty"`
	Limit   int      `json:"limit,omitempty"`
	Offset  int      `json:"offset,omitempty"`
}

type ConnectorPage struct {
	Items  []*Connector `json:"items"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type ConnectorTestResult struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	ErrorCode string    `json:"errorCode,omitempty"`
	Message   string    `json:"message,omitempty"`
	TestedAt  time.Time `json:"testedAt"`
}

// ConnectorProbe performs a real, host-owned connectivity check. It receives
// a Studio DTO only; secret resolution and driver linkage stay in the host.
type ConnectorProbe interface {
	Probe(context.Context, *Connector) (*ConnectorTestResult, error)
}

type ConnectorProbeFunc func(context.Context, *Connector) (*ConnectorTestResult, error)

func (f ConnectorProbeFunc) Probe(ctx context.Context, connector *Connector) (*ConnectorTestResult, error) {
	return f(ctx, connector)
}

type ConnectorService interface {
	Create(context.Context, CreateConnectorInput) (*Connector, error)
	Get(context.Context, string) (*Connector, error)
	List(context.Context, ListConnectorsInput) (*ConnectorPage, error)
	Update(context.Context, string, UpdateConnectorInput) (*Connector, error)
	Test(context.Context, string) (*ConnectorTestResult, error)
	Activate(context.Context, string, int64) (*Connector, error)
	Disable(context.Context, string, int64) (*Connector, error)
	Delete(context.Context, string, int64) error
}

type connectorClient struct{ transport Transport }

func (c connectorClient) Create(ctx context.Context, input CreateConnectorInput) (*Connector, error) {
	return invoke[Connector](ctx, c.transport, OperationConnectorCreate, input)
}
func (c connectorClient) Get(ctx context.Context, name string) (*Connector, error) {
	return invoke[Connector](ctx, c.transport, OperationConnectorGet, struct {
		Name string `json:"name"`
	}{name})
}
func (c connectorClient) List(ctx context.Context, input ListConnectorsInput) (*ConnectorPage, error) {
	return invoke[ConnectorPage](ctx, c.transport, OperationConnectorList, input)
}
func (c connectorClient) Update(ctx context.Context, name string, input UpdateConnectorInput) (*Connector, error) {
	return invoke[Connector](ctx, c.transport, OperationConnectorUpdate, struct {
		Name  string               `json:"name"`
		Input UpdateConnectorInput `json:"input"`
	}{name, input})
}
func (c connectorClient) Test(ctx context.Context, name string) (*ConnectorTestResult, error) {
	return invoke[ConnectorTestResult](ctx, c.transport, OperationConnectorTest, struct {
		Name string `json:"name"`
	}{name})
}
func (c connectorClient) Activate(ctx context.Context, name string, etag int64) (*Connector, error) {
	return invoke[Connector](ctx, c.transport, OperationConnectorActivate, connectorIdentity{Name: name, ETag: etag})
}
func (c connectorClient) Disable(ctx context.Context, name string, etag int64) (*Connector, error) {
	return invoke[Connector](ctx, c.transport, OperationConnectorDisable, connectorIdentity{Name: name, ETag: etag})
}
func (c connectorClient) Delete(ctx context.Context, name string, etag int64) error {
	return invokeEmpty(ctx, c.transport, OperationConnectorDelete, connectorIdentity{Name: name, ETag: etag})
}

type connectorIdentity struct {
	Name string `json:"name"`
	ETag int64  `json:"etag"`
}
