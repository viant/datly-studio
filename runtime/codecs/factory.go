// Package codecs composes Studio codecs with Datly's configured JWT verifier.
package codecs

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/runtime/accesscontext"
	nativeauth "github.com/viant/datly/runtime/auth"
	xcodec "github.com/viant/xdatly/codec"
	"strings"
)

type factory struct{ jwt xcodec.Factory }

func New(jwt xcodec.Factory) xcodec.Factory { return factory{jwt: jwt} }
func (f factory) New(config *xcodec.Config, options ...xcodec.Option) (xcodec.Instance, error) {
	if config != nil && strings.EqualFold(strings.TrimSpace(config.Body), nativeauth.JwtClaim) {
		if f.jwt == nil {
			return nil, fmt.Errorf("JwtClaim requires a host-configured JWT verifier")
		}
		return f.jwt.New(config, options...)
	}
	return accesscontext.Codecs().New(config, options...)
}

type contextKey struct{}

func WithFactory(ctx context.Context, value xcodec.Factory) context.Context {
	return context.WithValue(ctx, contextKey{}, value)
}
func FromContext(ctx context.Context) xcodec.Factory {
	if f, ok := ctx.Value(contextKey{}).(xcodec.Factory); ok {
		return f
	}
	return New(nil)
}
