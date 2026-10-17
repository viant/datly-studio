package preview

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xhandler "github.com/viant/xdatly/handler"
	xpredicate "github.com/viant/xdatly/predicate"
)

type PreviewIdentity struct{ ID int }
type PreviewOwnerPredicate struct{}

func (*PreviewOwnerPredicate) Compute(_ context.Context, value any) (*xpredicate.Criteria, error) {
	identity, ok := value.(*PreviewIdentity)
	if !ok || identity == nil {
		return &xpredicate.Criteria{Expression: "1=0"}, nil
	}
	return &xpredicate.Criteria{Expression: "ID = ?", Placeholders: []any{identity.ID}}, nil
}

type previewIdentityInput struct {
	Token string `parameter:"Token,kind=header,in=Authorization,required=true"`
}
type previewIdentityHandler struct{}

func (previewIdentityHandler) Exec(_ context.Context, _ xhandler.Session, input *previewIdentityInput, output *PreviewIdentity) error {
	switch input.Token {
	case "Bearer alice":
		output.ID = 1
	case "Bearer bob":
		output.ID = 2
	default:
		return fmt.Errorf("identity rejected")
	}
	return nil
}

func checkPreviewDependencyIdentity(t *testing.T, db *sql.DB, root, original string) {
	ctx := context.Background()
	dql := `#package('example.com/app/dynamic/vendors')
#import('identity','github.com/viant/datly-studio/runtime/preview')
#import('time','time')
#setting($_ = $connector('vendor'))
#setting($_ = $route('/vendors','GET'))
#define($_ = $Identity<*identity.PreviewIdentity>(component/GET:/identity).WithPredicate(0,'handler','identity.PreviewOwnerPredicate'))
#define($_ = $From<*time.Time>(form/from).Tag('format:"dateFormat=YYYY-MM-DD"').Optional().WithPredicate(0,'expr','? IS NOT NULL'))
#define($_ = $Vendors<[]*Vendor>(output/view))
SELECT vendors.*,type(vendors,'Vendor') FROM (SELECT ID,NAME FROM VENDOR ${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}) vendors`
	if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", dql, dql); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", original, original); err != nil {
			t.Fatal(err)
		}
	}()
	types := typecatalog.NewCatalog()
	for _, typ := range []reflect.Type{reflect.TypeFor[PreviewIdentity](), reflect.TypeFor[PreviewOwnerPredicate]()} {
		if err := types.Register(typecatalog.TypeOriginPackage, x.NewType(typ)); err != nil {
			t.Fatal(err)
		}
	}
	closed := 0
	dynamic := Dynamic{StudioDB: db, RootDir: root, Types: types, Dependencies: func(context.Context, *spec.Component) (*DependencyRuntime, error) {
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "test/identity", Name: "Identity"}, Routes: []*spec.Route{{Method: "GET", Path: "/identity", Internal: true}}}, InputType: reflect.TypeFor[previewIdentityInput](), OutputType: reflect.TypeFor[PreviewIdentity]()})
		if err != nil {
			return nil, err
		}
		registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[previewIdentityInput, PreviewIdentity](previewIdentityHandler{})})
		if err != nil {
			return nil, err
		}
		return &DependencyRuntime{Components: []*registry.RegisteredComponent{registered}, Shutdown: func(context.Context) error { closed++; return nil }}, nil
	}}
	for _, test := range []struct{ token, want, absent string }{{"alice", "Northwind", "Contoso"}, {"bob", "Contoso", "Northwind"}} {
		verified := sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: test.token}), sdk.VerifiedCredential{Bearer: test.token, Claims: struct{}{}})
		result, err := dynamic.Execute(verified, "vendors", 1, sdk.PreviewInput{Input: []byte(`{"From":"2026-09-01"}`), Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(result.Data), test.want) || strings.Contains(string(result.Data), test.absent) {
			t.Fatalf("predicate result for %s: %s", test.token, result.Data)
		}
		if _, err = dynamic.Execute(verified, "vendors", 1, sdk.PreviewInput{Input: []byte(`{"Identity":{"ID":3}}`)}); err == nil {
			t.Fatal("client replaced identity dependency")
		}
	}
	if _, err := dynamic.Execute(ctx, "vendors", 1, sdk.PreviewInput{}); err == nil {
		t.Fatal("missing authentication accepted")
	}
	if closed != 5 {
		t.Fatalf("dependency cleanup count=%d", closed)
	}
}
