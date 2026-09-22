package reader_test

import (
	"context"
	"net/http/httptest"
	"testing"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	authreader "github.com/viant/datly-studio/studio/auth/reader"
	druntime "github.com/viant/datly/runtime"
)

func TestAuthContextResolvesVerifiedJWTClaims(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "auth_context", "studio")
	jwt := datatest.NewJWTFixture(t)
	resources := resource.New()
	entry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	runtime, err := druntime.NewRuntime([]*druntime.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	request := httptest.NewRequest("GET", "/v1/studio/auth/context", nil)
	request.Header.Set("Authorization", jwt.Bearer(t, "owner-a"))
	scope, err := requestprovider.New(request)
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	actual, err := runtime.ExecuteRoute(ctx, "GET", "/v1/studio/auth/context", scope)
	if err != nil {
		t.Fatal(err)
	}
	output := actual.(*authreader.Output)
	if output.Auth == nil || output.Auth.Subject != "owner-a" {
		t.Fatalf("auth=%+v", output.Auth)
	}
}
