package namespacevisibility

import (
	"context"
	"os"
	"strings"
)

// CanCreate preserves unrestricted authenticated authoring when the embedding
// host configures no creation roles. Configured roles use verified provider
// facts and the existing invocation snapshot, never caller-supplied claims.
func CanCreate(ctx context.Context, subject string) bool {
	if subject == "" || ctx.Err() != nil {
		return false
	}
	configured := os.Getenv("STUDIO_NAMESPACE_CREATE_ROLES")
	if configured == "" {
		configured = os.Getenv("STUDIO_NAMESPACE_ADMIN_ROLE")
	}
	if configured == "" {
		return true
	}
	roles, err := verifiedRoles(ctx, subject)
	if err != nil {
		return false
	}
	for _, allowed := range strings.Split(configured, ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" || allowed == "*" {
			continue
		}
		for _, actual := range roles {
			if actual == allowed {
				return true
			}
		}
	}
	return false
}
