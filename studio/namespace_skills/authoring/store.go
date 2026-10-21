package authoring

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespaceskill"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	heads "github.com/viant/datly-studio/studio/namespace_skills/store_read"
	revisions "github.com/viant/datly-studio/studio/namespace_skills/store_revisions"
	writer "github.com/viant/datly-studio/studio/namespace_skills/store_write"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type store struct {
	invoker              exec.ComponentInvoker
	namespaceID, subject string
}

func authorize(ctx context.Context, session xhandler.Session, namespaceID string, claims *jwt.Claims, auth *studioauth.Output) (*store, error) {
	if claims == nil || auth == nil || auth.Auth == nil || claims.Subject == "" || auth.Auth.Subject != claims.Subject {
		return nil, publicError(403, "verified Studio principal is required")
	}
	if namespaceaccess.ValidateResourceOwnership(namespaceID, "") != nil {
		return nil, publicError(400, "valid namespace selection is required")
	}
	if session == nil || session.Binder() == nil {
		return nil, fmt.Errorf("namespace skill session is unavailable")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, fmt.Errorf("Datly component invoker is unavailable")
	}
	var headers struct {
		Authorization string `bind:"kind=header,in=Authorization"`
	}
	if err = session.Binder().Bind(ctx, &headers); err != nil {
		return nil, err
	}
	ctx = oauth.WithBearer(ctx, strings.TrimPrefix(headers.Authorization, "Bearer "))
	input := &namespaces.Input{}
	input.NamespaceId = namespaceID
	input.SetScoped(false)
	input.Has.NamespaceId = true
	input.SetStatus("active")
	input.SetPageLimit(2)
	input.SetPageOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: input})
	if err != nil {
		return nil, err
	}
	page, ok := value.(*namespaces.Output)
	if !ok || page == nil {
		return nil, fmt.Errorf("namespace reader returned %T", value)
	}
	if len(page.Namespaces) != 1 || page.Namespaces[0] == nil || page.Namespaces[0].NamespaceId != namespaceID || !namespacevisibility.CanManage(ctx, claims.Subject, page.Namespaces[0].OwnerId, page.Namespaces[0].AllowedRolesJson) {
		return nil, publicError(403, "namespace management denied")
	}
	return &store{invoker: invoker, namespaceID: namespaceID, subject: claims.Subject}, nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}
func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message)}
}
func pageSize(limit, offset int) (int, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return 0, publicError(400, "limit must be 1 to 100 and offset nonnegative")
	}
	return limit, nil
}

func (s *store) heads(ctx context.Context, id, name, query string, limit, offset int) ([]*heads.StoredSkill, error) {
	input := &heads.Input{}
	input.SetNamespaceId(s.namespaceID)
	input.SetPageLimit(limit)
	input.SetPageOffset(offset)
	if id != "" {
		input.SetSkillId(id)
	}
	if name != "" {
		input.SetName(name)
	}
	if query != "" {
		input.SetQuery(query)
	}
	value, err := s.invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[heads.SkillComponent](), "skill", "GET", "/_studio/namespace-skill-store/read"), Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*heads.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("namespace skill reader returned %T", value)
	}
	for _, row := range output.Skills {
		if row == nil || row.NamespaceId != s.namespaceID || id != "" && row.SkillId != id || name != "" && row.Name != name {
			return nil, fmt.Errorf("namespace skill reader returned mismatched identity")
		}
	}
	return output.Skills, nil
}
func (s *store) head(ctx context.Context, id string) (*heads.StoredSkill, error) {
	if id == "" {
		return nil, publicError(400, "skillId is required")
	}
	rows, err := s.heads(ctx, id, "", "", 2, 0)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, publicError(404, "namespace skill not found")
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("ambiguous namespace skill identity")
	}
	return rows[0], nil
}
func (s *store) revisions(ctx context.Context, id string, revision, limit, offset int) ([]*sdk.NamespaceSkillRevision, error) {
	input := &revisions.Input{}
	input.SetNamespaceId(s.namespaceID)
	input.SetSkillId(id)
	input.SetPageLimit(limit)
	input.SetPageOffset(offset)
	if revision > 0 {
		input.SetRevision(revision)
	}
	value, err := s.invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[revisions.SkillComponent](), "skill", "GET", "/_studio/namespace-skill-store/revisions"), Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*revisions.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("namespace skill revisions returned %T", value)
	}
	result := make([]*sdk.NamespaceSkillRevision, 0, len(output.Revisions))
	for _, row := range output.Revisions {
		if row == nil || row.NamespaceId != s.namespaceID || row.SkillId != id || revision > 0 && row.Revision != revision {
			return nil, fmt.Errorf("namespace skill revision reader returned mismatched identity")
		}
		snapshot, err := namespaceskill.Restore(ctx, row.NamespaceId, row.SkillId, row.Revision, row.BundleJson, row.ContentSha256)
		if err != nil {
			return nil, err
		}
		result = append(result, &sdk.NamespaceSkillRevision{NamespaceID: row.NamespaceId, SkillID: row.SkillId, Revision: row.Revision, ContentSHA256: row.ContentSha256, Bundle: snapshot.Bundle, CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy})
	}
	return result, nil
}
func (s *store) write(ctx context.Context, row *writer.StoredSkill) (*sdk.NamespaceSkill, error) {
	return s.writeObserved(ctx, row, false)
}
func (s *store) writeCommitted(ctx context.Context, row *writer.StoredSkill) (*sdk.NamespaceSkill, error) {
	return s.writeObserved(ctx, row, true)
}
func (s *store) writeObserved(ctx context.Context, row *writer.StoredSkill, requireCommit bool) (*sdk.NamespaceSkill, error) {
	var committed bool
	var completion func(xhandler.Outcome)
	if requireCommit {
		completion = func(outcome xhandler.Outcome) { committed = outcome.CommitConfirmed() }
	}

	input := &writer.Input{}
	input.SetSkills([]*writer.StoredSkill{row})
	value, err := s.invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[writer.SkillComponent](), "skill", "PATCH", "/_studio/namespace-skill-store/write"), Input: input, Completion: completion})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return nil, publicError(409, "namespace skill etag does not match")
		}
		return nil, err
	}
	if requireCommit && !committed {
		return nil, publicationCommitError()
	}
	output, ok := value.(*writer.Output)
	if !ok || output == nil || len(output.Data) != 1 || output.Data[0] == nil || output.Data[0].NamespaceId != s.namespaceID || output.Data[0].SkillId != row.SkillId || output.Data[0].Etag == nil {
		return nil, fmt.Errorf("namespace skill writer returned incomplete or mismatched identity")
	}
	changed := output.Data[0]
	return &sdk.NamespaceSkill{NamespaceID: changed.NamespaceId, SkillID: changed.SkillId, Name: changed.Name, Title: changed.Title, Description: changed.Description, LatestRevision: changed.LatestRevision, PublishedRevision: changed.PublishedRevision, Status: changed.Status, ETag: *changed.Etag, CreatedAt: changed.CreatedAt, UpdatedAt: changed.UpdatedAt}, nil
}
func summary(row *heads.StoredSkill) *sdk.NamespaceSkill {
	return &sdk.NamespaceSkill{NamespaceID: row.NamespaceId, SkillID: row.SkillId, Name: row.Name, Title: row.Title, Description: row.Description, LatestRevision: row.LatestRevision, PublishedRevision: row.PublishedRevision, Status: row.Status, ETag: row.Etag, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func mutation(row *heads.StoredSkill, subject string, now time.Time) *writer.StoredSkill {
	result := &writer.StoredSkill{}
	result.SetNamespaceId(row.NamespaceId)
	result.SetSkillId(row.SkillId)
	result.SetName(row.Name)
	result.SetTitle(row.Title)
	result.SetDescription(row.Description)
	result.SetLatestRevision(row.LatestRevision)
	result.SetPublishedRevision(row.PublishedRevision)
	result.SetStatus(row.Status)
	result.SetCreatedAt(row.CreatedAt)
	result.SetCreatedBy(row.CreatedBy)
	result.SetUpdatedAt(now)
	result.SetUpdatedBy(subject)
	result.SetDeletedAt(row.DeletedAt)
	etag := row.Etag
	result.SetEtag(&etag)
	return result
}
func addRevision(row *writer.StoredSkill, snapshot *namespaceskill.Snapshot, subject string, now time.Time) *sdk.NamespaceSkillRevision {
	revision := &writer.StoredRevision{}
	revision.SetNamespaceId(row.NamespaceId)
	revision.SetSkillId(row.SkillId)
	revision.SetRevision(row.LatestRevision)
	revision.SetBundleJson(snapshot.JSON)
	revision.SetContentSha256(snapshot.SHA256)
	revision.SetCreatedAt(now)
	revision.SetUpdatedAt(now)
	revision.SetCreatedBy(subject)
	revision.SetUpdatedBy(subject)
	row.SetRevisions([]*writer.StoredRevision{revision})
	row.SetTitle(snapshot.Bundle.Title)
	row.SetDescription(snapshot.Description)
	return &sdk.NamespaceSkillRevision{NamespaceID: row.NamespaceId, SkillID: row.SkillId, Revision: row.LatestRevision, ContentSHA256: snapshot.SHA256, Bundle: snapshot.Bundle, CreatedAt: now, CreatedBy: subject}
}
