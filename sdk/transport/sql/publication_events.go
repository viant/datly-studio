package sqltransport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/viant/datly-studio/sdk"
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
// trail. Authorization is applied before this method by Transport.authorize.
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
	where := " WHERE report_id=?"
	args := []any{in.ReportID}
	if operation := strings.TrimSpace(in.Input.Operation); operation != "" {
		if !validPublicationOperation(operation) {
			return invalid(errors.New("operation must be publish, rollback, or unpublish"))
		}
		where += " AND operation=?"
		args = append(args, operation)
	}
	if status := strings.TrimSpace(in.Input.Status); status != "" {
		if status != "succeeded" && status != "failed" {
			return invalid(errors.New("status must be succeeded or failed"))
		}
		where += " AND status=?"
		args = append(args, status)
	}
	args = append(args, limit, in.Input.Offset)
	rows, err := t.DB.QueryContext(ctx, publicationEventSelect+where+" ORDER BY occurred_at DESC,event_id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.PublicationEventPage{Limit: limit, Offset: in.Input.Offset}
	for rows.Next() {
		value, scanErr := scanPublicationEvent(rows)
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

const publicationEventSelect = `SELECT event_id,report_id,owner_id,operation,version_no,generation_no,status,requested_by,reason,failure_code,failure_message,occurred_at FROM report_publication_events`

func scanPublicationEvent(scanner interface{ Scan(...any) error }) (*sdk.PublicationEvent, error) {
	var value sdk.PublicationEvent
	var version, generation sql.NullInt64
	var reason, failureCode, failureMessage sql.NullString
	if err := scanner.Scan(&value.EventID, &value.ReportID, &value.OwnerID, &value.Operation, &version, &generation, &value.Status, &value.RequestedBy, &reason, &failureCode, &failureMessage, &value.OccurredAt); err != nil {
		return nil, err
	}
	if version.Valid {
		item := int(version.Int64)
		value.VersionNo = &item
	}
	if generation.Valid {
		item := generation.Int64
		value.GenerationNo = &item
	}
	if reason.Valid {
		value.Reason = reason.String
	}
	if failureCode.Valid {
		value.FailureCode = failureCode.String
	}
	if failureMessage.Valid {
		value.FailureMessage = failureMessage.String
	}
	return &value, nil
}

func (t *Transport) publicationEventOwner(ctx context.Context, reportID string) (string, error) {
	var owner string
	err := t.DB.QueryRowContext(ctx, `SELECT owner_id FROM reports WHERE id=? AND deleted_at IS NULL`, reportID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", &sdk.Error{Code: sdk.ErrorNotFound, Message: "report not found"}
	}
	if err != nil {
		return "", internal(err)
	}
	return owner, nil
}

func (t *Transport) appendPublicationEvent(ctx context.Context, event publicationEventRecord) error {
	return t.appendPublicationEventDB(ctx, t.DB, event)
}

func (t *Transport) appendPublicationEventTx(ctx context.Context, tx *sql.Tx, event publicationEventRecord) error {
	return t.appendPublicationEventDB(ctx, tx, event)
}

type publicationEventExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (t *Transport) appendPublicationEventDB(ctx context.Context, executor publicationEventExecer, event publicationEventRecord) error {
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
	if _, err = executor.ExecContext(ctx, `INSERT INTO report_publication_events(event_id,report_id,owner_id,operation,version_no,generation_no,status,requested_by,reason,failure_code,failure_message,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, eventID, event.ReportID, event.OwnerID, event.Operation, nullableInt(event.VersionNo), nullableInt64(event.GenerationNo), event.Status, event.RequestedBy, nullable(event.Reason), nullable(event.FailureCode), nullable(event.Failure), event.OccurredAt); err != nil {
		return err
	}
	return nil
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

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
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
