package host

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/viant/forge/backend/artifactcohort"
	primitive "github.com/viant/forge/backend/provider/contract"
)

func TestNamespaceArtifactSyncConditionalAndHostAdmission(t *testing.T) {
	ctx := context.Background()
	catalog := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	manifest, err := catalog.ReplaceCatalog(ctx, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	admissions := 0
	s := &Service{config: Config{ArtifactCohorts: catalog, AuthorizeArtifactHost: func(context.Context) error { admissions++; return nil }}}
	request := primitive.NamespaceGetRequest{Namespace: "example", Sync: &primitive.NamespaceSyncRequest{}}
	result, err := s.synchronizeNamespaceArtifacts(ctx, request)
	if err != nil || result.Sync == nil || admissions != 1 {
		t.Fatalf("namespace sync: %v", err)
	}
	var full artifactcohort.Manifest
	if json.Unmarshal(result.Sync.Catalog, &full) != nil || full.Hash != manifest.Hash {
		t.Fatal("catalog mismatch")
	}
	request.Sync.KnownHash = manifest.Hash
	result, err = s.synchronizeNamespaceArtifacts(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var compact map[string]any
	if json.Unmarshal(result.Sync.Catalog, &compact) != nil || compact["unchanged"] != true || compact["hash"] != manifest.Hash || len(compact) != 2 {
		t.Fatal("conditional index not compact")
	}
	s.config.AuthorizeArtifactHost = func(context.Context) error { return errors.New("not a host") }
	if _, err = s.synchronizeNamespaceArtifacts(ctx, request); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("host boundary: %v", err)
	}
}
