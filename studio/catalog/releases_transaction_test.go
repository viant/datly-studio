package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/schema"
	release "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/value"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/scy/auth/jwt"
	_ "modernc.org/sqlite"
)

type stagedReleaseManager struct {
	catalogReleaseFixture
	store  release.ReleaseStore
	bundle contract.Bundle
}

func (m *stagedReleaseManager) Publish(ctx context.Context, _ contract.PublishInput) (contract.Result, error) {
	if err := m.store.PutBundle(ctx, m.bundle); err != nil {
		return contract.Result{}, err
	}
	if _, err := m.store.Bundle(ctx, m.bundle.Hash); err != nil {
		return contract.Result{}, fmt.Errorf("read committed bundle before staging: %w", err)
	}
	return contract.Result{}, nil
}
func (m *stagedReleaseManager) Promote(ctx context.Context, _ contract.PromoteInput) (contract.Result, error) {
	return m.Publish(ctx, contract.PublishInput{})
}

func TestNativeReleaseEndpointCommitsChildBeforeNextStage(t *testing.T) {
	for _, operation := range []string{"publish", "promote"} {
		t.Run(operation, func(t *testing.T) {
			ctx := t.Context()
			db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/release.db?_pragma=busy_timeout(1000)")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
				t.Fatal(err)
			}
			scope := value.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo"}
			store, err := datatest.NativeReleaseStore(t, db, scope, "publisher")
			if err != nil {
				t.Fatal(err)
			}
			root := artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: strings.Repeat("a", 64)}
			bundle := contract.Bundle{Namespace: "demo", Roots: []artifactcohort.RootRef{root}}
			bundle.Hash = contract.BundleHash(bundle)
			manager := &stagedReleaseManager{store: store, bundle: bundle}
			ConfigureReleaseManagement(manager)
			defer ConfigureReleaseManagement(nil)
			claims := &jwt.Claims{}
			claims.Subject = "publisher"
			identity := ReleaseIdentity{Jwt: claims, Auth: &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "publisher"}}}
			var holder, inputType reflect.Type
			var input any
			var handler rhandler.TypedHandler
			switch operation {
			case "publish":
				holder = reflect.TypeFor[ReleasePublishComponent]()
				inputType = reflect.TypeFor[ReleasePublishInput]()
				handler, err = (ReleasePublishComponent{}).DatlyHandler("NewReleasePublish")()
				input = &ReleasePublishInput{ReleaseIdentity: identity, Request: contract.PublishInput{Namespace: "demo", Name: "main", Roots: []artifactcohort.RootRef{root}, Policies: []authz.NamedReference{{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}}}}
			case "promote":
				holder = reflect.TypeFor[ReleasePromoteComponent]()
				inputType = reflect.TypeFor[ReleasePromoteInput]()
				handler, err = (ReleasePromoteComponent{}).DatlyHandler("NewReleasePromote")()
				input = &ReleasePromoteInput{ReleaseIdentity: identity, Request: contract.PromoteInput{Namespace: "demo", Name: "main", BundleHash: bundle.Hash}}
			}
			if err != nil {
				t.Fatal(err)
			}
			field, _ := holder.FieldByName("Contract")
			metadata, _, err := dtag.ParseComponent(field.Tag)
			if err != nil {
				t.Fatal(err)
			}
			component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "catalog", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, reflect.TypeFor[ReleaseResultOutput]())
			if err != nil {
				t.Fatal(err)
			}
			jwtFixture := datatest.NewJWTFixture(t)
			artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: inputType, OutputType: reflect.TypeFor[ReleaseResultOutput](), Resources: resource.New(), Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwtFixture.Factory, Handler: handler, HandlerOwnedOutput: true})
			if err != nil {
				t.Fatal(err)
			}
			connector := &dsql.SQLComponent{DB: db}
			if err = connector.RegisterConnector("studio", db); err != nil {
				t.Fatal(err)
			}
			base := registry.RegisteredComponent{}
			base.Capabilities.Connector = connector
			// Match the configured host: explicit connector metadata owns a root unit.
			if component.Settings.DefaultConnector != "" {
				base.DataSource = dml.Source{DB: db}
			}
			registration, err := artifact.Registration(base)
			if err != nil {
				t.Fatal(err)
			}
			runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration})
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Shutdown(ctx)
			_, err = runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: component.Key, Route: spec.RouteRef{Method: "POST", Path: "/v1/studio/catalog/releases/" + operation}}, Input: input})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Bundle(ctx, bundle.Hash); err != nil {
				t.Fatal(err)
			}
		})
	}
}
