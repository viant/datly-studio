package versionrun

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	runaccess "github.com/viant/datly-studio/studio/reports/store_run_access"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type testSession struct{ binder *testBinder }

func (s testSession) Binder() xhandler.Binder  { return s.binder }
func (testSession) Response() xresponse.Writer { return nil }

type testBinder struct {
	db      *sql.DB
	bearer  string
	allowed bool
}

func (b *testBinder) Bind(_ context.Context, target any) error {
	reflect.ValueOf(target).Elem().FieldByName("Authorization").SetString(b.bearer)
	return nil
}
func (b *testBinder) Lookup(_ context.Context, key xhandler.ValueKey) (any, bool, error) {
	switch key {
	case exec.ComponentInvokerKey, rhandler.ConnectorCapabilityKey:
		return b, true, nil
	}
	return nil, false, nil
}
func (b *testBinder) Connector(context.Context, string) (*sql.DB, error) { return b.db, nil }
func (b *testBinder) InvokeComponent(_ context.Context, request exec.ComponentRequest) (any, error) {
	input, ok := request.Input.(*runaccess.Input)
	if !ok || input.ReportId != "report" || input.Subject != "alice" {
		return nil, fmt.Errorf("wrong run-access identity")
	}
	result := &runaccess.Output{}
	if b.allowed {
		result.Allowed = []*runaccess.RunAccess{{ReportId: "report"}}
	}
	return result, nil
}

func TestAuthorizedRetainsNativeBearerAndServerAuthority(t *testing.T) {
	for _, name := range []string{"STUDIO_ACCESS_ISSUER", "STUDIO_ACCESS_AUDIENCE", "STUDIO_ACCESS_PUBLIC_KEY_FILE", "STUDIO_ACCESS_USER_INFO_URL", "STUDIO_ACCESS_TENANT"} {
		t.Setenv(name, "")
	}
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	binder := &testBinder{db: new(sql.DB), allowed: true, bearer: "Bearer native-token"}
	ctx, engine, cancel, err := Authorized(context.Background(), testSession{binder}, claims, auth, "report", 11, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if engine.Access != nil {
		t.Fatal("absent provider fabricated authority")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://trusted.example")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", path)
	t.Setenv("STUDIO_ACCESS_TENANT", "tenant")
	ctx, engine, cancel, err = Authorized(context.Background(), testSession{binder}, claims, auth, "report", 11, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	principal, ok := sdk.PrincipalFromContext(ctx)
	credential, verified := sdk.VerifiedCredentialFromContext(ctx)
	if !ok || principal.Subject != "alice" || !verified || credential.Bearer != "native-token" || engine.Access == nil || engine.AccessTenant != "tenant" {
		t.Fatal("native identity or server authority lost")
	}
	if _, err := engine.Access.Provider.Resolve(ctx); err == nil {
		t.Fatal("opaque token unexpectedly supplied authorization facts")
	}
	binder.bearer = ""
	if _, _, _, err := Authorized(context.Background(), testSession{binder}, claims, auth, "report", 11, time.Second); err == nil {
		t.Fatal("missing configured-provider bearer accepted")
	}
	binder.allowed = false
	if _, _, _, err := Authorized(context.Background(), testSession{binder}, claims, auth, "report", 11, time.Second); err == nil {
		t.Fatal("run-access denial bypassed")
	}
}
