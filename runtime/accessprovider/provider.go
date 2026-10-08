// Package accessprovider defines the trusted startup and request-context
// bindings used by Studio's resource authorization paths.
package accessprovider

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/accessconfig"
)

// Config contains operator-owned resource identity settings. Settings carries
// any additional opaque configuration owned by an embedding host; Studio does
// not interpret it.
type Config struct {
	Issuer, Audience, PublicKeyFile, CertURL, UserInfoURL string
	Settings                                              []byte
}

// ProviderFactory creates one process-owned resource identity provider.
// Implementations should bind only deployment configuration and trusted
// remote authorities; request data must not choose a factory or its settings.
type ProviderFactory func(context.Context, Config) (authz.Provider, error)

// EnvironmentConfig reads operator-owned environment values for a startup
// resolution. Hosts should call it during startup, not from request values.
func EnvironmentConfig() Config {
	return Config{
		Issuer:        strings.TrimSpace(os.Getenv("STUDIO_ACCESS_ISSUER")),
		Audience:      strings.TrimSpace(os.Getenv("STUDIO_ACCESS_AUDIENCE")),
		PublicKeyFile: strings.TrimSpace(os.Getenv("STUDIO_ACCESS_PUBLIC_KEY_FILE")),
		CertURL:       strings.TrimSpace(os.Getenv("STUDIO_ACCESS_CERT_URL")),
		UserInfoURL:   strings.TrimSpace(os.Getenv("STUDIO_ACCESS_USER_INFO_URL")),
	}
}

// New resolves the operator configuration with the supplied host factory. A
// nil factory uses Studio's generic JWT identity provider and fails closed if
// a user-info authority was configured without a host-owned adapter.
func New(ctx context.Context, config Config, factory ProviderFactory) (authz.Provider, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, authz.ErrDenied
	}
	if factory != nil {
		provider, err := factory(ctx, config)
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, errors.New("resource access provider factory returned nil")
		}
		return provider, nil
	}
	return accessconfig.New(accessconfig.Config{Issuer: config.Issuer, Audience: config.Audience,
		PublicKeyFile: config.PublicKeyFile, CertURL: config.CertURL, UserInfoURL: config.UserInfoURL})
}

// FromEnvironment resolves the process's registered host factory. The first
// environment lookup freezes registration; hosts must register at startup.
// Per-request handlers may read a trusted context binding but cannot replace
// the process factory from request values.
func FromEnvironment(ctx context.Context) (authz.Provider, error) {
	if provider, ok := FromContext(ctx); ok {
		return provider, nil
	}
	factory := resolveRegisteredFactory()
	if factory == nil {
		return New(ctx, EnvironmentConfig(), nil)
	}
	environmentState.Do(func() {
		environmentState.provider, environmentState.err = New(context.Background(), EnvironmentConfig(), factory)
	})
	return environmentState.provider, environmentState.err
}

var processFactory struct {
	sync.RWMutex
	factory  ProviderFactory
	set      bool
	resolved bool
}

var environmentState struct {
	sync.Once
	provider authz.Provider
	err      error
}

// RegisterEnvironmentFactory installs the embedding host's startup factory.
// A second registration or a registration after the first environment lookup
// fails, so one process cannot silently change authority.
func RegisterEnvironmentFactory(factory ProviderFactory) error {
	if factory == nil {
		return errors.New("resource access provider factory is required")
	}
	processFactory.Lock()
	defer processFactory.Unlock()
	if processFactory.set {
		return errors.New("resource access provider factory is already registered")
	}
	if processFactory.resolved {
		return errors.New("resource access provider environment was resolved before startup registration")
	}
	processFactory.factory = factory
	processFactory.set = true
	return nil
}

func registeredFactory() ProviderFactory {
	processFactory.RLock()
	defer processFactory.RUnlock()
	return processFactory.factory
}

func resolveRegisteredFactory() ProviderFactory {
	processFactory.Lock()
	defer processFactory.Unlock()
	processFactory.resolved = true
	return processFactory.factory
}

// RegisteredEnvironmentFactory returns the immutable startup factory, if the
// embedding process installed one. It lets command-line wrappers pass the same
// binding through RunWithOptions while preserving CLI-resolved configuration.
func RegisteredEnvironmentFactory() ProviderFactory { return registeredFactory() }

type contextKey struct{}

// WithProvider places a trusted provider into a request context. Only startup
// or verified host middleware should call this function.
func WithProvider(ctx context.Context, provider authz.Provider) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, contextKey{}, provider)
}

// FromContext returns a provider explicitly bound by trusted host middleware.
func FromContext(ctx context.Context) (authz.Provider, bool) {
	if ctx == nil {
		return nil, false
	}
	provider, ok := ctx.Value(contextKey{}).(authz.Provider)
	return provider, ok && provider != nil
}
