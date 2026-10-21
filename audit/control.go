package audit

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/auditstore"
	"github.com/viant/datly-studio/internal/namespaceroles"
	storedread "github.com/viant/datly-studio/studio/tool_call_audit/store_read"
)

// Service owns the audit infrastructure and its trusted identity provider.
// No global toggle is exposed: management is always scoped to a resource.
type Service struct {
	DB         *sql.DB
	Provider   authz.Provider
	ViewerRole string
	Logger     *Logger
	mu         sync.Mutex
	queryOnce  sync.Once
	queryStore *auditstore.Store
	queryErr   error
}
type serviceKey struct{}

func WithService(ctx context.Context, s *Service) context.Context {
	return context.WithValue(ctx, serviceKey{}, s)
}
func FromContext(ctx context.Context) *Service { s, _ := ctx.Value(serviceKey{}).(*Service); return s }

type Scope struct {
	NamespaceID string `json:"namespaceId"`
	ComponentID string `json:"componentId,omitempty"`
	ToolName    string `json:"toolName,omitempty"`
}
type Setting struct {
	Scope
	Enabled    bool   `json:"enabled"`
	Effective  bool   `json:"effective"`
	CanManage  bool   `json:"canManage"`
	CanView    bool   `json:"canView"`
	ViewerRole string `json:"viewerRole"`
}
type Filter struct {
	NamespaceID string `json:"namespaceId,omitempty"`
	ToolName    string `json:"toolName,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Status      string `json:"status,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	BeforeID    int64  `json:"beforeId,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	ID          int64  `json:"id,omitempty"`
}
type Event struct {
	ID                int64     `json:"id"`
	NamespaceID       string    `json:"namespaceId"`
	ComponentID       string    `json:"componentId,omitempty"`
	ToolName          string    `json:"toolName"`
	StartedAt         time.Time `json:"startedAt"`
	DurationMS        int64     `json:"durationMs"`
	Status            string    `json:"status"`
	Subject           string    `json:"subject"`
	ClientIP          string    `json:"clientIp"`
	ResponseTruncated bool      `json:"responseTruncated"`
	Request           string    `json:"request,omitempty"`
	Response          string    `json:"response,omitempty"`
}
type Page struct {
	Items   []Event `json:"items"`
	NextID  int64   `json:"nextId,omitempty"`
	Dropped uint64  `json:"dropped"`
	Failed  uint64  `json:"failed"`
}

var ErrForbidden = errors.New("audit access denied")
var ErrInvalid = errors.New("invalid audit scope or filter")

func (s *Service) facts(ctx context.Context) (authz.Facts, error) {
	if s.Provider == nil {
		return authz.Facts{}, ErrForbidden
	}
	facts, err := s.Provider.Resolve(ctx)
	if err != nil || facts.Subject == "" || facts.Issuer == "" || !facts.ValidUntil.After(time.Now()) {
		return authz.Facts{}, ErrForbidden
	}
	return facts, nil
}
func (s *Service) viewer(f authz.Facts) bool {
	return s.ViewerRole != "" && s.ViewerRole != "*" && namespaceroles.Matches([]string{s.ViewerRole}, f.Roles)
}

// Identity uses verified provider facts, never unverified JWT payloads.
func (s *Service) Identity(r *http.Request) Identity {
	ctx := oauth.WithBearer(r.Context(), strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
	facts, err := s.facts(ctx)
	if err != nil {
		return Identity{}
	}
	return Identity{Subject: facts.Subject}
}

// Policy failures preserve auditing. A forged or unknown resource cannot disable capture.
func (s *Service) Policy(r *http.Request, tool string, identity *Identity) bool {
	namespace := identity.NamespaceID
	if namespace == "" {
		namespace = r.Header.Get("X-Studio-Namespace")
	}
	identity.NamespaceID = namespace
	if namespace == "" {
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	var enabled bool
	if err := s.DB.QueryRowContext(ctx, "SELECT audit_enabled FROM namespaces WHERE namespace_id=? AND deleted_at IS NULL", namespace).Scan(&enabled); err != nil {
		return true
	}
	if !enabled {
		return false
	}
	component, componentEnabled, err := s.toolComponent(ctx, namespace, tool)
	if err == nil {
		identity.ComponentID = component
		if !componentEnabled {
			return false
		}
	}
	var toolEnabled bool
	if err := s.DB.QueryRowContext(ctx, "SELECT enabled FROM tool_audit_settings WHERE namespace_id=? AND tool_name=?", namespace, tool).Scan(&toolEnabled); err == nil {
		return toolEnabled
	}
	return true
}

// The tool inventory is taken from the active publication, including Datly's
// derived cube names only when those exposures are enabled.
func (s *Service) toolComponent(ctx context.Context, namespace, tool string) (string, bool, error) {
	var id string
	var enabled bool
	err := s.DB.QueryRowContext(ctx, `SELECT c.id,c.audit_enabled FROM components c
 JOIN namespaces n ON n.owner_id=c.owner_id AND n.name=c.namespace
 JOIN component_publications p ON p.report_id=c.id AND p.active_generation IS NOT NULL
 JOIN component_mcp_exposures e ON e.report_id=c.id AND e.version_no=p.active_version_no
 LEFT JOIN component_cube_configs cube ON cube.report_id=e.report_id AND cube.version_no=e.version_no
 WHERE n.namespace_id=? AND c.deleted_at IS NULL AND e.enabled=TRUE AND e.kind='tool'
 AND (e.name=? OR (cube.cube_enabled=TRUE AND ?=CONCAT(e.name,'Cube')) OR (cube.compose_enabled=TRUE AND ?=CONCAT(e.name,'CubeCompose')))
 LIMIT 1`, namespace, tool, tool, tool).Scan(&id, &enabled)
	return id, enabled, err
}

func (s *Service) setting(ctx context.Context, scope Scope, facts authz.Facts) (Setting, error) {
	result := Setting{Scope: scope, Enabled: true, Effective: true, ViewerRole: s.ViewerRole, CanView: s.viewer(facts)}
	if scope.NamespaceID == "" || len(scope.ToolName) > 255 || len(scope.ComponentID) > 64 {
		return result, ErrInvalid
	}
	var owner, roles, visibility string
	var namespaceEnabled bool
	err := s.DB.QueryRowContext(ctx, `SELECT owner_id,allowed_roles_json,visibility,audit_enabled FROM namespaces WHERE namespace_id=? AND deleted_at IS NULL AND status='active'`, scope.NamespaceID).Scan(&owner, &roles, &visibility, &namespaceEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrForbidden
	}
	if err != nil {
		return result, err
	}
	grants, err := namespaceroles.Decode(&roles)
	if err != nil {
		return result, ErrForbidden
	}
	result.CanManage = facts.Subject == owner || namespaceroles.Matches(grants.Manage, facts.Roles)
	visible := result.CanManage || visibility == "public" || namespaceroles.Matches(grants.View, facts.Roles)
	result.Enabled = namespaceEnabled
	result.Effective = namespaceEnabled
	component := scope.ComponentID
	if scope.ToolName != "" {
		discovered, _, err := s.toolComponent(ctx, scope.NamespaceID, scope.ToolName)
		if err != nil {
			// Administrative tools have namespace-wide settings controlled by its manager.
			if !strings.HasPrefix(scope.ToolName, "studio.sdk.") || !result.CanManage {
				return result, ErrForbidden
			}
		} else {
			if component != "" && component != discovered {
				return result, ErrInvalid
			}
			component = discovered
		}
	}
	if component != "" {
		var componentOwner string
		var enabled bool
		err = s.DB.QueryRowContext(ctx, `SELECT c.owner_id,c.audit_enabled FROM components c JOIN namespaces n ON n.owner_id=c.owner_id AND n.name=c.namespace WHERE c.id=? AND n.namespace_id=? AND c.deleted_at IS NULL`, component, scope.NamespaceID).Scan(&componentOwner, &enabled)
		if err != nil {
			return result, ErrForbidden
		}
		result.ComponentID = component
		result.CanManage = result.CanManage || componentOwner == facts.Subject
		if !result.CanManage {
			rows, err := s.DB.QueryContext(ctx, `SELECT subject_type,subject_id FROM component_acl WHERE report_id=? AND (can_edit=TRUE OR can_publish=TRUE)`, component)
			if err != nil {
				return result, err
			}
			for rows.Next() {
				var kind, id string
				if err = rows.Scan(&kind, &id); err != nil {
					rows.Close()
					return result, err
				}
				if kind == "user" && id == facts.Subject || kind == "role" && namespaceroles.Matches([]string{id}, facts.Roles) {
					result.CanManage = true
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return result, err
			}
		}
		result.Enabled = enabled
		result.Effective = namespaceEnabled && enabled
	}
	if !visible && !result.CanManage {
		return result, ErrForbidden
	}
	if scope.ToolName != "" {
		toolEnabled := true
		err = s.DB.QueryRowContext(ctx, `SELECT enabled FROM tool_audit_settings WHERE namespace_id=? AND tool_name=?`, scope.NamespaceID, scope.ToolName).Scan(&toolEnabled)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		result.Enabled = toolEnabled
		result.Effective = result.Effective && toolEnabled
	}
	return result, nil
}
func (s *Service) Status(ctx context.Context, scope Scope) (Setting, error) {
	f, e := s.facts(ctx)
	if e != nil {
		return Setting{}, e
	}
	return s.setting(ctx, scope, f)
}
func (s *Service) Configure(ctx context.Context, scope Scope, enabled bool) (Setting, error) {
	facts, err := s.facts(ctx)
	if err != nil {
		return Setting{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.setting(ctx, scope, facts)
	if err != nil {
		return current, err
	}
	if !current.CanManage {
		return current, ErrForbidden
	}
	now := time.Now().UTC()
	switch {
	case scope.ToolName != "":
		// Use an explicit transaction for a portable SQLite/MySQL upsert.
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return current, err
		}
		defer tx.Rollback()
		var exists int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tool_audit_settings WHERE namespace_id=? AND tool_name=?", scope.NamespaceID, scope.ToolName).Scan(&exists); err != nil {
			return current, err
		}
		if exists == 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO tool_audit_settings(namespace_id,tool_name,enabled,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,?)`, scope.NamespaceID, scope.ToolName, enabled, now, facts.Subject, now, facts.Subject)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE tool_audit_settings SET enabled=?,updated_at=?,updated_by=? WHERE namespace_id=? AND tool_name=?`, enabled, now, facts.Subject, scope.NamespaceID, scope.ToolName)
		}
		if err != nil {
			return current, err
		}
		if err = tx.Commit(); err != nil {
			return current, err
		}
	case scope.ComponentID != "":
		_, err = s.DB.ExecContext(ctx, `UPDATE components SET audit_enabled=?,etag=etag+1,updated_at=? WHERE id=? AND deleted_at IS NULL`, enabled, now, scope.ComponentID)
	default:
		_, err = s.DB.ExecContext(ctx, `UPDATE namespaces SET audit_enabled=?,etag=etag+1,updated_at=? WHERE namespace_id=? AND deleted_at IS NULL`, enabled, now, scope.NamespaceID)
	}
	if err != nil {
		return current, err
	}
	// Control changes remain auditable even when they disable their own scope.
	if s.Logger != nil {
		s.Logger.enqueue(Record{Identity: Identity{Subject: facts.Subject, NamespaceID: scope.NamespaceID, ComponentID: current.ComponentID}, ToolName: "audit.configuration", StartedAt: now, Status: fmt.Sprintf("enabled=%t", enabled), RequestBody: []byte(fmt.Sprintf(`{"toolName":%q,"enabled":%t}`, scope.ToolName, enabled))})
	}
	return s.setting(ctx, scope, facts)
}
func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	page := Page{Items: []Event{}}
	facts, err := s.facts(ctx)
	if err != nil || !s.viewer(facts) {
		return page, ErrForbidden
	}
	if filter.NamespaceID != "" {
		if _, err = s.setting(ctx, Scope{NamespaceID: filter.NamespaceID}, facts); err != nil {
			return page, err
		}
	}
	if filter.From != "" && filter.To != "" {
		from, e1 := time.Parse(time.RFC3339, filter.From)
		to, e2 := time.Parse(time.RFC3339, filter.To)
		if e1 != nil || e2 != nil || from.After(to) {
			return page, ErrInvalid
		}
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	input := &storedread.Input{}
	input.SetNamespaceId(filter.NamespaceID)
	input.SetIncludeBodies(filter.ID > 0)

	if filter.ToolName != "" {
		input.SetToolName(filter.ToolName)
	}
	if filter.Subject != "" {
		input.SetSubject(filter.Subject)
	}
	if filter.Status != "" {
		input.SetStatus(filter.Status)
	}
	if filter.ID > 0 {
		input.SetId(filter.ID)
	}
	if filter.BeforeID > 0 {
		input.SetBeforeId(filter.BeforeID)
	}
	if filter.From != "" {
		value, err := time.Parse(time.RFC3339, filter.From)
		if err != nil {
			return page, ErrInvalid
		}
		input.SetFrom(value.UTC())
	}
	if filter.To != "" {
		value, err := time.Parse(time.RFC3339, filter.To)
		if err != nil {
			return page, ErrInvalid
		}
		input.SetTo(value.UTC())
	}
	s.queryOnce.Do(func() { s.queryStore, s.queryErr = auditstore.New(s.DB) })
	if s.queryErr != nil {
		return page, s.queryErr
	}
	rows, err := s.queryStore.Read(ctx, input)
	if err != nil {
		return page, err
	}
	for _, row := range rows {
		if len(page.Items) == filter.Limit {
			page.NextID = page.Items[len(page.Items)-1].ID
			break
		}
		event := Event{ID: row.Id, NamespaceID: stringValue(row.NamespaceId), ComponentID: stringValue(row.ComponentId), ToolName: stringValue(row.ToolName), StartedAt: row.StartedAt, Status: stringValue(row.Status), Subject: stringValue(row.JwtSub), ClientIP: stringValue(row.ClientIp), ResponseTruncated: row.ResponseTruncated}
		if row.DurationMs != nil {
			event.DurationMS = int64(*row.DurationMs)
		}
		if filter.ID > 0 {
			event.Request = decodeBody(row.RequestBody, stringValue(row.RequestCompression), 1<<20)
			event.Response = decodeBody(row.ResponseBody, stringValue(row.ResponseCompression), ResponseLimit)
		}
		page.Items = append(page.Items, event)
	}

	if s.Logger != nil {
		page.Dropped, page.Failed = s.Logger.Losses()
	}
	return page, nil
}
func decodeBody(body []byte, compression string, limit int) string {
	var reader io.Reader = bytes.NewReader(body)
	if compression == "gzip" {
		z, err := gzip.NewReader(reader)
		if err != nil {
			return "[unavailable]"
		}
		defer z.Close()
		reader = z
	}
	result, err := io.ReadAll(io.LimitReader(reader, int64(limit)))
	if err != nil {
		return "[unavailable]"
	}
	return string(result)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func (s *Service) Close(ctx context.Context) error { return s.queryStore.Close(ctx) }
