package access

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/authz"
)

type providerResult struct {
	facts authz.Facts
	err   error
}

func (p providerResult) Resolve(context.Context) (authz.Facts, error) { return p.facts, p.err }

func TestSubjectProviderPreservesUnavailableAuthority(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider authz.Provider
		want     error
	}{
		{"missing", nil, authz.ErrUnavailable},
		{"outage", providerResult{err: authz.ErrUnavailable}, authz.ErrUnavailable},
		{"denied", providerResult{err: authz.ErrDenied}, authz.ErrDenied},
		{"different subject", providerResult{facts: authz.Facts{Subject: "other"}}, authz.ErrDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts, err := (subjectProvider{Provider: tc.provider, subject: "alice"}).Resolve(context.Background())
			if !errors.Is(err, tc.want) || facts.Subject != "" {
				t.Fatalf("facts=%+v err=%v", facts, err)
			}
		})
	}
}
