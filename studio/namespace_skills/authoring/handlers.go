package authoring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/viant/datly-studio/internal/namespaceskill"
	"github.com/viant/datly-studio/sdk"
	heads "github.com/viant/datly-studio/studio/namespace_skills/store_read"
	xhandler "github.com/viant/xdatly/handler"
)

func runList(ctx context.Context, session xhandler.Session, in *ListInput, out *ListOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	limit, err := pageSize(in.Limit, in.Offset)
	if err != nil {
		return err
	}
	rows, err := store.heads(ctx, "", "", in.Query, limit, in.Offset)
	if err != nil {
		return err
	}
	out.Items = make([]*sdk.NamespaceSkill, 0, len(rows))
	for _, row := range rows {
		out.Items = append(out.Items, summary(row))
	}
	return nil
}
func runGet(ctx context.Context, session xhandler.Session, in *GetInput, out *GetOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	row, err := store.head(ctx, in.SkillID)
	if err != nil {
		return err
	}
	revision := in.Revision
	if revision == 0 {
		revision = row.LatestRevision
	}
	if revision < 1 || revision > row.LatestRevision {
		return publicError(404, "namespace skill revision not found")
	}
	rows, err := store.revisions(ctx, in.SkillID, revision, 2, 0)
	if err != nil {
		return err
	}
	if len(rows) != 1 {
		return publicError(404, "namespace skill revision not found")
	}
	out.Skill = summary(row)
	out.Revision = rows[0]
	return nil
}
func runCreate(ctx context.Context, session xhandler.Session, in *CreateInput, out *GetOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	var identity [16]byte
	if _, err = rand.Read(identity[:]); err != nil {
		return err
	}
	id := hex.EncodeToString(identity[:])
	snapshot, err := namespaceskill.Compile(ctx, in.NamespaceId, id, 1, in.Bundle)
	if err != nil {
		return publicError(400, err.Error())
	}
	existing, err := store.heads(ctx, "", snapshot.Name, "", 1, 0)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return publicError(409, "namespace skill name already exists")
	}
	now := time.Now().UTC()
	row := mutation(&heads.StoredSkill{NamespaceId: in.NamespaceId, SkillId: id, Name: snapshot.Name, LatestRevision: 1, Status: "active", Etag: 1, CreatedAt: now, CreatedBy: store.subject}, store.subject, now)
	revision := addRevision(row, snapshot, store.subject, now)
	out.Skill, err = store.write(ctx, row)
	if err != nil {
		return err
	}
	out.Revision = revision
	return nil
}
func runUpdate(ctx context.Context, session xhandler.Session, in *UpdateInput, out *GetOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	current, err := store.head(ctx, in.SkillID)
	if err != nil {
		return err
	}
	if in.ETag <= 0 || in.ETag != current.Etag {
		return publicError(409, "namespace skill etag does not match")
	}
	if current.LatestRevision+1 <= 0 {
		return publicError(409, "namespace skill revision exhausted")
	}
	snapshot, err := namespaceskill.Compile(ctx, in.NamespaceId, in.SkillID, current.LatestRevision+1, in.Bundle)
	if err != nil {
		return publicError(400, err.Error())
	}
	if snapshot.Name != current.Name {
		return publicError(400, "namespace skill name is immutable")
	}
	now := time.Now().UTC()
	row := mutation(current, store.subject, now)
	row.SetLatestRevision(current.LatestRevision + 1)
	revision := addRevision(row, snapshot, store.subject, now)
	out.Skill, err = store.write(ctx, row)
	if err != nil {
		return err
	}
	out.Revision = revision
	return nil
}
func runDelete(ctx context.Context, session xhandler.Session, in *DeleteInput, out *DeleteOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	current, err := store.head(ctx, in.SkillID)
	if err != nil {
		return err
	}
	if in.ETag <= 0 || in.ETag != current.Etag {
		return publicError(409, "namespace skill etag does not match")
	}
	if current.PublishedRevision > 0 {
		return publicError(409, "unpublish the namespace skill before archiving")
	}
	now := time.Now().UTC()
	row := mutation(current, store.subject, now)
	row.SetStatus("archived")
	row.SetDeletedAt(&now)
	out.Skill, err = store.write(ctx, row)
	return err
}
func runRevisions(ctx context.Context, session xhandler.Session, in *RevisionsInput, out *RevisionsOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	if _, err = store.head(ctx, in.SkillID); err != nil {
		return err
	}
	limit, err := pageSize(in.Limit, in.Offset)
	if err != nil {
		return err
	}
	out.Items, err = store.revisions(ctx, in.SkillID, 0, limit, in.Offset)
	return err
}
