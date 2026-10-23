package host

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/types"
)

func TestPublishedRootLedgerRetainsExactRevisionsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "management-artifacts.json")
	uri := "window://datastudio/connectors"
	validator := func(_ context.Context, ref artifactcohort.RootRef, body json.RawMessage) error {
		if ref.Kind != "window" || ref.URI != uri {
			return artifactcohort.ErrInvalid
		}
		var envelope types.WindowResourceEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			return err
		}
		return envelope.Validate()
	}
	candidate := func(id string) []PublishedRootPayload {
		return []PublishedRootPayload{{URI: uri, Kind: "window", Payload: closedWindowForCohort(t, id)}}
	}
	firstCatalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	first, err := StagePublishedRoots(ctx, firstCatalog, path, candidate("first"), validator)
	if err != nil || first[uri].Version != 1 {
		t.Fatalf("initial publication: %v %+v", err, first)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secondCatalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	second, err := StagePublishedRoots(ctx, secondCatalog, path, candidate("second"), validator)
	if err != nil || second[uri].Version != 2 || second[uri].Hash == first[uri].Hash {
		t.Fatalf("changed publication did not advance: %v %+v", err, second)
	}
	for _, ref := range []artifactcohort.RootRef{first[uri], second[uri]} {
		artifact, err := secondCatalog.Publisher.Get(ref)
		if err != nil || artifact.Root != ref || artifactcohort.ValidateArtifact(artifact) != nil {
			t.Fatalf("historical revision was not restored exactly: %v", err)
		}
	}
	thirdCatalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	third, err := StagePublishedRoots(ctx, thirdCatalog, path, candidate("second"), validator)
	if err != nil || third[uri] != second[uri] {
		t.Fatalf("same bytes reused different version: %v", err)
	}
	stable, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, stable) || len(stable) <= len(before) {
		t.Fatal("changed candidate did not append immutable revision")
	}
	invalidCatalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	if _, err := StagePublishedRoots(ctx, invalidCatalog, path, []PublishedRootPayload{{URI: uri, Kind: "window", Payload: json.RawMessage(`{}`)}}, validator); err == nil {
		t.Fatal("invalid new candidate overwrote valid history")
	}
	afterFailure, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(afterFailure, stable) {
		t.Fatal("failed candidate changed active ledger")
	}
}
