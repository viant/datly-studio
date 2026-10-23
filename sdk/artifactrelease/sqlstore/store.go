// Package sqlstore persists immutable release bundles and durable channel
// projection intents in the existing Studio database. A serving adapter binds
// this store to a verified tenant/account/workspace/namespace; browser fields
// are never accepted as a substitute for that scope.
package sqlstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/forge/backend/artifactrelease"
)

// Scope is injected by a verified Studio host. The channel name is supplied
// separately, but all other Key fields must match this immutable binding.
type Scope struct {
	Tenant, Account, Workspace, Namespace string
}

func (s Scope) valid() bool {
	return (artifactrelease.Key{Tenant: s.Tenant, Account: s.Account, Workspace: s.Workspace, Namespace: s.Namespace, Name: "scope"}).Valid()
}

func (s Scope) id() string {
	raw, _ := json.Marshal([]string{s.Tenant, s.Account, s.Workspace, s.Namespace})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s Scope) owns(key artifactrelease.Key) bool {
	return key.Valid() && key.Tenant == s.Tenant && key.Account == s.Account && key.Workspace == s.Workspace && key.Namespace == s.Namespace
}

// Store borrows DB. It owns no connection lifetime and never projects a
// Publisher itself; the host completes a pending revision only after the
// corresponding manifest is visible under its release read/write gate.
type Store struct {
	DB    *sql.DB
	scope Scope
	actor func(context.Context) (string, error)
	clock func() time.Time
}

var _ artifactrelease.ChannelStore = (*Store)(nil)

func New(db *sql.DB, scope Scope, verifiedActor string) (*Store, error) {
	if !validActor(verifiedActor) {
		return nil, artifactrelease.ErrInvalid
	}
	return NewWithActorResolver(db, scope, func(context.Context) (string, error) { return verifiedActor, nil })
}

// NewWithActorResolver obtains the verified actor from each request context.
// Serving hosts must use their existing authenticated identity seam rather
// than accepting an actor supplied by browser input or a shared mutable field.
func NewWithActorResolver(db *sql.DB, scope Scope, resolve func(context.Context) (string, error)) (*Store, error) {
	if db == nil || !scope.valid() || resolve == nil {
		return nil, artifactrelease.ErrInvalid
	}
	return &Store{DB: db, scope: scope, actor: resolve, clock: func() time.Time { return time.Now().UTC() }}, nil
}

func validActor(actor string) bool {
	return actor != "" && strings.TrimSpace(actor) == actor && !strings.ContainsRune(actor, '\x00')
}

func (s *Store) requestActor(ctx context.Context) (string, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return "", err
	}
	if !validActor(actor) {
		return "", artifactrelease.ErrInvalid
	}
	return actor, nil
}

func (s *Store) now() time.Time { return s.clock().UTC() }

func (s *Store) PutBundle(ctx context.Context, bundle artifactrelease.Bundle) error {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || bundle.Namespace != s.scope.Namespace || artifactrelease.ValidateBundle(bundle) != nil {
		return artifactrelease.ErrInvalid
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.DB.ExecContext(ctx, `INSERT INTO artifact_release_bundles
 (scope_id,tenant_id,account_id,workspace_id,namespace,bundle_hash,bundle_json,created_at,created_by,updated_at,updated_by)
 VALUES (?,?,?,?,?,?,?,?,?,?,?)`, s.scope.id(), s.scope.Tenant, s.scope.Account, s.scope.Workspace,
		s.scope.Namespace, bundle.Hash, raw, now, actor, now, actor)
	if err == nil {
		return nil
	}
	prior, readErr := s.Bundle(ctx, bundle.Hash)
	if readErr == nil && reflect.DeepEqual(prior, bundle) {
		return nil
	}
	if readErr == nil {
		return artifactrelease.ErrConflict
	}
	return err
}

func (s *Store) Bundle(ctx context.Context, hash string) (artifactrelease.Bundle, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || len(hash) != 64 {
		return artifactrelease.Bundle{}, artifactrelease.ErrInvalid
	}
	var tenant, account, workspace, namespace string
	var raw []byte
	err := s.DB.QueryRowContext(ctx, `SELECT tenant_id,account_id,workspace_id,namespace,bundle_json
 FROM artifact_release_bundles WHERE scope_id=? AND bundle_hash=?`, s.scope.id(), hash).
		Scan(&tenant, &account, &workspace, &namespace, &raw)
	if err != nil {
		return artifactrelease.Bundle{}, err
	}
	if tenant != s.scope.Tenant || account != s.scope.Account || workspace != s.scope.Workspace || namespace != s.scope.Namespace {
		return artifactrelease.Bundle{}, artifactrelease.ErrInvalid
	}
	var bundle artifactrelease.Bundle
	if json.Unmarshal(raw, &bundle) != nil || bundle.Hash != hash || artifactrelease.ValidateBundle(bundle) != nil {
		return artifactrelease.Bundle{}, artifactrelease.ErrInvalid
	}
	return bundle, nil
}

func (s *Store) CreateChannel(ctx context.Context, channel artifactrelease.Channel) error {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(channel.Key) ||
		artifactrelease.ValidateChannel(channel) != nil || channel.Revision != 0 || channel.ActiveBundleHash != "" {
		return artifactrelease.ErrInvalid
	}
	policies, err := json.Marshal(channel.Policies)
	if err != nil {
		return err
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.DB.ExecContext(ctx, `INSERT INTO artifact_release_channels
 (scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,active_bundle_hash,revision,pending_operation_id,created_at,created_by,updated_at,updated_by)
 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, s.scope.id(), s.scope.Tenant, s.scope.Account,
		s.scope.Workspace, s.scope.Namespace, channel.Key.Name, channel.Description, policies,
		nil, 0, nil, now, actor, now, actor)
	if err != nil {
		var existing string
		if readErr := s.DB.QueryRowContext(ctx, `SELECT name FROM artifact_release_channels WHERE scope_id=? AND name=?`, s.scope.id(), channel.Key.Name).Scan(&existing); readErr == nil {
			return artifactrelease.ErrConflict
		}
		return fmt.Errorf("create release channel: %w", err)
	}
	return nil
}

func (s *Store) Channel(ctx context.Context, key artifactrelease.Key) (artifactrelease.Channel, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(key) {
		return artifactrelease.Channel{}, artifactrelease.ErrInvalid
	}
	return s.channel(ctx, s.DB, key)
}

// ListChannels returns only committed channel pointers. Pending revisions are
// exposed separately by Pending and never masquerade as active releases.
func (s *Store) ListChannels(ctx context.Context) ([]artifactrelease.Channel, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil {
		return nil, artifactrelease.ErrInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT name FROM artifact_release_channels WHERE scope_id=? ORDER BY name`, s.scope.id())
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	result := make([]artifactrelease.Channel, 0, len(names))
	for _, name := range names {
		key := artifactrelease.Key{Tenant: s.scope.Tenant, Account: s.scope.Account, Workspace: s.scope.Workspace, Namespace: s.scope.Namespace, Name: name}
		channel, readErr := s.Channel(ctx, key)
		if readErr != nil {
			return nil, readErr
		}
		result = append(result, channel)
	}
	return result, nil
}

// History includes all attempted revisions for audit. Only an applied entry
// selects a bundle eligible for rollback; pending/failed entries never do.
type HistoryEntry struct {
	OperationID string
	Next        artifactrelease.Channel
	Status      string
	CreatedAt   time.Time
	CreatedBy   string
	CompletedAt *time.Time
	FailureCode string
}

func (s *Store) History(ctx context.Context, key artifactrelease.Key) ([]HistoryEntry, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(key) {
		return nil, artifactrelease.ErrInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT operation_id,next_channel_json,status,created_at,created_by,completed_at,failure_code
 FROM artifact_release_operations WHERE scope_id=? AND channel_name=? ORDER BY created_at,operation_id`, s.scope.id(), key.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []HistoryEntry
	for rows.Next() {
		var item HistoryEntry
		var raw []byte
		var completed sql.NullTime
		var code sql.NullString
		if err = rows.Scan(&item.OperationID, &raw, &item.Status, &item.CreatedAt, &item.CreatedBy, &completed, &code); err != nil {
			return nil, err
		}
		if !validOperationID(item.OperationID) || json.Unmarshal(raw, &item.Next) != nil || artifactrelease.ValidateChannel(item.Next) != nil || item.Next.Key != key {
			return nil, artifactrelease.ErrInvalid
		}
		if item.Status != "pending" && item.Status != "applied" && item.Status != "failed" {
			return nil, artifactrelease.ErrInvalid
		}
		if completed.Valid {
			item.CompletedAt = &completed.Time
		}
		if code.Valid {
			item.FailureCode = code.String
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type queryRow interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) channel(ctx context.Context, source queryRow, key artifactrelease.Key) (artifactrelease.Channel, error) {
	var tenant, account, workspace, namespace, description string
	var policies []byte
	var active sql.NullString
	var revision int64
	err := source.QueryRowContext(ctx, `SELECT tenant_id,account_id,workspace_id,namespace,description,policy_refs_json,active_bundle_hash,revision
 FROM artifact_release_channels WHERE scope_id=? AND name=?`, s.scope.id(), key.Name).
		Scan(&tenant, &account, &workspace, &namespace, &description, &policies, &active, &revision)
	if err != nil {
		return artifactrelease.Channel{}, err
	}
	if tenant != s.scope.Tenant || account != s.scope.Account || workspace != s.scope.Workspace || namespace != s.scope.Namespace || revision < 0 {
		return artifactrelease.Channel{}, artifactrelease.ErrInvalid
	}
	channel := artifactrelease.Channel{Key: key, Description: description, Revision: uint64(revision)}
	if active.Valid {
		channel.ActiveBundleHash = active.String
	}
	if json.Unmarshal(policies, &channel.Policies) != nil || artifactrelease.ValidateChannel(channel) != nil {
		return artifactrelease.Channel{}, artifactrelease.ErrInvalid
	}
	return channel, nil
}

// StageRevision atomically reserves the channel and records a durable intent;
// it does not change the committed active bundle or revision.
func (s *Store) StageRevision(ctx context.Context, current, next artifactrelease.Channel, bundle artifactrelease.Bundle) (string, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil {
		return "", artifactrelease.ErrInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	operationID, err := s.StageRevisionTx(ctx, tx, current, next, bundle)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return operationID, nil
}

// StageRevisionTx lets the trusted host commit other release-grant metadata
// with the pending channel intent. The caller owns commit and rollback.
func (s *Store) StageRevisionTx(ctx context.Context, tx *sql.Tx, current, next artifactrelease.Channel, bundle artifactrelease.Bundle) (string, error) {
	if s == nil || tx == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(current.Key) ||
		current.Key != next.Key || artifactrelease.ValidateChannel(current) != nil || artifactrelease.ValidateChannel(next) != nil ||
		artifactrelease.ValidateBundle(bundle) != nil || bundle.Namespace != s.scope.Namespace || next.ActiveBundleHash != bundle.Hash ||
		next.Revision != current.Revision+1 {
		return "", artifactrelease.ErrInvalid
	}
	var storedRaw []byte
	if err := tx.QueryRowContext(ctx, `SELECT bundle_json FROM artifact_release_bundles WHERE scope_id=? AND bundle_hash=?`, s.scope.id(), bundle.Hash).Scan(&storedRaw); err != nil {
		return "", artifactrelease.ErrInvalid
	}
	var storedBundle artifactrelease.Bundle
	if json.Unmarshal(storedRaw, &storedBundle) != nil || artifactrelease.ValidateBundle(storedBundle) != nil || !reflect.DeepEqual(storedBundle, bundle) {
		return "", artifactrelease.ErrInvalid
	}
	stored, err := s.channel(ctx, tx, current.Key)
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(stored, current) {
		return "", artifactrelease.ErrConflict
	}
	operationID, err := randomOperationID()
	if err != nil {
		return "", err
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return "", err
	}
	now := s.now()
	result, err := tx.ExecContext(ctx, `UPDATE artifact_release_channels SET pending_operation_id=?,updated_at=?,updated_by=?
 WHERE scope_id=? AND name=? AND revision=? AND pending_operation_id IS NULL`, operationID, now, actor,
		s.scope.id(), current.Key.Name, int64(current.Revision))
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if changed != 1 {
		return "", artifactrelease.ErrConflict
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO artifact_release_operations
 (operation_id,scope_id,channel_name,expected_revision,target_revision,target_bundle_hash,next_channel_json,status,created_at,created_by,updated_at,updated_by)
 VALUES (?,?,?,?,?,?,?,'pending',?,?,?,?)`, operationID, s.scope.id(), current.Key.Name,
		int64(current.Revision), int64(next.Revision), bundle.Hash, raw, now, actor, now, actor)
	if err != nil {
		return "", err
	}
	return operationID, nil
}

func randomOperationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

// CompleteRevision is idempotent after a successful projection. A pending
// intent remains durable if the transaction fails; restart recovery replays
// the same projection before calling CompleteRevision again.
func (s *Store) CompleteRevision(ctx context.Context, key artifactrelease.Key, expectedCurrentRevision uint64, operationID string) error {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(key) || !validOperationID(operationID) || expectedCurrentRevision >= math.MaxInt64 {
		return artifactrelease.ErrInvalid
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := s.channel(ctx, tx, key)
	if err != nil {
		return err
	}
	if current.Revision == expectedCurrentRevision+1 {
		var applied int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_operations WHERE operation_id=? AND scope_id=? AND channel_name=? AND target_revision=? AND status='applied'`,
			operationID, s.scope.id(), key.Name, int64(current.Revision)).Scan(&applied)
		if err == nil && applied == 1 {
			return nil
		}
		return artifactrelease.ErrConflict
	}
	if current.Revision != expectedCurrentRevision {
		return artifactrelease.ErrConflict
	}
	operation, err := s.pendingForChannel(ctx, tx, key)
	if err != nil {
		return err
	}
	if operation.id != operationID || operation.Current.Revision != expectedCurrentRevision || operation.Next.Revision != expectedCurrentRevision+1 {
		return artifactrelease.ErrConflict
	}
	policies, err := json.Marshal(operation.Next.Policies)
	if err != nil {
		return err
	}
	now := s.now()
	result, err := tx.ExecContext(ctx, `UPDATE artifact_release_channels SET description=?,policy_refs_json=?,active_bundle_hash=?,revision=?,pending_operation_id=NULL,updated_at=?,updated_by=?
 WHERE scope_id=? AND name=? AND revision=? AND pending_operation_id=?`, operation.Next.Description, policies,
		operation.Next.ActiveBundleHash, int64(operation.Next.Revision), now, actor,
		s.scope.id(), key.Name, int64(expectedCurrentRevision), operation.id)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return artifactrelease.ErrConflict
	}
	result, err = tx.ExecContext(ctx, `UPDATE artifact_release_operations SET status='applied',completed_at=?,updated_at=?,updated_by=?
 WHERE operation_id=? AND status='pending'`, now, now, actor, operation.id)
	if err != nil {
		return err
	}
	changed, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return artifactrelease.ErrConflict
	}
	return tx.Commit()
}

// FailRevision retains the previous active bundle when projection failed.
// Only a bounded categorical code is persisted, never payload or credentials.
func (s *Store) FailRevision(ctx context.Context, key artifactrelease.Key, expectedCurrentRevision uint64, operationID, code string) error {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(key) || !validOperationID(operationID) || !validFailureCode(code) || expectedCurrentRevision >= math.MaxInt64 {
		return artifactrelease.ErrInvalid
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	operation, err := s.pendingForChannel(ctx, tx, key)
	if err != nil {
		return err
	}
	if operation.id != operationID || operation.Current.Revision != expectedCurrentRevision {
		return artifactrelease.ErrConflict
	}
	now := s.now()
	result, err := tx.ExecContext(ctx, `UPDATE artifact_release_channels SET pending_operation_id=NULL,updated_at=?,updated_by=?
 WHERE scope_id=? AND name=? AND revision=? AND pending_operation_id=?`, now, actor,
		s.scope.id(), key.Name, int64(expectedCurrentRevision), operation.id)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return artifactrelease.ErrConflict
	}
	result, err = tx.ExecContext(ctx, `UPDATE artifact_release_operations SET status='failed',failure_code=?,completed_at=?,updated_at=?,updated_by=?
 WHERE operation_id=? AND status='pending'`, code, now, now, actor, operation.id)
	if err != nil {
		return err
	}
	changed, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return artifactrelease.ErrConflict
	}
	return tx.Commit()
}

func validFailureCode(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if character == '_' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

func validOperationID(value string) bool {
	if len(value) != 32 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type pendingRow struct {
	artifactrelease.PendingRevision
	id string
}

func (s *Store) pendingForChannel(ctx context.Context, tx *sql.Tx, key artifactrelease.Key) (pendingRow, error) {
	var id, hash, status string
	var expected, target int64
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT o.operation_id,o.expected_revision,o.target_revision,o.target_bundle_hash,o.next_channel_json,o.status
 FROM artifact_release_operations o JOIN artifact_release_channels c ON c.scope_id=o.scope_id AND c.name=o.channel_name
 WHERE o.scope_id=? AND o.channel_name=? AND c.pending_operation_id=o.operation_id`, s.scope.id(), key.Name).
		Scan(&id, &expected, &target, &hash, &raw, &status)
	if err != nil {
		return pendingRow{}, err
	}
	if status != "pending" || !validOperationID(id) || expected < 0 || target != expected+1 {
		return pendingRow{}, artifactrelease.ErrInvalid
	}
	current, err := s.channel(ctx, tx, key)
	if err != nil {
		return pendingRow{}, err
	}
	var next artifactrelease.Channel
	if json.Unmarshal(raw, &next) != nil || artifactrelease.ValidateChannel(next) != nil || next.Key != key ||
		next.Revision != uint64(target) || next.ActiveBundleHash != hash || current.Revision != uint64(expected) {
		return pendingRow{}, artifactrelease.ErrInvalid
	}
	return pendingRow{PendingRevision: artifactrelease.PendingRevision{OperationID: id, Current: current, Next: next, BundleHash: hash}, id: id}, nil
}

// Pending returns only intents still referenced by their channel's pending
// pointer. A restarted host reprojects each target idempotently, then calls
// CompleteRevision; nothing is reported as active before that completion.
func (s *Store) Pending(ctx context.Context) ([]artifactrelease.PendingRevision, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil {
		return nil, artifactrelease.ErrInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT name FROM artifact_release_channels WHERE scope_id=? AND pending_operation_id IS NOT NULL ORDER BY name`, s.scope.id())
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	result := make([]artifactrelease.PendingRevision, 0, len(names))
	for _, name := range names {
		key := artifactrelease.Key{Tenant: s.scope.Tenant, Account: s.scope.Account, Workspace: s.scope.Workspace, Namespace: s.scope.Namespace, Name: name}
		tx, beginErr := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if beginErr != nil {
			return nil, beginErr
		}
		item, readErr := s.pendingForChannel(ctx, tx, key)
		closeErr := tx.Rollback()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil && !errors.Is(closeErr, sql.ErrTxDone) {
			return nil, closeErr
		}
		result = append(result, item.PendingRevision)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Current.Key.Name < result[j].Current.Key.Name })
	return result, nil
}
