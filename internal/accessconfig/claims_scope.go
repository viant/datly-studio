package accessconfig

import (
	"context"
	"crypto/sha256"
	"slices"
	"sync"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/gating"
	"github.com/viant/authz/oauth"
)

type identityScopeKey struct{}
type identityScope struct {
	mu               sync.Mutex
	provider         *verifiedIdentity
	source           context.Context
	bearer           [32]byte
	principal        gating.Principal
	metadata, closed bool
}

// JWT identity is immutable for the lifetime of its signed credential. These
// scopes retain only its verified projection, never a permission decision.
func (p *verifiedIdentity) BeginMetadataRead(ctx context.Context) (context.Context, func() error, error) {
	return p.beginIdentityScope(ctx, true)
}
func (p *verifiedIdentity) BeginOperation(ctx context.Context) (context.Context, func() error, error) {
	return p.beginIdentityScope(ctx, false)
}
func (p *verifiedIdentity) beginIdentityScope(ctx context.Context, metadata bool) (context.Context, func() error, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, nil, authz.ErrDenied
	}
	if state, _ := ctx.Value(identityScopeKey{}).(*identityScope); state != nil {
		if _, err := state.resolve(ctx, p); err != nil {
			return nil, nil, err
		}
		return ctx, func() error { _, err := state.resolve(ctx, p); return err }, nil
	}
	principal, err := p.readPrincipal(ctx)
	if err != nil {
		return nil, nil, err
	}
	state := &identityScope{provider: p, source: ctx, bearer: sha256.Sum256([]byte(oauth.Bearer(ctx))), principal: cloneIdentityPrincipal(principal), metadata: metadata}
	scoped := context.WithValue(ctx, identityScopeKey{}, state)
	finish := func() error {
		state.mu.Lock()
		defer state.mu.Unlock()
		err := state.validate(scoped, p)
		state.closed = true
		return err
	}
	return scoped, finish, nil
}
func (s *identityScope) validate(ctx context.Context, p *verifiedIdentity) error {
	if ctx == nil || ctx.Err() != nil || s.provider != p || s.closed || s.source.Err() != nil || s.bearer != sha256.Sum256([]byte(oauth.Bearer(ctx))) || !s.principal.Facts.ValidUntil.After(time.Now()) {
		return authz.ErrDenied
	}
	return nil
}
func (s *identityScope) resolve(ctx context.Context, p *verifiedIdentity) (gating.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validate(ctx, p); err != nil {
		return gating.Principal{}, err
	}
	return cloneIdentityPrincipal(s.principal), nil
}
func (p *verifiedIdentity) WithoutMetadataRead(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, identityScopeKey{}, (*identityScope)(nil))
}
func (p *verifiedIdentity) MetadataReadActive(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	state, _ := ctx.Value(identityScopeKey{}).(*identityScope)
	if state == nil {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.metadata && state.validate(ctx, p) == nil
}
func cloneIdentityPrincipal(p gating.Principal) gating.Principal {
	p.Facts.Roles = slices.Clone(p.Facts.Roles)
	p.Facts.Exposures = slices.Clone(p.Facts.Exposures)
	p.Facts.GrantedScopes = slices.Clone(p.Facts.GrantedScopes)
	p.MembershipGroups = slices.Clone(p.MembershipGroups)
	p.Facts.Entities = slices.Clone(p.Facts.Entities)
	p.Facts.EntityPermissions = slices.Clone(p.Facts.EntityPermissions)
	for i := range p.Facts.EntityPermissions {
		p.Facts.EntityPermissions[i].Permissions = slices.Clone(p.Facts.EntityPermissions[i].Permissions)
	}
	if p.Facts.EntityGroups != nil {
		groups := make(authz.EntityGroups, len(p.Facts.EntityGroups))
		for key, ids := range p.Facts.EntityGroups {
			groups[key] = slices.Clone(ids)
		}
		p.Facts.EntityGroups = groups
	}
	return p
}
