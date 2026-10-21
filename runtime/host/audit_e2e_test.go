package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/audit"
	"github.com/viant/datly-studio/schema"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

type auditGateSink struct {
	next audit.Sink
	gate <-chan struct{}
}

func (s auditGateSink) Write(ctx context.Context, r audit.Record) error {
	select {
	case <-s.gate:
		return s.next.Write(ctx, r)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestToolCallAuditE2E runs the real published Datly MCP reader over TCP and
// verifies asynchronous SQLite persistence. Endly invokes this focused case.
func TestToolCallAuditE2E(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err = schema.ApplySQLite(context.Background(), db, "tool_call_audit"); err != nil {
				t.Fatal(err)
			}
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			claims := jwtv5.MapClaims{"sub": "audit-owner", "user_id": "42", "iss": "audit-e2e", "aud": "studio", "exp": time.Now().Add(time.Hour).Unix()}
			token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			gate := make(chan struct{})
			released := false
			defer func() {
				if !released {
					close(gate)
				}
			}()
			logger := audit.New(audit.Config{Enabled: enabled, CompressionThreshold: 64, Sanitize: audit.RedactJSON, Identity: func(r *http.Request) audit.Identity {
				parsed, err := jwtv5.Parse(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), func(*jwtv5.Token) (any, error) { return &key.PublicKey, nil }, jwtv5.WithValidMethods([]string{"RS256"}), jwtv5.WithIssuer("audit-e2e"), jwtv5.WithAudience("studio"))
				if err != nil || !parsed.Valid {
					return audit.Identity{}
				}
				c := parsed.Claims.(jwtv5.MapClaims)
				return audit.Identity{Subject: c["sub"].(string), UserID: c["user_id"].(string), NamespaceID: "audit-namespace", TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef"}
			}}, auditGateSink{next: &audit.SQLSink{DB: db}, gate: gate})
			defer logger.Close(context.Background())
			testDynamicHost(t, false, false, dynamicHostExtension{configure: func(c *Config) { c.ToolCallAudit = logger }, verify: func(_ *Service, address string) {
				body := `{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"records.read","arguments":{},"password":"do-not-store","_meta":{"io.modelcontextprotocol/protocolVersion":"` + mcpschema.LatestProtocolVersion + `","io.modelcontextprotocol/clientCapabilities":{}}}}`
				request, err := http.NewRequest("POST", "http://"+address+"/mcp", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set("Authorization", "Bearer "+token)
				request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
				request.Header.Set(mcpschema.HeaderMethod, "tools/call")
				request.Header.Set("Mcp-Name", "records.read")
				client := &http.Client{Timeout: 3 * time.Second}
				response, err := client.Do(request)
				if err != nil {
					t.Fatal("response waited for blocked audit writer:", err)
				}
				payload, _ := io.ReadAll(response.Body)
				response.Body.Close()
				if response.StatusCode != 200 || !bytes.Contains(payload, []byte("ready")) {
					t.Fatalf("MCP response %d %s", response.StatusCode, payload)
				}
				var count int
				if err = db.QueryRow("SELECT COUNT(*) FROM tool_call_audit").Scan(&count); err != nil || count != 0 {
					t.Fatalf("audit writer was not gated: %d %v", count, err)
				}
				close(gate)
				released = true
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err = logger.Close(ctx); err != nil {
					t.Fatal(err)
				}
				if err = db.QueryRow("SELECT COUNT(*) FROM tool_call_audit").Scan(&count); err != nil {
					t.Fatal(err)
				}
				if !enabled {
					if count != 0 {
						t.Fatal("disabled audit wrote rows")
					}
					return
				}
				if count != 1 {
					t.Fatalf("audit count=%d", count)
				}
				var name, ip, sub, user, ns, trace, span, rc, sc string
				var req, res []byte
				var rqSize int64
				err = db.QueryRow(`SELECT tool_name,client_ip,jwt_sub,user_id,namespace_id,trace_id,span_id,request_body,response_body,request_compression,response_compression,request_original_bytes FROM tool_call_audit`).Scan(&name, &ip, &sub, &user, &ns, &trace, &span, &req, &res, &rc, &sc, &rqSize)
				if err != nil {
					t.Fatal(err)
				}
				if name != "records.read" || ip != "127.0.0.1" || sub != "audit-owner" || user != "42" || ns != "audit-namespace" || len(trace) != 32 || len(span) != 16 {
					t.Fatalf("incorrect audit metadata: %s %s %s %s %s", name, ip, sub, user, ns)
				}
				decode := func(b []byte, compression string, size int64) []byte {
					if compression != "gzip" {
						t.Fatalf("compression=%s", compression)
					}
					reader, e := gzip.NewReader(bytes.NewReader(b))
					if e != nil {
						t.Fatal(e)
					}
					defer reader.Close()
					plain, e := io.ReadAll(reader)
					if e != nil || size >= 0 && int64(len(plain)) != size {
						t.Fatalf("gzip roundtrip: %v size %d/%d", e, len(plain), size)
					}
					return plain
				}
				req = decode(req, rc, rqSize)
				res = decode(res, sc, -1)
				if bytes.Contains(req, []byte("do-not-store")) || bytes.Contains(req, []byte(token)) || !bytes.Contains(req, []byte("[REDACTED]")) || !bytes.Contains(res, []byte("ready")) {
					t.Fatal("body retention/redaction failed")
				}
			}})
		})
	}
}
