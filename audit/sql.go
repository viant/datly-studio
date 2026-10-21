package audit

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/internal/auditstore"
	stored "github.com/viant/datly-studio/studio/tool_call_audit/store_write"
	"sync"
)

// SQLSink invokes the generated Datly v1 insert component. The logger's worker
// owns its calls; this adapter never issues INSERT SQL or allocates IDs itself.
type SQLSink struct {
	DB    *sql.DB
	once  sync.Once
	store *auditstore.Store
	err   error
}

func (s *SQLSink) Write(ctx context.Context, r Record) error {
	s.once.Do(func() { s.store, s.err = auditstore.New(s.DB) })
	if s.err != nil {
		return s.err
	}
	row := &stored.ToolCallAudit{}
	row.SetNamespaceId(optionalString(r.NamespaceID))
	row.SetComponentId(optionalString(r.ComponentID))
	row.SetToolName(r.ToolName)
	row.SetStartedAt(r.StartedAt)
	row.SetDurationMs(&r.DurationMS)
	row.SetStatus(r.Status)
	row.SetClientIp(optionalString(r.ClientIP))
	row.SetJwtSub(optionalString(r.Subject))
	row.SetUserId(optionalString(r.UserID))
	row.SetTraceId(optionalString(r.TraceID))
	row.SetSpanId(optionalString(r.SpanID))
	row.SetRequestBody(r.RequestBody)
	row.SetRequestCompression(r.RequestCompression)
	row.SetRequestOriginalBytes(&r.RequestOriginalBytes)
	row.SetResponseBody(r.ResponseBody)
	row.SetResponseCompression(r.ResponseCompression)
	row.SetResponseTruncated(r.ResponseTruncated)
	return s.store.Insert(ctx, row)
}
func (s *SQLSink) Close(ctx context.Context) error { return s.store.Close(ctx) }
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
