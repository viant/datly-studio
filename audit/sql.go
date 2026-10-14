package audit

import (
	"context"
	"database/sql"
)

// SQLSink inserts into the explicitly provisioned MySQL audit table.
// It performs no DDL and uses native AUTO_INCREMENT for the identity.
type SQLSink struct{ DB *sql.DB }

func (s SQLSink) Write(ctx context.Context, r Record) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO tool_call_audit
 (namespace_id,tool_name,started_at,duration_ms,status,client_ip,jwt_sub,user_id,trace_id,span_id,
 request_body,request_compression,request_original_bytes,response_body,response_compression,response_original_bytes)
 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, nullable(r.NamespaceID), r.ToolName, r.StartedAt, r.DurationMS, r.Status, nullable(r.ClientIP), nullable(r.Subject), nullable(r.UserID), nullable(r.TraceID), nullable(r.SpanID), r.RequestBody, r.RequestCompression, r.RequestOriginalBytes, r.ResponseBody, r.ResponseCompression, r.ResponseOriginalBytes)
	return err
}
func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}
