package host

import (
	"github.com/viant/datly-studio/studio/settings"
)

// ValidateSecuritySettings limits namespace selection to trusted linked handlers.
func (a *Authoring) ValidateSecuritySettings(value *settings.Security) error {
	return a.predicates.ValidateSecuritySettings(value)
}
