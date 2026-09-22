package connectorsecret

import (
	"context"
	"fmt"
	"strings"

	"github.com/viant/scy/cred/secret"
)

// Resolver expands a server-held DSN template from one secret reference.
// Implementations must never return secret material to browser DTOs or logs.
type Resolver interface {
	Resolve(context.Context, string, string) (string, error)
}

type ResolverFunc func(context.Context, string, string) (string, error)

func (f ResolverFunc) Resolve(ctx context.Context, template, reference string) (string, error) {
	return f(ctx, template, reference)
}

// SCY resolves local, cloud, or encrypted Scy resources and expands their
// values into the server-held template. A plain secret with an empty template
// is treated as the complete DSN.
type SCY struct{}

func (SCY) Resolve(ctx context.Context, template, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return strings.TrimSpace(template), nil
	}
	value, err := secret.New().Lookup(ctx, secret.Resource(reference))
	if err != nil {
		return "", fmt.Errorf("resolve connector secret: %w", err)
	}
	if strings.TrimSpace(template) == "" && value.IsPlain {
		return strings.TrimSpace(value.String()), nil
	}
	resolved := strings.TrimSpace(value.Expand(template))
	if resolved == "" {
		return "", fmt.Errorf("connector secret produced an empty DSN")
	}
	return resolved, nil
}

func Resolve(ctx context.Context, resolver Resolver, template, reference string) (string, error) {
	if strings.TrimSpace(reference) == "" {
		return strings.TrimSpace(template), nil
	}
	if resolver == nil {
		resolver = SCY{}
	}
	return resolver.Resolve(ctx, template, reference)
}
