package host

import (
	"context"
	"fmt"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"github.com/viant/datly-studio/studio/settings"
)

type trustedNativeInvocationKey struct{}

func (s *Service) runtimePolicyRegistry() *predicatepolicy.RuntimePolicies {
	s.runtimePolicyMu.Lock()
	defer s.runtimePolicyMu.Unlock()
	if s.config.runtimePolicies == nil {
		s.config.runtimePolicies = &predicatepolicy.RuntimePolicies{}
	}
	return s.config.runtimePolicies
}

// ApplyNamespaceSettings is a trusted post-commit update. Existing public
// transport callbacks observe it even if a subsequent generation build fails.
func (s *Service) ApplyNamespaceSettings(ctx context.Context, identity change.Identity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !identity.SecurityChanged {
		return nil
	}
	if identity.OwnerID == "" || identity.Name == "" || identity.Revision <= 0 || identity.NamespaceID != namespaceaccess.ID(identity.OwnerID, identity.Name) {
		return fmt.Errorf("invalid committed namespace security identity")
	}
	s.runtimePolicyRegistry().Update(identity.NamespaceID, identity.Revision, &settings.Namespace{Security: identity.Security})
	return nil
}
