package sqltransport

import (
	"context"
	"github.com/viant/datly-studio/sdk"
	"testing"
)

func TestACLUnavailableInDevelopmentAndPublicModes(t *testing.T) {
	for _, test := range []struct {
		name        string
		development bool
		mode        string
		allowed     bool
	}{
		{"development", true, "required", false},
		{"public", false, "public", true},
		{"both", true, "public", false},
		{"unknown", false, "", true},
		{"authenticated", false, "required", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner", Development: test.development})
			transport := &Transport{RuntimeProbe: RuntimeHostProbeFunc(func(context.Context) (*sdk.RuntimeHost, error) {
				return &sdk.RuntimeHost{AuthenticationMode: test.mode}, nil
			})}
			if transport.aclAvailable(ctx) != test.allowed {
				t.Fatal("incorrect ACL availability")
			}
			if !test.allowed {
				for _, operation := range []string{sdk.OperationACLList, sdk.OperationACLUpsert, sdk.OperationACLDelete} {
					if err := transport.acl(ctx, operation, nil, nil); err == nil {
						t.Fatalf("allowed %s", operation)
					}
				}
			}
		})
	}
}
