package namespacevisibility

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	xexec "github.com/viant/xdatly/exec"
)

// IsAdministrator accepts only the configured exact role in verified provider
// facts. It grants namespace administration, never component execution.
func IsAdministrator(ctx context.Context, subject string) bool {
	role := os.Getenv("STUDIO_NAMESPACE_ADMIN_ROLE")
	if role == "" || role == "*" || strings.TrimSpace(role) != role || subject == "" || ctx.Err() != nil {
		return false
	}
	// SQL predicates bind the verified Authorization header in a derived context.
	// Finalizers share the invocation context, but not that derived bearer value.
	// Retain only a private, expiring decision in this invocation, never a token.
	execution := xexec.GetContext(ctx)
	bearer := oauth.Bearer(ctx)
	if execution != nil {
		if value, ok := execution.Value(adminDecisionKey); ok {
			if decision, ok := value.(adminDecision); ok && decision.subject == subject && decision.role == role && decision.until.After(time.Now()) && (bearer == "" || decision.credential == sha256.Sum256([]byte(bearer))) {
				return true
			}
		}
	}
	// A verified negative role result remains stable within this invocation.
	// Positive administration still requires the separately bound decision below.
	if saved, ok := scopedRoleFacts(ctx, subject); ok && !matchesAdministrator(role, subject, saved, time.Now()) {
		return false
	}
	provider, err := accessprovider.FromEnvironment(ctx)
	if err != nil || provider == nil {
		return false
	}
	facts, err := provider.Resolve(ctx)
	if err != nil || ctx.Err() != nil || facts.Subject != subject || facts.Issuer == "" || !facts.ValidUntil.After(time.Now()) {
		return false
	}
	rememberRoles(ctx, facts)
	if !matchesAdministrator(role, subject, facts, time.Now()) {
		return false
	}
	if execution != nil && bearer != "" {
		execution.SetValue(adminDecisionKey, adminDecision{subject: subject, role: role, until: facts.ValidUntil, credential: sha256.Sum256([]byte(bearer))})
	}
	return true
}

func matchesAdministrator(role, subject string, facts authz.Facts, now time.Time) bool {
	if role == "" || role == "*" || strings.TrimSpace(role) != role || subject == "" || facts.Subject != subject || facts.Issuer == "" || facts.Tenant == "" || !facts.ValidUntil.After(now) {
		return false
	}
	for _, actual := range facts.Roles {
		if actual == role {
			return true
		}
	}
	return false
}

const adminDecisionKey = "datly-studio.namespace-administrator"

type adminDecision struct {
	subject    string
	role       string
	until      time.Time
	credential [sha256.Size]byte
}
