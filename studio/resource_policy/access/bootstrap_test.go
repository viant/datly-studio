package access

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
)

const nativeFactoryChildEnv = "STUDIO_NATIVE_ACCESS_FACTORY_CHILD"

type nativeFactoryProvider struct {
	subject string
	calls   int
}

func (p *nativeFactoryProvider) Resolve(ctx context.Context) (authz.Facts, error) {
	p.calls++
	if oauth.Bearer(ctx) == "" {
		return authz.Facts{}, authz.ErrIdentityDenied
	}
	return authz.Facts{Subject: p.subject, Issuer: "test", Tenant: "tenant", ValidUntil: time.Now().Add(time.Minute)}, nil
}

// NativeFactorySubprocess returns nil in the parent process after it has run
// the target test in a fresh process. The child registers before any native
// handler calls FromEnvironment, isolating the process singleton.
func nativeFactorySubprocess(t *testing.T, target string) *nativeFactoryProvider {
	t.Helper()
	if os.Getenv(nativeFactoryChildEnv) != "1" {
		command := exec.Command(os.Args[0], "-test.run=^"+target+"$")
		command.Env = append(os.Environ(), nativeFactoryChildEnv+"=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("isolated native factory test failed: %v\n%s", err, output)
		}
		return nil
	}
	provider := &nativeFactoryProvider{subject: "owner"}
	err := accessprovider.RegisterEnvironmentFactory(func(context.Context, accessprovider.Config) (authz.Provider, error) {
		return provider, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}
