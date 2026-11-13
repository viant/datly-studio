package preview

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	runtimeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	"github.com/viant/x"
	xhandler "github.com/viant/xdatly/handler"
	handlerexec "github.com/viant/xdatly/handler/exec"
)

type warmupJWTIdentityInput struct {
	JWT *jwt.Claims `parameter:"JWT,kind=header,in=Authorization,dataType=string,required,errorCode=401" codec:"JwtClaim"`
}
type warmupAlternateJWTIdentityInput struct {
	JWT *jwt.Claims `parameter:"JWT,kind=header,in=X-Identity,dataType=string,required,errorCode=401" codec:"JwtClaim"`
}
type warmupJWTIdentityHandler struct{}

func (warmupJWTIdentityHandler) Exec(_ context.Context, _ xhandler.Session, input *warmupJWTIdentityInput, output *PreviewIdentity) error {
	if input.JWT == nil || input.JWT.Subject != "alice" {
		return fmt.Errorf("identity rejected")
	}
	output.ID = int(input.JWT.UserID)
	return nil
}

type warmupAlternateJWTIdentityHandler struct{}

func (warmupAlternateJWTIdentityHandler) Exec(_ context.Context, _ xhandler.Session, _ *warmupAlternateJWTIdentityInput, _ *PreviewIdentity) error {
	return fmt.Errorf("unsupported header handler must not execute")
}

func checkWarmupJWTCredentials(ctx context.Context, t *testing.T, db *sql.DB, root, source string, types *typecatalog.Catalog) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	factory, err := runtimeauth.New(ctx, &runtimeauth.Config{JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: "fixture-public-key", Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeFor[jwt.Claims]())); err != nil {
		t.Fatal(err)
	}
	sign := func(subject string, expires time.Time) string {
		token, e := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, jwtv5.MapClaims{"sub": subject, "user_id": 1, "exp": expires.Unix()}).SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return token
	}
	valid := sign("alice", time.Now().Add(time.Hour))
	defer func() {
		if _, e := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", source, source); e != nil {
			t.Error(e)
		}
	}()
	for _, tc := range []struct {
		name, token, extra, header string
		wantSuccess                bool
	}{
		{name: "valid JWT", token: valid, wantSuccess: true},
		{name: "missing JWT"},
		{name: "invalid JWT", token: "invalid.token.signature"},
		{name: "expired JWT", token: sign("alice", time.Now().Add(-time.Hour))},
		{name: "identity rejected", token: sign("bob", time.Now().Add(time.Hour))},
		{name: "unsupported credential header", token: valid, header: "X-Identity"},
		{name: "authored credential override", token: valid, extra: ",'Caller=untrusted'"},
		{name: "authored credential index", token: valid, extra: ",'IndexParameter=Caller'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cachePath := filepath.Join(t.TempDir(), "cache")
			dql := strings.Replace(source, "SELECT vendors.*", fmt.Sprintf("#setting($_ = $cache('vendors', '1m').WithLocation('%s'))\n#setting($_ = $cache_warmup('ID','Limit=0'%s))\nSELECT vendors.*", cachePath, tc.extra), 1)
			if tc.extra != "" {
				dql = "#import('jwt','github.com/viant/scy/auth/jwt')\n#define($_ = $Caller<string,*jwt.Claims>(header/Authorization).WithCodec('JwtClaim').Required())\n" + dql
			}
			if _, e := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", dql, dql); e != nil {
				t.Fatal(e)
			}
			predicates, closed := 0, 0
			var phases []handlerexec.WarmupPhase
			callCtx := context.WithValue(ctx, warmupPredicateCallsKey{}, &predicates)
			callCtx = context.WithValue(callCtx, warmupInvocationPhasesKey{}, &phases)
			callCtx = studiocodecs.WithFactory(callCtx, studiocodecs.New(factory))
			if tc.token != "" {
				callCtx = sdk.WithVerifiedCredential(callCtx, sdk.VerifiedCredential{Bearer: tc.token, Claims: struct{}{}})
			}
			dynamic := Dynamic{StudioDB: db, RootDir: root, Types: types, Dependencies: func(ctx context.Context, _ *spec.Component) (*DependencyRuntime, error) {
				inputType := reflect.TypeFor[warmupJWTIdentityInput]()
				handler := custom.New[warmupJWTIdentityInput, PreviewIdentity](warmupJWTIdentityHandler{})
				if tc.header != "" {
					inputType = reflect.TypeFor[warmupAlternateJWTIdentityInput]()
					handler = custom.New[warmupAlternateJWTIdentityInput, PreviewIdentity](warmupAlternateJWTIdentityHandler{})
				}
				artifact, e := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "test/identity", Name: "Identity"}, Routes: []*spec.Route{{Method: "GET", Path: "/identity", Internal: true}}}, InputType: inputType, OutputType: reflect.TypeFor[PreviewIdentity](), CodecFactory: factory})
				if e != nil {
					return nil, e
				}
				entry, e := artifact.Registration(registry.RegisteredComponent{Handler: handler})
				if e != nil {
					return nil, e
				}
				return &DependencyRuntime{Components: []*registry.RegisteredComponent{entry}, Shutdown: func(context.Context) error { closed++; return nil }}, nil
			}}
			result, e := dynamic.Warmup(callCtx, "vendors", 1)
			if tc.wantSuccess {
				if e != nil {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(phases, []handlerexec.WarmupPhase{handlerexec.WarmupPhasePrepare, handlerexec.WarmupPhaseFill}) {
					t.Fatalf("native warmup phases=%v", phases)
				}
				if result.CompletedCases != 1 || result.Entries < 1 || predicates < 2 {
					t.Fatalf("result=%+v predicate calls=%d", result, predicates)
				}
			} else {
				if e == nil {
					t.Fatal("rejected credentials/override accepted")
				}
				if tc.header != "" && !strings.Contains(e.Error(), "Studio cannot supply warmup credential header") {
					t.Fatalf("wrong unsupported requirement error: %v", e)
				}
				if tc.extra != "" && !strings.Contains(e.Error(), "JWT input Caller") {
					var detail *sdk.Error
					if !errors.As(e, &detail) {
						t.Fatal(e)
					}
					t.Fatalf("wrong authored credential override error: %v cause: %v", e, detail.Cause)
				}
				if tc.extra == "" && tc.header == "" && !strings.Contains(e.Error(), "prepare dynamic cache warmup") {
					t.Fatalf("not rejected during native preparation: %v", e)
				}
				if predicates != 0 {
					t.Fatalf("rejected credentials reached reader predicate %d times", predicates)
				}
			}
			if closed != 1 {
				t.Fatalf("dependency cleanup calls=%d", closed)
			}
			files := 0
			walkErr := filepath.WalkDir(cachePath, func(_ string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !entry.IsDir() {
					files++
				}
				return nil
			})
			if walkErr != nil && !os.IsNotExist(walkErr) {
				t.Fatal(walkErr)
			}
			if tc.wantSuccess && files == 0 {
				t.Fatal("successful warmup wrote no cache files")
			}
			if !tc.wantSuccess && files != 0 {
				t.Fatalf("rejected preparation wrote %d cache files", files)
			}
		})
	}
}
