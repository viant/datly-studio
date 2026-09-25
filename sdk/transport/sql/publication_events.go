package sqltransport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	publicationstore "github.com/viant/datly-studio/sdk/transport/sql/internal/publications"
	insert "github.com/viant/datly-studio/studio/report_publication_events/store_insert"
	list "github.com/viant/datly-studio/studio/report_publication_events/store_list"
	owner "github.com/viant/datly-studio/studio/report_publication_events/store_owner"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

const publicationEventTextLimit = 1000

var (
	publicationEventCredentialURL = regexp.MustCompile(`(?i)(://)[^/@\s]+@`)
	publicationEventSecretPair    = regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|authorization|api[_-]?key)\s*([=:])\s*[^\s,;]+`)
)

type publicationEventRecord struct {
	ReportID     string
	OwnerID      string
	Operation    string
	VersionNo    *int
	GenerationNo *int64
	Status       string
	RequestedBy  string
	Reason       string
	FailureCode  string
	Failure      string
	OccurredAt   time.Time
}

type publicationEventListRequest struct {
	ReportID string                         `json:"reportId"`
	Input    sdk.ListPublicationEventsInput `json:"input"`
}

// listPublicationEvents exposes only a report's immutable, non-secret audit
// trail. It also checks current ownership here because delegated publish
// permission must not grant access to the owner's event history.
func (t *Transport) listPublicationEvents(ctx context.Context, input, output any) error {
	var in publicationEventListRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.Input.Offset < 0 {
		return invalid(errors.New("reportId and non-negative offset are required"))
	}
	limit := in.Input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	operation := strings.TrimSpace(in.Input.Operation)
	if operation != "" {
		if !validPublicationOperation(operation) {
			return invalid(errors.New("operation must be publish, rollback, or unpublish"))
		}
	}
	status := strings.TrimSpace(in.Input.Status)
	if status != "" {
		if status != "succeeded" && status != "failed" {
			return invalid(errors.New("status must be succeeded or failed"))
		}
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || strings.TrimSpace(principal.Subject) == "" {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	ownerID, err := t.publicationEventOwner(ctx, in.ReportID)
	if err != nil {
		return err
	}
	if ownerID != principal.Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "publication event history is owner-only"}
	}
	reader, err := t.publicationEventReader()
	if err != nil {
		return internal(err)
	}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.list, Input: &list.Input{
		ReportId: in.ReportID, Operation: operation, Status: status, PageLimit: limit, PageOffset: in.Input.Offset,
		Has: &list.InputHas{ReportId: true, Operation: true, Status: true, PageLimit: true, PageOffset: true},
	}})
	if err != nil {
		return internal(err)
	}
	result, ok := value.(*list.Output)
	if !ok {
		return internal(fmt.Errorf("publication event reader returned %T", value))
	}
	page := &sdk.PublicationEventPage{Limit: limit, Offset: in.Input.Offset}
	for _, row := range result.Events {
		if row == nil || row.ReportId != in.ReportID {
			return internal(errors.New("publication event reader returned a mismatched report"))
		}
		item := &sdk.PublicationEvent{EventID: row.EventId, ReportID: row.ReportId, OwnerID: row.OwnerId,
			Operation: row.Operation, VersionNo: row.VersionNo, GenerationNo: row.GenerationNo,
			Status: row.Status, RequestedBy: row.RequestedBy, OccurredAt: row.OccurredAt}
		if row.Reason != nil {
			item.Reason = *row.Reason
		}
		if row.FailureCode != nil {
			item.FailureCode = *row.FailureCode
		}
		if row.FailureMessage != nil {
			item.FailureMessage = *row.FailureMessage
		}
		page.Items = append(page.Items, item)
	}
	return assign(output, page)
}

func (t *Transport) publicationEventOwner(ctx context.Context, reportID string) (string, error) {
	reader, err := t.publicationEventReader()
	if err != nil {
		return "", internal(err)
	}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.owner, Input: &owner.Input{
		ReportId: reportID, Has: &owner.InputHas{ReportId: true},
	}})
	if err != nil {
		return "", internal(err)
	}
	result, ok := value.(*owner.Output)
	if !ok {
		return "", internal(fmt.Errorf("publication owner reader returned %T", value))
	}
	if len(result.Reports) == 0 {
		return "", &sdk.Error{Code: sdk.ErrorNotFound, Message: "report not found"}
	}
	if len(result.Reports) != 1 || result.Reports[0] == nil || result.Reports[0].Id != reportID {
		return "", internal(errors.New("publication owner reader returned an ambiguous or mismatched report"))
	}
	return result.Reports[0].OwnerId, nil
}

type publicationEventStoreReader struct {
	runtime *druntime.Runtime
	list    dexec.ComponentTarget
	owner   dexec.ComponentTarget
}

func (t *Transport) publicationEventReader() (*publicationEventStoreReader, error) {
	t.publicationReaderMu.Lock()
	defer t.publicationReaderMu.Unlock()
	if t.publicationReader != nil {
		return t.publicationReader, nil
	}
	resources := resource.New()
	if err := resources.Register(list.EventDatlyResourceNamespace, list.EventDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(owner.ReportDatlyResourceNamespace, owner.ReportDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	listRegistration, listTarget, err := readercomponent.Compile(reflect.TypeOf(list.EventComponent{}), "store_list",
		reflect.TypeOf(list.Input{}), reflect.TypeOf(list.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	ownerRegistration, ownerTarget, err := readercomponent.Compile(reflect.TypeOf(owner.ReportComponent{}), "store_owner",
		reflect.TypeOf(owner.Input{}), reflect.TypeOf(owner.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{listRegistration, ownerRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	t.publicationReader = &publicationEventStoreReader{runtime: runtime, list: listTarget, owner: ownerTarget}
	return t.publicationReader, nil
}

func (t *Transport) appendPublicationEvent(ctx context.Context, event publicationEventRecord) error {
	return t.appendPublicationEventNative(ctx, nil, event)
}

func (t *Transport) appendPublicationEventTx(ctx context.Context, tx *sql.Tx, event publicationEventRecord) error {
	return t.appendPublicationEventNative(ctx, tx, event)
}

func (t *Transport) appendPublicationEventNative(ctx context.Context, tx *sql.Tx, event publicationEventRecord) error {
	if event.OwnerID == "" || event.ReportID == "" || !validPublicationOperation(event.Operation) || (event.Status != "succeeded" && event.Status != "failed") {
		return errors.New("invalid publication event")
	}
	eventID, err := generatedPublicationEventID()
	if err != nil {
		return err
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = t.now()
	}
	requestedBy := sdk.SystemPrincipal().Subject
	if principal, ok := sdk.PrincipalFromContext(ctx); ok {
		requestedBy = principal.Subject
	}
	row := &insert.StoredEvent{EventId: eventID, ReportId: event.ReportID, OwnerId: event.OwnerID,
		Operation: event.Operation, VersionNo: event.VersionNo, GenerationNo: event.GenerationNo,
		Status: event.Status, RequestedBy: requestedBy, Reason: optionalPublicationText(event.Reason),
		FailureCode: optionalPublicationText(event.FailureCode), FailureMessage: optionalPublicationText(event.Failure),
		OccurredAt: event.OccurredAt,
		Has: &insert.StoredEventHas{EventId: true, ReportId: true, OwnerId: true,
			Operation: true, VersionNo: true, GenerationNo: true, Status: true,
			RequestedBy: true, Reason: true, FailureCode: true, FailureMessage: true,
			OccurredAt: true}}
	return publicationstore.WriteEvent(ctx, t.DB, tx, row)
}

func optionalPublicationText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (t *Transport) recordPublicationFailure(ctx context.Context, event publicationEventRecord, cause error) {
	if event.OwnerID == "" {
		return
	}
	event.Status = "failed"
	event.OccurredAt = t.now()
	event.FailureCode, event.Failure = boundedPublicationFailure(cause)
	// The audit trail must never alter publication failure handling or active
	// runtime state. A database outage is already represented by the original
	// transition failure and should not be masked by audit persistence.
	_ = t.appendPublicationEvent(context.WithoutCancel(ctx), event)
}

func boundedPublicationFailure(cause error) (string, string) {
	code := "transition_failed"
	message := "publication transition failed"
	var sdkErr *sdk.Error
	if errors.As(cause, &sdkErr) {
		if sdkErr.Code != "" {
			code = string(sdkErr.Code)
		}
		if strings.TrimSpace(sdkErr.Message) != "" {
			message = sdkErr.Message
		}
	} else if errors.Is(cause, context.DeadlineExceeded) {
		code, message = "timeout", "publication transition exceeded its deadline"
	} else if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		message = cause.Error()
	}
	return boundedPublicationText(code), boundedPublicationText(message)
}

func boundedPublicationText(value string) string {
	value = strings.TrimSpace(value)
	value = publicationEventCredentialURL.ReplaceAllString(value, "${1}***@")
	value = publicationEventSecretPair.ReplaceAllString(value, "$1$2***")
	if utf8.RuneCountInString(value) <= publicationEventTextLimit {
		return value
	}
	runes := []rune(value)
	return string(runes[:publicationEventTextLimit])
}

func generatedPublicationEventID() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("random publication event id: %w", err)
	}
	return "pe" + hex.EncodeToString(data), nil
}

func validPublicationOperation(value string) bool {
	switch value {
	case "publish", "rollback", "unpublish":
		return true
	default:
		return false
	}
}
