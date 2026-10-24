package host

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

// ReleaseComponentBinding is compiled from one producer-validated immutable
// window closure. It records the exact native component revision, not an alias
// inferred from a public window URI or a browser request.
type ReleaseComponentBinding struct {
	Root         artifactcohort.RootRef
	DataSourceID string
	Reference    ComponentReference
	Binding      windowprotocol.ComponentBinding
}

// CompileReleaseWindowBindings reuses deployment-owned datasource→component
// routes, then verifies each pin against the native exact-revision resolver.
// Missing or target-varying undeclared dependencies fail publication.
func CompileReleaseWindowBindings(ctx context.Context, artifact *artifactcohort.Artifact, configured map[string]ComponentReference, resolve func(context.Context, ComponentReference) (windowprotocol.ComponentBinding, error)) ([]ReleaseComponentBinding, error) {
	if ctx == nil || ctx.Err() != nil || artifact == nil || artifact.Root.Kind != "window" || artifactcohort.ValidateArtifact(artifact) != nil || resolve == nil {
		return nil, artifactcohort.ErrInvalid
	}
	var envelope types.WindowResourceEnvelope
	if json.Unmarshal(artifact.Payload, &envelope) != nil || envelope.SchemaVersion != 2 || envelope.Format != types.WindowBundleFormat || envelope.Validate() != nil {
		return nil, artifactcohort.ErrInvalid
	}
	if len(configured) == 0 {
		return nil, artifactcohort.ErrInvalid
	}
	seen := map[string]ReleaseComponentBinding{}
	for _, variant := range envelope.Variants {
		if len(variant.DataSources) != len(configured) {
			return nil, artifactcohort.ErrInvalid
		}
		for id, raw := range variant.DataSources {
			base, exists := configured[id]
			if !exists || validateComponentReference(base) != nil {
				return nil, artifactcohort.ErrInvalid
			}
			var descriptor windowprotocol.DataSource
			if json.Unmarshal(raw, &descriptor) != nil || descriptor.ID != id || descriptor.Backend == nil || descriptor.Backend.Kind != "datly" || descriptor.Backend.Ownership != "provider" || descriptor.Backend.Service != "" || descriptor.Backend.Component == nil {
				return nil, artifactcohort.ErrInvalid
			}
			if descriptor.Backend.Method != "windows/datasource" && descriptor.Backend.Method != "datasources/fetch" {
				return nil, artifactcohort.ErrInvalid
			}
			pin := *descriptor.Backend.Component
			if validateCompleteComponentBinding(pin) != nil || pin.Kind != base.Kind || pin.ID != base.ID {
				return nil, artifactcohort.ErrInvalid
			}
			ref := base
			ref.Revision = pin.Revision
			if validateComponentReference(ref) != nil {
				return nil, artifactcohort.ErrInvalid
			}
			observed, err := resolve(ctx, ref)
			if err != nil || windowprotocol.ValidateComponentDispatch(&pin, observed) != nil {
				return nil, artifactcohort.ErrDenied
			}
			key := id + "\x00" + pin.Kind + "\x00" + pin.ID + "\x00" + pin.Revision
			seen[key] = ReleaseComponentBinding{Root: artifact.Root, DataSourceID: id, Reference: ref, Binding: pin}
		}
	}
	result := make([]ReleaseComponentBinding, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		return fmt.Sprint(left.DataSourceID, left.Binding.Kind, left.Binding.ID, left.Binding.Revision) < fmt.Sprint(right.DataSourceID, right.Binding.Kind, right.Binding.ID, right.Binding.Revision)
	})
	return result, nil
}
