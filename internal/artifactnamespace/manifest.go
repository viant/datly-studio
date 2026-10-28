package artifactnamespace

import (
	"context"
	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	identity "github.com/viant/forge/backend/resource"
)

// ValidateManifestScope refuses unscoped roots and unrelated policy documents;
// filtering would change the producer catalog hash and its authority meaning.
func ValidateManifestScope(ctx context.Context, manifest artifactcohort.Manifest, namespace string) error {
	if namespace == "" {
		return nil
	}
	store, err := authz.NewStaticStore(manifest.Policies)
	if err != nil {
		return artifactcohort.ErrInvalid
	}
	used := map[authz.Resource]bool{}
	for _, cohort := range manifest.Cohorts {
		for _, root := range append(append([]artifactcohort.RootRef(nil), cohort.Roots...), cohort.RevisionOverrides...) {
			if !artifactRootInNamespace(root, namespace) {
				return artifactcohort.ErrDenied
			}
		}
		for _, rawURI := range append(append([]string(nil), cohort.FollowPublished...), cohort.DeniedURIs...) {
			uri, err := identity.ParseResourceURI(rawURI)
			if err != nil || namespace != "" && uri.Namespace != namespace {
				return artifactcohort.ErrDenied
			}
		}
		doc, err := store.Get(ctx, cohort.PolicyResource)
		if err != nil {
			return artifactcohort.ErrDenied
		}
		used[doc.Resource] = true
		_, named, err := authz.ResolveNamedDocument(ctx, store, doc)
		if err != nil {
			return artifactcohort.ErrDenied
		}
		for _, dependency := range named {
			used[dependency.Resource] = true
		}
	}
	for _, doc := range manifest.Policies {
		if !used[doc.Resource] {
			return artifactcohort.ErrDenied
		}
	}
	return nil
}
func artifactRootInNamespace(root artifactcohort.RootRef, namespace string) bool {
	if namespace == "" {
		return true
	}
	uri, err := identity.ParseResourceURI(root.URI)
	return err == nil && uri.Namespace == namespace && uri.Kind == root.Kind
}
