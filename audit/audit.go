// Package audit provides optional, asynchronous MCP HTTP tool-call auditing.
package audit

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type Identity struct{ Subject, UserID, NamespaceID, ComponentID, TraceID, SpanID string }
type identityKey struct{}

// WithIdentity attaches trusted identity after host authentication.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

const ResponseLimit = 32 * 1024

type Record struct {
	source            *http.Request
	oversized         bool
	ResponseTruncated bool
	Identity
	ToolName, ClientIP, Status              string
	StartedAt                               time.Time
	DurationMS                              int64
	RequestBody, ResponseBody               []byte
	RequestCompression, ResponseCompression string
	RequestOriginalBytes                    int64
}
type Sink interface {
	Write(context.Context, Record) error
}
type Config struct {
	Enabled                                       bool
	QueueSize, CompressionThreshold, MaxBodyBytes int
	// Identity must return verified, host-owned identity; never decode unverified JWTs here.
	Identity func(*http.Request) Identity
	// Policy resolves the trusted resource and current persisted audit setting.
	Policy func(*http.Request, string, *Identity) bool
	// Sanitize must redact sensitive payloads. Nil stores no bodies.
	Sanitize func([]byte) []byte
	OnError  func(error)
	OnDrop   func()
}
type Logger struct {
	config  Config
	sink    Sink
	queue   chan Record
	done    chan struct{}
	mu      sync.RWMutex
	closed  bool
	enabled atomic.Bool
	dropped atomic.Uint64
	failed  atomic.Uint64
}

func New(config Config, sink Sink) *Logger {
	l := &Logger{config: config, sink: sink, done: make(chan struct{})}
	l.enabled.Store(config.Enabled)
	if sink == nil {
		close(l.done)
		return l
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 256
	}
	if config.CompressionThreshold <= 0 {
		config.CompressionThreshold = 1
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 1 << 20
	}
	l.config = config
	l.queue = make(chan Record, config.QueueSize)
	go l.run()
	return l
}
func (l *Logger) SetEnabled(enabled bool)  { l.enabled.Store(enabled) }
func (l *Logger) Losses() (uint64, uint64) { return l.dropped.Load(), l.failed.Load() }

func (l *Logger) run() {
	defer close(l.done)
	defer func() {
		if closer, ok := l.sink.(interface{ Close(context.Context) error }); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = closer.Close(ctx)
		}
	}()
	for r := range l.queue {
		if r.source != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			request := r.source.WithContext(ctx)
			namespace, traceID, spanID := r.NamespaceID, r.TraceID, r.SpanID
			if l.config.Identity != nil {
				r.Identity = l.config.Identity(request)
			}
			if namespace != "" {
				r.NamespaceID = namespace
			}
			if traceID != "" {
				r.TraceID = traceID
				r.SpanID = spanID
			}
			enabled := r.oversized || l.config.Policy == nil || l.config.Policy(request, r.ToolName, &r.Identity)
			cancel()
			r.source = nil
			if !enabled {
				continue
			}
		}
		if l.config.Sanitize != nil {
			r.RequestBody = l.config.Sanitize(r.RequestBody)
			r.ResponseBody = l.config.Sanitize(r.ResponseBody)
		} else {
			r.RequestBody = nil
			r.ResponseBody = nil
		}
		r.RequestOriginalBytes = int64(len(r.RequestBody))
		if len(r.ResponseBody) > ResponseLimit {
			r.ResponseBody = r.ResponseBody[:ResponseLimit]
			r.ResponseTruncated = true
		}
		r.RequestBody, r.RequestCompression = compress(r.RequestBody, l.config.CompressionThreshold)
		r.ResponseBody, r.ResponseCompression = compress(r.ResponseBody, l.config.CompressionThreshold)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := l.sink.Write(ctx, r)
		cancel()
		if err != nil {
			l.failed.Add(1)
			if l.config.OnError != nil {
				l.config.OnError(err)
			}
		}
	}
}
func compress(body []byte, threshold int) ([]byte, string) {
	if len(body) < threshold {
		return body, "none"
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(body)
	_ = w.Close()
	return b.Bytes(), "gzip"
}
func (l *Logger) Close(ctx context.Context) error {
	l.mu.Lock()
	if !l.closed {
		l.closed = true
		if l.queue != nil {
			close(l.queue)
		}
	}
	l.mu.Unlock()
	select {
	case <-l.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (l *Logger) enqueue(r Record) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return
	}
	select {
	case l.queue <- r:
	default:
		l.dropped.Add(1)
		if l.config.OnDrop != nil {
			l.config.OnDrop()
		}
	}
}
func (l *Logger) Wrap(next http.Handler) http.Handler {
	if l.queue == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !l.enabled.Load() || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, int64(l.config.MaxBodyBytes)+1))
		// Replay every byte already consumed, including a partial read, to the handler.
		r.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(body), r.Body), Closer: r.Body}
		oversized := err != nil || len(body) > l.config.MaxBodyBytes
		var call struct {
			Method string `json:"method"`
			Params struct {
				Name      string `json:"name"`
				Arguments struct {
					NamespaceID string `json:"namespaceId"`
				} `json:"arguments"`
			} `json:"params"`
		}
		if oversized {
			call.Method = r.Header.Get("Mcp-Method")
			call.Params.Name = r.Header.Get("Mcp-Name")
			if strings.HasSuffix(r.URL.Path, "/mcp") {
				call.Method = "tools/call"
				if call.Params.Name == "" {
					call.Params.Name = "[oversized request]"
				}
			}
			body = nil
		} else {
			_ = json.Unmarshal(body, &call)
		}
		if call.Method != "tools/call" || call.Params.Name == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-Studio-Namespace") == "" && call.Params.Arguments.NamespaceID != "" {
			r = r.Clone(r.Context())
			r.Header.Set("X-Studio-Namespace", call.Params.Arguments.NamespaceID)
		}
		record := Record{ToolName: call.Params.Name, StartedAt: time.Now().UTC(), RequestBody: body}
		record.Identity, _ = r.Context().Value(identityKey{}).(Identity)
		record.source = r.Clone(context.Background())
		record.source.Body = nil
		record.oversized = oversized
		if scope, ok := r.Context().Value(scopeKey{}).(string); ok && scope != "" {
			record.NamespaceID = scope
		}

		if span := trace.SpanContextFromContext(r.Context()); span.IsValid() {
			record.TraceID = span.TraceID().String()
			record.SpanID = span.SpanID().String()
		}
		record.ClientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		if record.ClientIP == "" {
			record.ClientIP = r.RemoteAddr
		}
		capture := &responseCapture{ResponseWriter: w, max: l.config.MaxBodyBytes, status: 200}
		defer func() {
			record.DurationMS = time.Since(record.StartedAt).Milliseconds()
			record.Status = http.StatusText(capture.status)
			record.ResponseTruncated = capture.overflow
			if toolError(capture.body.Bytes()) {
				record.Status = "Tool error"
			}
			if !capture.overflow {
				record.ResponseBody = append([]byte(nil), capture.body.Bytes()...)
			}
			l.enqueue(record)
		}()
		next.ServeHTTP(capture, r)
	})
}

type replayBody struct {
	io.Reader
	io.Closer
}
type responseCapture struct {
	http.ResponseWriter
	body        bytes.Buffer
	max, status int
	overflow    bool
}

func (w *responseCapture) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *responseCapture) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseCapture) Write(b []byte) (int, error) {
	n, e := w.ResponseWriter.Write(b)
	if !w.overflow {
		if w.body.Len()+n > w.max {
			w.overflow = true
			w.body.Reset()
		} else {
			_, _ = w.body.Write(b[:n])
		}
	}
	return n, e
}
func (w *responseCapture) Flush() { _ = http.NewResponseController(w.ResponseWriter).Flush() }

// RedactJSON redacts JSON or JSON data frames in an MCP SSE response.
func RedactJSON(body []byte) []byte {
	var v any
	if json.Unmarshal(body, &v) != nil {
		if bytes.Contains(body, []byte("data:")) {
			var result bytes.Buffer
			for _, line := range bytes.Split(body, []byte("\n")) {
				if bytes.HasPrefix(line, []byte("data:")) {
					data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
					redacted := RedactJSON(data)
					if redacted == nil {
						return nil
					}
					result.WriteString("data: ")
					result.Write(redacted)
					result.WriteByte('\n')
				} else if len(bytes.TrimSpace(line)) == 0 {
					result.WriteByte('\n')
				}
			}
			return result.Bytes()
		}
		return nil
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				lower := strings.ToLower(k)
				if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || lower == "authorization" {
					x[k] = "[REDACTED]"
				} else if text, ok := val.(string); ok && (strings.HasPrefix(strings.TrimSpace(text), "{") || strings.HasPrefix(strings.TrimSpace(text), "[")) {
					if clean := RedactJSON([]byte(text)); clean != nil {
						x[k] = string(clean)
					} else {
						x[k] = "[OMITTED]"
					}
				} else {
					walk(val)
				}
			}
		case []any:
			for _, val := range x {
				walk(val)
			}
		}
	}
	walk(v)
	b, _ := json.Marshal(v)
	return b
}

// WithNamespace is set by the trusted namespace listener, not client arguments.
func WithNamespace(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, scopeKey{}, id)
}

type scopeKey struct{}

func toolError(body []byte) bool {
	var response struct {
		Error  json.RawMessage `json:"error"`
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &response) == nil {
		return len(response.Error) > 0 && string(response.Error) != "null" || response.Result.IsError
	}
	for _, line := range bytes.Split(body, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("data:")) && toolError(bytes.TrimSpace(line[5:])) {
			return true
		}
	}
	return false
}
