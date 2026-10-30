package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	identity "github.com/viant/forge/backend/resource"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

// NativeComponentBinding adds an explicit family proof to the unchanged primary
// component pin. A legacy ComponentBinding does not grant sibling capabilities.
// This native-only value is not used by window bindings or JSON dispatch.
type NativeComponentBinding struct {
	windowprotocol.ComponentBinding
	FamilyFingerprint string `json:"familyFingerprint"`
}

// ResolveNativeComponentBinding requires the same describe authority as ordinary
// component resolution, and captures the single-source native family explicitly.
// It is an in-process embedding API; no transport endpoint is added.
func (s *Service) ResolveNativeComponentBinding(ctx context.Context, ref ComponentReference) (result NativeComponentBinding, resultErr error) {
	metadata := &exactComponentMetadata{}
	manager, current, err := s.componentRuntimeInspected(ctx, ref, true, metadata)
	if err != nil {
		return result, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, shutdownNativeMetadata(ctx, manager.Shutdown))
		if resultErr != nil {
			result = NativeComponentBinding{}
		}
	}()
	return nativeComponentBinding(ref, current, metadata)
}

func nativeComponentBinding(ref ComponentReference, primary windowprotocol.ComponentBinding, metadata *exactComponentMetadata) (NativeComponentBinding, error) {
	family, err := exactFamilyFingerprint(metadata.family)
	if err != nil {
		return NativeComponentBinding{}, err
	}
	raw, err := json.Marshal(struct {
		Contract  string
		Reference ComponentReference
		Primary   windowprotocol.ComponentBinding
		Family    string
	}{"native-component-family/v1", ref, primary, family})
	if err != nil {
		return NativeComponentBinding{}, err
	}
	return NativeComponentBinding{ComponentBinding: primary, FamilyFingerprint: identity.ContentFingerprint(raw)}, nil
}

func validateNativeComponentBinding(ref ComponentReference, pinned NativeComponentBinding, primary windowprotocol.ComponentBinding, metadata *exactComponentMetadata) error {
	if pinned.Kind == "" || pinned.ContentFingerprint == "" || pinned.SchemaFingerprint == "" || !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: pinned.FamilyFingerprint}).Valid() {
		return fmt.Errorf("complete native component family pin is required")
	}
	if err := windowprotocol.ValidateComponentDispatch(&pinned.ComponentBinding, primary); err != nil {
		return err
	}
	current, err := nativeComponentBinding(ref, primary, metadata)
	if err != nil {
		return err
	}
	if current.FamilyFingerprint != pinned.FamilyFingerprint {
		return fmt.Errorf("native component family binding changed")
	}
	return nil
}
