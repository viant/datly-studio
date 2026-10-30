package artifactrelease

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/forge/backend/artifactcohort"
)

func TestTypedSourceUsesOneReleasePathForDifferentKinds(t *testing.T) {
	for _, kind := range []string{"skill", "playbook", "prompt", "eval", "agent", "toolfeed"} {
		t.Run(kind, func(t *testing.T) {
			ref := artifactcohort.RootRef{URI: kind + "://example/guide", Kind: kind, Version: 3}
			reads := 0
			source := Source[map[string]any]{List: func(context.Context) ([]SourceEntry, error) { return []SourceEntry{{Root: ref, Title: "Guide"}}, nil }, Read: func(_ context.Context, got artifactcohort.RootRef) (map[string]any, error) {
				reads++
				if got.Version != 3 {
					return nil, errors.New("wrong version")
				}
				return map[string]any{"schemaVersion": 1, "example": "value"}, nil
			}}
			candidates, err := source.Candidates(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(candidates) != 1 || reads != 1 || candidates[0].Title != "Guide" {
				t.Fatal("incorrect inventory")
			}
			restored, err := source.Artifact(context.Background(), candidates[0].Root)
			if err != nil {
				t.Fatal(err)
			}
			if restored.Root != candidates[0].Root {
				t.Fatal("restoration changed identity")
			}
			source.Read = func(context.Context, artifactcohort.RootRef) (map[string]any, error) {
				return nil, artifactcohort.ErrDenied
			}
			if _, err = source.Artifact(context.Background(), ref); !errors.Is(err, artifactcohort.ErrDenied) {
				t.Fatalf("source denial lost: %v", err)
			}
		})
	}
}

func TestSourceAvailabilityUsesOptionalHookWithoutLoadingPayload(t *testing.T) {
	ref := artifactcohort.RootRef{URI: "skill://example/guide", Kind: "skill", Version: 1}
	checks, reads := 0, 0
	source := Source[map[string]any]{
		Select: func(root artifactcohort.RootRef) bool { return root.URI == ref.URI },
		Check:  func(context.Context, artifactcohort.RootRef) error { checks++; return artifactcohort.ErrDenied },
		Read: func(context.Context, artifactcohort.RootRef) (map[string]any, error) {
			reads++
			return map[string]any{"text": "Example"}, nil
		},
	}
	if !source.Owns(ref) || source.Owns(artifactcohort.RootRef{URI: "skill://other/guide"}) {
		t.Fatal("source ownership mismatch")
	}
	if err := source.Available(context.Background(), ref); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("availability denial lost: %v", err)
	}
	if checks != 1 || reads != 0 {
		t.Fatal("availability loaded the payload")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := source.Available(ctx, ref); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if checks != 1 {
		t.Fatal("canceled availability invoked hook")
	}
	source.Check = nil
	if err := source.Available(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if reads != 1 {
		t.Fatal("default availability did not use existing artifact loader")
	}
}
