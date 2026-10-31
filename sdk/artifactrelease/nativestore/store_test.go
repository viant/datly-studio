package nativestore

import (
	"context"
	"errors"
	contract "github.com/viant/forge/backend/artifactrelease"
	"testing"
)

func TestNilStoreRejectsBeforeScopeAccess(t *testing.T) {
	var store *Store
	if _, err := store.BundleBindings(context.Background(), contract.Bundle{}); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("nil store binding read: %v", err)
	}
	if _, err := store.Pending(context.Background()); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("nil store pending recovery: %v", err)
	}
}
