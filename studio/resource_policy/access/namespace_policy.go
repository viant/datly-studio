package access

import (
	"context"
	"github.com/viant/authz"
	store "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/internal/namespacepolicy"
)

// Namespace defaults are administered by the current owner/delegated managers,
// not by an editable manageAccess rule or a deployment-wide policy-editor role.
func canManageNamespacePolicy(ctx context.Context, service *authz.Service, facts authz.Facts, resource authz.Resource) (bool, error) {
	policyStore, ok := service.Store.(*store.Store)
	if !ok {
		return false, authz.ErrDenied
	}
	reader, err := namespacepolicy.NewReader(policyStore.DB)
	if err != nil {
		return false, err
	}
	defer reader.Close(context.WithoutCancel(ctx))
	return reader.CanManage(ctx, facts, resource)
}

func checkNamespacePolicy(ctx context.Context, service *authz.Service, namespaceID *string, resource authz.Resource) error {
	// The namespace resource is itself an explicit scope. HTTP may additionally
	// carry a workspace header; MCP clients can address the resource directly.
	if resource.ID == "" || resource.Version != "1" || namespaceID != nil && *namespaceID != resource.ID {
		return publicError(403, "Select the namespace whose default policy you want to manage")
	}
	allowed, err := creationAdministration(ctx, service).CanReplace(ctx, resource)
	if err != nil || !allowed {
		return publicError(403, "Namespace owner or manager permission is required")
	}
	return nil
}

func validateNamespacePolicy(document authz.Document) error {
	if document.Resource.Kind != "namespace" {
		return nil
	}
	for action := range document.Policies {
		switch action {
		case "discover", "describe", "execute":
		default:
			return publicError(400, "Namespace defaults support discover, describe and execute")
		}
	}
	return nil
}
