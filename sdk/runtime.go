package sdk

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OperationPreviewExecute        = "preview.execute"
	OperationPublicationPublish    = "publications.publish"
	OperationPublicationRemove     = "publications.unpublish"
	OperationPublicationRollback   = "publications.rollback"
	OperationPublicationGet        = "publications.get"
	OperationPublicationEventsList = "publications.events.list"
	OperationRuntimeStatus         = "runtime.status"
)

type PreviewInput struct {
	Input json.RawMessage `json:"input,omitempty"`
	Limit int             `json:"limit,omitempty"`
}
type PreviewResult struct {
	Data        json.RawMessage   `json:"data,omitempty"`
	Diagnostics []Diagnostic      `json:"diagnostics,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Evidence    ExecutionEvidence `json:"evidence"`
}

type ExecutionEvidence struct {
	ReportID       string `json:"reportId,omitempty"`
	VersionNo      int    `json:"versionNo,omitempty"`
	SourceRevision int64  `json:"sourceRevision,omitempty"`
	Connector      string `json:"connector,omitempty"`
	Limit          int    `json:"limit"`
	ReturnedRows   int    `json:"returnedRows"`
	EncodedBytes   int    `json:"encodedBytes"`
	Truncated      bool   `json:"truncated"`
}
type PublishInput struct {
	RequestedBy            string `json:"requestedBy"`
	Reason                 string `json:"reason,omitempty"`
	ExpectedSourceRevision int64  `json:"expectedSourceRevision,omitempty"`
}
type UnpublishInput struct {
	RequestedBy string `json:"requestedBy"`
	Reason      string `json:"reason,omitempty"`
}
type Publication struct {
	ReportID          string     `json:"reportId"`
	ActiveVersionNo   int        `json:"activeVersionNo"`
	DesiredVersionNo  *int       `json:"desiredVersionNo,omitempty"`
	DesiredGeneration int64      `json:"desiredGeneration"`
	ActiveGeneration  *int64     `json:"activeGeneration,omitempty"`
	Status            string     `json:"status"`
	RuntimeRevision   string     `json:"runtimeRevision,omitempty"`
	SpecHash          string     `json:"specHash,omitempty"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
}

// PublicationEvent is append-only lifecycle evidence. Failure fields are
// deliberately bounded and secret-redacted before persistence.
type PublicationEvent struct {
	EventID        string    `json:"eventId"`
	ReportID       string    `json:"reportId"`
	OwnerID        string    `json:"ownerId"`
	Operation      string    `json:"operation"`
	VersionNo      *int      `json:"versionNo,omitempty"`
	GenerationNo   *int64    `json:"generationNo,omitempty"`
	Status         string    `json:"status"`
	RequestedBy    string    `json:"requestedBy"`
	Reason         string    `json:"reason,omitempty"`
	FailureCode    string    `json:"failureCode,omitempty"`
	FailureMessage string    `json:"failureMessage,omitempty"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type ListPublicationEventsInput struct {
	Operation string `json:"operation,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type PublicationEventPage struct {
	Items  []*PublicationEvent `json:"items"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}
type RuntimeStatus struct {
	ActiveGeneration int64           `json:"activeGeneration"`
	Status           string          `json:"status"`
	ReportCount      int             `json:"reportCount"`
	ActivatedAt      *time.Time      `json:"activatedAt,omitempty"`
	Diagnostics      []Diagnostic    `json:"diagnostics,omitempty"`
	Readers          []RuntimeReader `json:"readers,omitempty"`
	Host             *RuntimeHost    `json:"host,omitempty"`
}

type RuntimeHost struct {
	Status    string    `json:"status"`
	Revision  int64     `json:"revision,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

type RuntimeReader struct {
	ReportID        string               `json:"reportId"`
	Title           string               `json:"title"`
	Namespace       string               `json:"namespace"`
	OwnerPackage    string               `json:"ownerPackage"`
	ConnectorName   string               `json:"connectorName"`
	ComponentName   string               `json:"componentName,omitempty"`
	VersionNo       int                  `json:"versionNo"`
	Status          string               `json:"status"`
	RuntimeRevision string               `json:"runtimeRevision,omitempty"`
	ActivatedAt     *time.Time           `json:"activatedAt,omitempty"`
	MCPExposures    []RuntimeMCPExposure `json:"mcpExposures,omitempty"`
	MCPResources    []RuntimeMCPResource `json:"mcpResources,omitempty"`
	Skills          []RuntimeSkill       `json:"skills,omitempty"`
}

type RuntimeMCPExposure struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Component   string `json:"component,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type RuntimeMCPResource struct {
	Namespace string `json:"namespace"`
	RootPath  string `json:"rootPath"`
	URIPrefix string `json:"uriPrefix"`
}

type RuntimeSkill struct {
	SkillID   string `json:"skillId"`
	SkillRoot string `json:"skillRoot"`
	URIPrefix string `json:"uriPrefix"`
}

type PreviewService interface {
	Execute(context.Context, string, int, PreviewInput) (*PreviewResult, error)
}
type PublicationService interface {
	Get(context.Context, string) (*Publication, error)
	Publish(context.Context, string, int, PublishInput) (*Publication, error)
	Unpublish(context.Context, string, UnpublishInput) (*Publication, error)
	Rollback(context.Context, string, int, PublishInput) (*Publication, error)
	ListEvents(context.Context, string, ListPublicationEventsInput) (*PublicationEventPage, error)
}
type RuntimeService interface {
	Status(context.Context) (*RuntimeStatus, error)
}

type previewClient struct{ transport Transport }

func (c previewClient) Execute(ctx context.Context, id string, v int, input PreviewInput) (*PreviewResult, error) {
	return invoke[PreviewResult](ctx, c.transport, OperationPreviewExecute, struct {
		versionIdentity
		Input PreviewInput `json:"input"`
	}{versionIdentity{id, v}, input})
}

type publicationClient struct{ transport Transport }

func (c publicationClient) Get(ctx context.Context, id string) (*Publication, error) {
	return invoke[Publication](ctx, c.transport, OperationPublicationGet, struct {
		ReportID string `json:"reportId"`
	}{id})
}

func (c publicationClient) Publish(ctx context.Context, id string, v int, input PublishInput) (*Publication, error) {
	return invoke[Publication](ctx, c.transport, OperationPublicationPublish, struct {
		versionIdentity
		Input PublishInput `json:"input"`
	}{versionIdentity{id, v}, input})
}
func (c publicationClient) Unpublish(ctx context.Context, id string, input UnpublishInput) (*Publication, error) {
	return invoke[Publication](ctx, c.transport, OperationPublicationRemove, struct {
		ReportID string         `json:"reportId"`
		Input    UnpublishInput `json:"input"`
	}{id, input})
}
func (c publicationClient) Rollback(ctx context.Context, id string, v int, input PublishInput) (*Publication, error) {
	return invoke[Publication](ctx, c.transport, OperationPublicationRollback, struct {
		versionIdentity
		Input PublishInput `json:"input"`
	}{versionIdentity{id, v}, input})
}
func (c publicationClient) ListEvents(ctx context.Context, id string, input ListPublicationEventsInput) (*PublicationEventPage, error) {
	return invoke[PublicationEventPage](ctx, c.transport, OperationPublicationEventsList, struct {
		ReportID string                     `json:"reportId"`
		Input    ListPublicationEventsInput `json:"input"`
	}{id, input})
}

type runtimeClient struct{ transport Transport }

func (c runtimeClient) Status(ctx context.Context) (*RuntimeStatus, error) {
	return invoke[RuntimeStatus](ctx, c.transport, OperationRuntimeStatus, nil)
}
