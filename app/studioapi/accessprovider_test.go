package studioapi

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
)

const registeredFactoryTestChild = "STUDIO_API_FACTORY_TEST_CHILD"

type appFactoryProvider struct{ subject string }

func (p appFactoryProvider) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: p.subject}, nil
}

func TestConfiguredAccessProviderUsesRegisteredStartupFactory(t *testing.T) {
	if os.Getenv(registeredFactoryTestChild) != "1" {
		command := exec.Command(os.Args[0], "-test.run=^TestConfiguredAccessProviderUsesRegisteredStartupFactory$")
		command.Env = append(os.Environ(), registeredFactoryTestChild+"=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("isolated startup factory selection failed: %v\n%s", err, output)
		}
		return
	}
	provider := appFactoryProvider{subject: "operator-bound"}
	if err := accessprovider.RegisterEnvironmentFactory(func(context.Context, accessprovider.Config) (authz.Provider, error) {
		return provider, nil
	}); err != nil {
		t.Fatal(err)
	}
	actual, err := configuredAccessProvider(context.Background(), Options{}, accessprovider.Config{})
	if err != nil || actual != provider {
		t.Fatalf("configured provider=%v err=%v", actual, err)
	}
}
