package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/viant/forge/backend/artifactcohort"
)

// PublishedRootPayload is producer-closed metadata ready for immutable
// publication. Version assignment belongs to the producer ledger, never a
// client-supplied MCP request.
type PublishedRootPayload struct {
	URI, Kind string
	Payload   json.RawMessage
}

type publishedLedgerRecord struct {
	Root    artifactcohort.RootRef `json:"root"`
	Payload json.RawMessage        `json:"payload"`
}
type publishedLedgerFile struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Records       []publishedLedgerRecord `json:"records"`
}

// StagePublishedRoots replays exact historical bytes, validates every new
// candidate, and assigns a monotonic version to changed management metadata.
// The ledger must be on durable storage owned by this one producer process.
// Call before the MCP listener serves the catalog; any failure aborts startup.
func StagePublishedRoots(ctx context.Context, catalog *PublishedArtifactCatalog, path string, candidates []PublishedRootPayload, validate func(context.Context, artifactcohort.RootRef, json.RawMessage) error) (map[string]artifactcohort.RootRef, error) {
	if ctx == nil || ctx.Err() != nil || catalog == nil || catalog.Publisher == nil || !filepath.IsAbs(path) || validate == nil {
		return nil, artifactcohort.ErrInvalid
	}
	ledger := publishedLedgerFile{SchemaVersion: 1}
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, &ledger) != nil || ledger.SchemaVersion != 1 {
			return nil, artifactcohort.ErrInvalid
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	latest := map[string]artifactcohort.RootRef{}
	for _, record := range ledger.Records {
		if artifactcohort.ValidateArtifact(&artifactcohort.Artifact{Root: record.Root, Payload: record.Payload}) != nil || record.Root.Version <= latest[record.Root.URI].Version {
			return nil, artifactcohort.ErrInvalid
		}
		if err := validate(ctx, record.Root, record.Payload); err != nil {
			return nil, err
		}
		artifact, _, err := catalog.Publisher.Publish(ctx, record.Root, record.Payload, func(_ context.Context, _ json.RawMessage) error { return nil })
		if err != nil || artifact.Root != record.Root || !bytes.Equal(artifact.Payload, record.Payload) {
			return nil, artifactcohort.ErrInvalid
		}
		latest[record.Root.URI] = artifact.Root
	}
	seen := map[string]bool{}
	candidates = append([]PublishedRootPayload(nil), candidates...)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].URI < candidates[j].URI })
	changed := false
	for _, candidate := range candidates {
		if seen[candidate.URI] || candidate.URI == "" || !artifactcohort.RootKind(candidate.Kind) || len(candidate.Payload) == 0 {
			return nil, artifactcohort.ErrInvalid
		}
		seen[candidate.URI] = true
		canonical, err := json.Marshal(json.RawMessage(candidate.Payload))
		if err != nil {
			return nil, err
		}
		prior := latest[candidate.URI]
		if prior.URI != "" && prior.Kind != candidate.Kind {
			return nil, artifactcohort.ErrConflict
		}
		if prior.URI != "" && prior.Hash == artifactcohort.Hash(canonical) {
			continue
		}
		version := prior.Version + 1
		if version == 0 {
			return nil, artifactcohort.ErrConflict
		}
		ref := artifactcohort.RootRef{URI: candidate.URI, Kind: candidate.Kind, Version: version}
		if err := validate(ctx, ref, canonical); err != nil {
			return nil, err
		}
		artifact, _, err := catalog.Publisher.Publish(ctx, ref, canonical, func(context.Context, json.RawMessage) error { return nil })
		if err != nil {
			return nil, err
		}
		latest[candidate.URI] = artifact.Root
		ledger.Records = append(ledger.Records, publishedLedgerRecord{Root: artifact.Root, Payload: artifact.Payload})
		changed = true
	}
	if changed || len(ledger.Records) == 0 {
		if err := writePublishedLedger(path, ledger); err != nil {
			return nil, err
		}
	}
	return latest, nil
}

func writePublishedLedger(path string, ledger publishedLedgerFile) error {
	raw, err := json.Marshal(ledger)
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".published-artifact-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(raw)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("activate published artifact ledger: %w", err)
	}
	return nil
}
