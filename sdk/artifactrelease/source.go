package artifactrelease

import (
	"context"

	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// SourceEntry is producer-owned inventory metadata. It is not an authority
// supplied by a client; the release controller still authorizes the operation.
type SourceEntry struct {
	Root  artifactcohort.RootRef
	Title string
}

// Source adapts a typed immutable store to the common release pipeline.
// Storage selection stays with the producer; encoding and optional custom
// validation reuse ArtifactBuilder rather than implementing publication again.
type Source[T any] struct {
	// Select identifies this trusted source's URI space. It must not depend on
	// caller-provided permission claims or perform discovery.
	Select func(artifactcohort.RootRef) bool
	// Check optionally performs a cheap source-availability check for background
	// release reconciliation. Without it, the immutable source is read normally.
	Check   func(context.Context, artifactcohort.RootRef) error
	List    func(context.Context) ([]SourceEntry, error)
	Read    func(context.Context, artifactcohort.RootRef) (T, error)
	Builder artifactcohort.ArtifactBuilder[T]
}

func (s Source[T]) Artifact(ctx context.Context, ref artifactcohort.RootRef) (*artifactcohort.Artifact, error) {
	if ctx == nil || s.Read == nil {
		return nil, contract.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, err := s.Read(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.Builder.Build(ctx, ref, value)
}

func (s Source[T]) Candidates(ctx context.Context) ([]contract.Candidate, error) {
	if ctx == nil || s.List == nil {
		return nil, contract.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]contract.Candidate, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		if seen[entry.Root.URI] {
			return nil, contract.ErrConflict
		}
		seen[entry.Root.URI] = true
		value, err := s.Artifact(ctx, entry.Root)
		if err != nil {
			return nil, err
		}
		result = append(result, contract.Candidate{Root: value.Root, Title: entry.Title})
	}
	return result, nil
}

func (s Source[T]) Owns(ref artifactcohort.RootRef) bool { return s.Select != nil && s.Select(ref) }
func (s Source[T]) Available(ctx context.Context, ref artifactcohort.RootRef) error {
	if ctx == nil {
		return contract.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Check != nil {
		return s.Check(ctx, ref)
	}
	_, err := s.Artifact(ctx, ref)
	return err
}
