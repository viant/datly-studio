package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// ComponentBinding is producer-compiled companion metadata for an immutable
// release bundle. Public release DTOs never accept it. Both Studio API and
// native runtime read the same exact component revision and fingerprints.
type ComponentBinding struct {
	Root               artifactcohort.RootRef `json:"root"`
	DataSourceID       string                 `json:"dataSourceId"`
	ComponentKind      string                 `json:"componentKind"`
	ComponentID        string                 `json:"componentId"`
	ComponentRevision  string                 `json:"componentRevision"`
	Method             string                 `json:"method"`
	Route              string                 `json:"route"`
	SchemaFingerprint  string                 `json:"schemaFingerprint"`
	ContentFingerprint string                 `json:"contentFingerprint"`
}

func validateBindings(bundle contract.Bundle, bindings []ComponentBinding) ([]ComponentBinding, []byte, string, error) {
	if contract.ValidateBundle(bundle) != nil {
		return nil, nil, "", contract.ErrInvalid
	}
	allowed := map[string]artifactcohort.RootRef{}
	for _, root := range bundle.Roots {
		allowed[artifactcohort.RootKey(root)] = root
	}
	copy := append([]ComponentBinding(nil), bindings...)
	sort.Slice(copy, func(i, j int) bool { return bindingKey(copy[i]) < bindingKey(copy[j]) })
	prior := ""
	for _, binding := range copy {
		key := bindingKey(binding)
		if key <= prior || allowed[artifactcohort.RootKey(binding.Root)] != binding.Root || binding.Root.Kind != "window" || binding.DataSourceID == "" || strings.TrimSpace(binding.DataSourceID) != binding.DataSourceID || binding.ComponentID == "" || strings.TrimSpace(binding.ComponentID) != binding.ComponentID || binding.ComponentRevision == "" || strings.TrimSpace(binding.ComponentRevision) != binding.ComponentRevision || (binding.ComponentKind != "dynamic" && binding.ComponentKind != "linked") || (binding.Method != http.MethodGet && binding.Method != http.MethodPost) || !strings.HasPrefix(binding.Route, "/") || strings.HasPrefix(binding.Route, "//") || strings.ContainsAny(binding.Route, "?#") || !validHash(binding.SchemaFingerprint) || !validHash(binding.ContentFingerprint) {
			return nil, nil, "", contract.ErrInvalid
		}
		prior = key
	}
	if copy == nil {
		copy = []ComponentBinding{}
	}
	raw, err := json.Marshal(copy)
	if err != nil {
		return nil, nil, "", err
	}
	sum := sha256.Sum256(raw)
	return copy, raw, hex.EncodeToString(sum[:]), nil
}

func bindingKey(value ComponentBinding) string {
	return artifactcohort.RootKey(value.Root) + "\x00" + value.DataSourceID + "\x00" + value.ComponentKind + "\x00" + value.ComponentID + "\x00" + value.ComponentRevision
}
func validHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// StageRevisionWithBindings writes immutable producer closure metadata and the
// pending operation in one caller-independent DB transaction. Its commitment
// does not activate the channel; Publisher projection and Complete follow.
func (s *Store) StageRevisionWithBindings(ctx context.Context, current, next contract.Channel, bundle contract.Bundle, bindings []ComponentBinding) (string, error) {
	return s.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, bindings, nil)
}

func (s *Store) StageRevisionWithBindingsAndDependencies(ctx context.Context, current, next contract.Channel, bundle contract.Bundle, bindings []ComponentBinding, dependencies []authz.NamedDependency) (string, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil {
		return "", contract.ErrInvalid
	}
	_, raw, hash, err := validateBindings(bundle, bindings)
	if err != nil {
		return "", err
	}
	depsRaw, err := marshalDependencies(dependencies)
	if err != nil {
		return "", err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err = s.putBindingsTx(ctx, tx, bundle, raw, hash); err != nil {
		return "", err
	}
	operationID, err := s.stageRevisionTx(ctx, tx, current, next, bundle, depsRaw)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return operationID, nil
}

func marshalDependencies(input []authz.NamedDependency) ([]byte, error) {
	copy := append([]authz.NamedDependency(nil), input...)
	sort.Slice(copy, func(i, j int) bool { return dependencyKey(copy[i]) < dependencyKey(copy[j]) })
	prior := ""
	for _, item := range copy {
		key := dependencyKey(item)
		if item.Revision < 1 || item.Resource.Tenant == "" || item.Resource.Kind == "" || item.Resource.ID == "" || item.Resource.Version == "" || key <= prior {
			return nil, contract.ErrInvalid
		}
		prior = key
	}
	if copy == nil {
		copy = []authz.NamedDependency{}
	}
	return json.Marshal(copy)
}
func dependencyKey(value authz.NamedDependency) string {
	return value.Resource.Tenant + "\x00" + value.Resource.Kind + "\x00" + value.Resource.ID + "\x00" + value.Resource.Version
}

func (s *Store) ActiveDependencies(ctx context.Context, key contract.Key) ([]authz.NamedDependency, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.owns(key) {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT active_dependencies_json FROM artifact_release_channels WHERE scope_id=? AND name=?`, s.scope.id(), key.Name).Scan(&raw); err != nil {
		return nil, err
	}
	return decodeDependencies(raw)
}
func (s *Store) PendingDependencies(ctx context.Context, operationID string) ([]authz.NamedDependency, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !validOperationID(operationID) {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT dependencies_json FROM artifact_release_operations WHERE scope_id=? AND operation_id=? AND status='pending'`, s.scope.id(), operationID).Scan(&raw); err != nil {
		return nil, err
	}
	return decodeDependencies(raw)
}
func decodeDependencies(raw []byte) ([]authz.NamedDependency, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value []authz.NamedDependency
	if json.Unmarshal(raw, &value) != nil {
		return nil, contract.ErrInvalid
	}
	canonical, err := marshalDependencies(value)
	if err != nil || string(canonical) != string(raw) {
		return nil, contract.ErrInvalid
	}
	return value, nil
}

func (s *Store) putBindingsTx(ctx context.Context, tx *sql.Tx, bundle contract.Bundle, raw []byte, hash string) error {
	var priorRaw []byte
	var priorHash string
	err := tx.QueryRowContext(ctx, `SELECT bindings_json,bindings_hash FROM artifact_release_bundle_bindings WHERE scope_id=? AND bundle_hash=?`, s.scope.id(), bundle.Hash).Scan(&priorRaw, &priorHash)
	if err == nil {
		if priorHash == hash && string(priorRaw) == string(raw) {
			return nil
		}
		return contract.ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	actor, err := s.requestActor(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = tx.ExecContext(ctx, `INSERT INTO artifact_release_bundle_bindings(scope_id,bundle_hash,bindings_json,bindings_hash,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,?,?)`, s.scope.id(), bundle.Hash, raw, hash, now, actor, now, actor)
	return err
}

func (s *Store) BundleBindings(ctx context.Context, bundle contract.Bundle) ([]ComponentBinding, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || bundle.Namespace != s.scope.Namespace || contract.ValidateBundle(bundle) != nil {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	var hash string
	if err := s.DB.QueryRowContext(ctx, `SELECT bindings_json,bindings_hash FROM artifact_release_bundle_bindings WHERE scope_id=? AND bundle_hash=?`, s.scope.id(), bundle.Hash).Scan(&raw, &hash); err != nil {
		return nil, err
	}
	var bindings []ComponentBinding
	if json.Unmarshal(raw, &bindings) != nil {
		return nil, contract.ErrInvalid
	}
	validated, canonical, expected, err := validateBindings(bundle, bindings)
	if err != nil || hash != expected || string(canonical) != string(raw) {
		return nil, contract.ErrInvalid
	}
	return validated, nil
}
