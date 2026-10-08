package accessprovider

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/viant/authz"
)

type testProvider struct{}

func (testProvider) Resolve(context.Context) (authz.Facts, error) { return authz.Facts{}, nil }

func TestNewUsesHostFactoryWithOperatorConfig(t *testing.T) {
	provider := testProvider{}
	config := Config{Issuer: "issuer", Audience: "audience", UserInfoURL: "https://idp.example/userinfo", Settings: []byte(`{"tenant":"team"}`)}
	calls := 0
	actual, err := New(context.Background(), config, func(_ context.Context, got Config) (authz.Provider, error) {
		calls++
		if got.Issuer != config.Issuer || got.Audience != config.Audience || got.UserInfoURL != config.UserInfoURL || string(got.Settings) != string(config.Settings) {
			t.Fatalf("host factory received unexpected config: %#v", got)
		}
		return provider, nil
	})
	if err != nil || actual != provider || calls != 1 {
		t.Fatalf("provider=%v calls=%d err=%v", actual, calls, err)
	}
}

func TestFromEnvironmentUsesTrustedContextBinding(t *testing.T) {
	provider := testProvider{}
	ctx := WithProvider(context.Background(), provider)
	actual, err := FromEnvironment(ctx)
	if err != nil || actual != provider {
		t.Fatalf("provider=%v err=%v", actual, err)
	}
}

func TestNewRejectsCanceledStartupContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(ctx, Config{}, nil); err == nil {
		t.Fatal("canceled startup accepted")
	}
}

func TestEnvironmentLookupFreezesStartupRegistration(t *testing.T) {
	const childEnv = "STUDIO_ACCESS_PROVIDER_LATE_REGISTER_CHILD"
	if os.Getenv(childEnv) == "1" {
		if _, err := FromEnvironment(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := RegisterEnvironmentFactory(func(context.Context, Config) (authz.Provider, error) { return testProvider{}, nil }); err == nil {
			t.Fatal("factory registration succeeded after native environment lookup")
		}
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestEnvironmentLookupFreezesStartupRegistration$")
	command.Env = append(os.Environ(), childEnv+"=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("isolated late-registration check failed: %v\n%s", err, output)
	}
}
