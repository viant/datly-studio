package sdk

import (
	"context"
	"strings"
	"time"
)

const (
	OperationReportCreate = "reports.create"
	OperationReportGet    = "reports.get"
	OperationReportList   = "reports.list"
	OperationReportUpdate = "reports.update"
)

type Report struct {
	ID                   string    `json:"id"`
	Namespace            string    `json:"namespace"`
	Slug                 string    `json:"slug"`
	Title                string    `json:"title"`
	Description          string    `json:"description,omitempty"`
	OwnerID              string    `json:"ownerId"`
	OwnerPackage         string    `json:"ownerPackage"`
	Status               string    `json:"status"`
	DefaultConnectorName string    `json:"defaultConnectorName"`
	ComponentScope       string    `json:"componentScope"`
	ComponentName        string    `json:"componentName"`
	CurrentDraftVersion  *int      `json:"currentDraftVersion,omitempty"`
	ETag                 int64     `json:"etag"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type CreateReportInput struct {
	// ID, OwnerID, ComponentScope and ComponentName are optional for the public
	// SDK. The server derives stable values from the verified principal and a
	// generated report identity when they are not supplied by a trusted caller.
	ID                   string `json:"id,omitempty"`
	Slug                 string `json:"slug"`
	Namespace            string `json:"namespace"`
	Title                string `json:"title"`
	Description          string `json:"description,omitempty"`
	OwnerID              string `json:"ownerId,omitempty"`
	DefaultConnectorName string `json:"defaultConnectorName"`
	ComponentScope       string `json:"componentScope,omitempty"`
	ComponentName        string `json:"componentName,omitempty"`
}
type UpdateReportInput struct {
	Namespace            *string `json:"namespace,omitempty"`
	Slug                 *string `json:"slug,omitempty"`
	Title                *string `json:"title,omitempty"`
	Description          *string `json:"description,omitempty"`
	Status               *string `json:"status,omitempty"`
	DefaultConnectorName *string `json:"defaultConnectorName,omitempty"`
	ComponentScope       *string `json:"componentScope,omitempty"`
	ComponentName        *string `json:"componentName,omitempty"`
	CurrentDraftVersion  *int    `json:"currentDraftVersion,omitempty"`
	ETag                 int64   `json:"etag"`
}
type ListReportsInput struct {
	Query         string   `json:"query,omitempty"`
	Namespace     string   `json:"namespace,omitempty"`
	Status        string   `json:"status,omitempty"`
	OwnerID       string   `json:"ownerId,omitempty"`
	ConnectorName string   `json:"connectorName,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	OrderBy       string   `json:"orderBy,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	Offset        int      `json:"offset,omitempty"`
}
type ReportPage struct {
	Items  []*Report `json:"items"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// OwnerPackageSegment returns a deterministic Go-package-safe form of the
// verified JWT sub. It is used by dynamic component packages and globally
// scoped MCP tool names; callers must never substitute display-name claims.
func OwnerPackageSegment(subject string) string {
	base := strings.ToLower(strings.TrimSpace(subject))
	if at := strings.IndexByte(base, '@'); at >= 0 {
		base = base[:at]
	}
	var result strings.Builder
	for _, character := range base {
		valid := character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
		if valid {
			result.WriteRune(character)
		}
	}
	value := result.String()
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		value = "user" + value
	}
	if len(value) > 56 {
		value = value[:56]
	}
	return value
}

type ReportService interface {
	Create(context.Context, CreateReportInput) (*Report, error)
	Get(context.Context, string) (*Report, error)
	List(context.Context, ListReportsInput) (*ReportPage, error)
	Update(context.Context, string, UpdateReportInput) (*Report, error)
}

type reportClient struct{ transport Transport }

func (c reportClient) Create(ctx context.Context, input CreateReportInput) (*Report, error) {
	return invoke[Report](ctx, c.transport, OperationReportCreate, input)
}
func (c reportClient) Get(ctx context.Context, id string) (*Report, error) {
	return invoke[Report](ctx, c.transport, OperationReportGet, struct {
		ID string `json:"id"`
	}{id})
}
func (c reportClient) List(ctx context.Context, input ListReportsInput) (*ReportPage, error) {
	return invoke[ReportPage](ctx, c.transport, OperationReportList, input)
}
func (c reportClient) Update(ctx context.Context, id string, input UpdateReportInput) (*Report, error) {
	return invoke[Report](ctx, c.transport, OperationReportUpdate, struct {
		ID    string            `json:"id"`
		Input UpdateReportInput `json:"input"`
	}{id, input})
}
