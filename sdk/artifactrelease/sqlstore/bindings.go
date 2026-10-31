package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"reflect"

	"github.com/viant/authz"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// ComponentBinding is producer-compiled companion metadata for an immutable
// release bundle. Public release DTOs never accept it. Both Studio API and
// native runtime read the same exact component revision and fingerprints.
type ComponentBinding = releasevalue.ComponentBinding

func validateBindings(bundle contract.Bundle, bindings []ComponentBinding) ([]ComponentBinding, []byte, string, error) {
	return releasevalue.ValidateBindings(bundle, bindings)
}

func bindingKey(value ComponentBinding) string { return releasevalue.BindingKey(value) }

func validHash(value string) bool { return releasevalue.ValidHash(value) }

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
	return releasevalue.MarshalDependencies(input)
}

func dependencyKey(value authz.NamedDependency) string { return releasevalue.DependencyKey(value) }

func (s *Store) ActiveDependencies(ctx context.Context, key contract.Key) ([]authz.NamedDependency, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !s.scope.Owns(key) {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT active_dependencies_json FROM artifact_release_channels WHERE scope_id=? AND name=?`, s.scope.ID(), key.Name).Scan(&raw); err != nil {
		return nil, err
	}
	return decodeDependencies(raw)
}
func (s *Store) PendingDependencies(ctx context.Context, operationID string) ([]authz.NamedDependency, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || !validOperationID(operationID) {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT dependencies_json FROM artifact_release_operations WHERE scope_id=? AND operation_id=? AND status='pending'`, s.scope.ID(), operationID).Scan(&raw); err != nil {
		return nil, err
	}
	return decodeDependencies(raw)
}
func decodeDependencies(raw []byte) ([]authz.NamedDependency, error) {
	return releasevalue.DecodeDependencies(raw)
}

func strictJSON(raw []byte, target any) error { return releasevalue.StrictJSON(raw, target) }

func (s *Store) putBindingsTx(ctx context.Context, tx *sql.Tx, bundle contract.Bundle, raw []byte, hash string) error {
	var priorRaw []byte
	var priorHash string
	err := tx.QueryRowContext(ctx, `SELECT bindings_json,bindings_hash FROM artifact_release_bundle_bindings WHERE scope_id=? AND bundle_hash=?`, s.scope.ID(), bundle.Hash).Scan(&priorRaw, &priorHash)
	if err == nil {
		prior, readErr := decodeBindings(bundle, priorRaw, priorHash)
		var incoming []ComponentBinding
		if readErr == nil && json.Unmarshal(raw, &incoming) == nil && reflect.DeepEqual(prior, incoming) && priorHash == hash {
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
	_, err = tx.ExecContext(ctx, `INSERT INTO artifact_release_bundle_bindings(scope_id,bundle_hash,bindings_json,bindings_hash,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,?,?)`, s.scope.ID(), bundle.Hash, raw, hash, now, actor, now, actor)
	return err
}

func (s *Store) BundleBindings(ctx context.Context, bundle contract.Bundle) ([]ComponentBinding, error) {
	if s == nil || s.DB == nil || ctx == nil || ctx.Err() != nil || bundle.Namespace != s.scope.Namespace || contract.ValidateBundle(bundle) != nil {
		return nil, contract.ErrInvalid
	}
	var raw []byte
	var hash string
	if err := s.DB.QueryRowContext(ctx, `SELECT bindings_json,bindings_hash FROM artifact_release_bundle_bindings WHERE scope_id=? AND bundle_hash=?`, s.scope.ID(), bundle.Hash).Scan(&raw, &hash); err != nil {
		return nil, err
	}
	return decodeBindings(bundle, raw, hash)
}

func decodeBindings(bundle contract.Bundle, raw []byte, hash string) ([]ComponentBinding, error) {
	return releasevalue.DecodeBindings(bundle, raw, hash)
}
