package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/viant/datly-studio/internal/compiledartifact"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/executionguard"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/sqlx/io/errx"
	xhandler "github.com/viant/xdatly/handler"
)

// previewContract shares capture/identity between full-reader execution and input
// inspection. Connections are request-owned; artifacts contain no live services.
func (d Dynamic) previewContract(ctx context.Context, definition *definition, sources *sources, dql string, refresh, fullReader bool) (*transcribe.RuntimeContract, error) {
	live := func() (*transcribe.RuntimeContract, error) {
		return d.runtimeContracts(ctx, &transcribe.Source{Types: definition.Types, Scope: definition.Scope, Name: definition.Name, Text: dql, Connector: definition.Connector, Resources: definition.Resources, ColumnRefiner: column.New(sources.Connections)})
	}
	// Directory-based compilation can consume mutable source files that a path
	// alone cannot identify. Transient/selected-view contracts remain live too.
	if !fullReader || d.StudioDB == nil || strings.TrimSpace(d.ModulePath) == "" || definition.ResourceHash == "" {
		return live()
	}
	captured := make(map[string]compiledartifact.Connector, len(sources.captured))
	for name, c := range sources.captured {
		captured[name] = compiledartifact.Connector{Driver: normalizeDriver(c.Driver), DSN: c.DSN, SecretRef: c.SecretRef, Options: c.Options}
	}
	id, owner, err := compiledartifact.Identity(definition.NamespaceID, d.ModulePath, d.RootDir, compiledartifact.Source{Preview: true, ReportID: definition.ReportID, VersionNo: definition.VersionNo, DQL: dql, Scope: definition.Scope, Name: definition.Name, Connector: definition.Connector, Driver: definition.Driver}, definition.ResourceHash, d.Const.Identity(), captured, definition.Types)
	if err != nil {
		return live()
	} // Native portable authority is unavailable; do not invent it.
	if err = executionguard.Check(ctx); err != nil {
		return nil, err
	}
	store, err := compiledartifact.NewStore(d.StudioDB)
	if err != nil {
		return nil, previewArtifactError(err)
	}
	defer store.Close(context.WithoutCancel(ctx))
	previous, err := store.Get(ctx, definition.ReportID, definition.VersionNo, id)
	absent := errors.Is(err, sql.ErrNoRows)
	if err != nil && !absent {
		return nil, previewArtifactError(err)
	}
	if !absent {
		var frozen transcribe.RuntimeContractArtifact
		if err = json.Unmarshal(previous, &frozen); err != nil {
			return nil, previewArtifactError(err)
		}
		materialized, err := frozen.Materialize(definition.Types, definition.Resources)
		if err != nil {
			return nil, previewArtifactError(err)
		}
		if !refresh {
			return materialized, nil
		}
	}
	contract, err := live()
	if err != nil {
		return nil, err
	}
	if err = predicatepolicy.ValidateInternalComponents(contract.Component, definition.Policy); err != nil {
		return nil, err
	}
	frozen, err := contract.Freeze()
	if err != nil {
		if absent {
			return contract, nil
		}
		return nil, previewArtifactError(err)
	}
	payload, err := json.Marshal(frozen)
	if err != nil {
		return nil, previewArtifactError(err)
	}
	if err = executionguard.Check(ctx); err != nil {
		return nil, err
	}
	// This compares the captured verified baseline, without asserting that it is
	// still current. A concurrent refresh's different result is left untouched.
	if !absent && bytes.Equal(previous, payload) {
		return contract, nil
	}
	var expected *string
	if !absent {
		hash := sha256.Sum256(previous)
		value := hex.EncodeToString(hash[:])
		expected = &value
	}
	if err = store.CompareAndSwap(ctx, owner, definition.ReportID, definition.VersionNo, id, payload, expected); err != nil {
		// Match publication's existing storage-capability boundary: an ordinary
		// miss must not make a previously executable reader depend on fitting
		// the private resource store. Explicit refresh keeps persistence strict.
		if absent && !refresh && errors.Is(err, compiledartifact.ErrTooLarge) {
			return contract, nil
		}
		return nil, previewArtifactError(err)
	}
	return contract, nil
}

func previewArtifactError(err error) error {
	var conflict *xhandler.Conflict
	if errors.Is(err, compiledartifact.ErrConflict) || errors.As(err, &conflict) || errx.IsDuplicateKey(err) {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "compiled preview artifact changed; retry preview", Cause: err}
	}
	var public *sdk.Error
	if errors.As(err, &public) {
		return err
	}
	return &sdk.Error{Code: sdk.ErrorInternal, Message: "compiled preview artifact is unavailable", Cause: err}
}
