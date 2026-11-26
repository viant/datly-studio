package host

import "github.com/viant/datly-studio/sdk"

// RuntimeDiagnostics reports immutable preparation outcomes. No source values,
// SQL, credentials or raw driver messages are exposed to management callers.
func (s *Service) RuntimeDiagnostics() []sdk.Diagnostic {
	s.diagnosticMu.RLock()
	defer s.diagnosticMu.RUnlock()
	return append([]sdk.Diagnostic(nil), s.readerDiagnostics...)
}
