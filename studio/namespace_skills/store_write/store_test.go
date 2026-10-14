package store_write

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xhandler "github.com/viant/xdatly/handler"
)

func TestNamespaceSkillWriterAtomicHistoryAndCAS(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "namespace_skill_history", "studio")
	holder := reflect.TypeFor[SkillComponent]()
	field, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "store_write", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(SkillDatlyResourceNamespace, SkillDatlyResources); err != nil {
		t.Fatal(err)
	}
	types := typecatalog.NewCatalog()
	for _, value := range []any{SkillStoreRules{}, SkillRevisionRules{}} {
		if err = types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(value))); err != nil {
			t.Fatal(err)
		}
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[Input](), OutputType: reflect.TypeFor[Output](), Resources: resources, Types: types})
	if err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err = connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	var providers []locator.Provider
	if len(artifact.ViewDependencies) > 0 {
		views, e := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
		if e != nil {
			t.Fatal(e)
		}
		providers = append(providers, views)
	}
	writer, err := writerhandler.New(component, reflect.TypeFor[Input](), reflect.TypeFor[Output](), "patch")
	if err != nil {
		t.Fatal(err)
	}
	registration := registry.RegisteredComponent{Handler: writer, Providers: providers, DataSource: dml.Source{DB: db}}
	registration.Capabilities.Connector = connector
	entry, err := artifact.Registration(registration)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	write := func(head *StoredSkill) (*Output, error) {
		input := &Input{}
		input.SetSkills([]*StoredSkill{head})
		value, e := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: dexec.ComponentTarget{Component: component.Key, Route: spec.RouteRef{Method: "PATCH", Path: "/_studio/namespace-skill-store/write"}}, Input: input})
		if e != nil {
			return nil, e
		}
		return value.(*Output), nil
	}
	at := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	head := func(latest, published int, etag int64) *StoredSkill {
		row := &StoredSkill{}
		row.SetNamespaceId("namespace-a")
		row.SetSkillId("guide-id")
		row.SetName("guide")
		row.SetTitle("Guide")
		row.SetDescription("Namespace guide")
		row.SetLatestRevision(latest)
		row.SetPublishedRevision(published)
		row.SetStatus("active")
		row.SetEtag(&etag)
		row.SetCreatedAt(at)
		row.SetUpdatedAt(at.Add(time.Duration(etag) * time.Second))
		row.SetCreatedBy("owner")
		row.SetUpdatedBy("owner")
		return row
	}
	revision := func(number int, content string) *StoredRevision {
		body, _ := json.Marshal(map[string]string{"SKILL.md": content})
		sum := sha256.Sum256(body)
		row := &StoredRevision{}
		row.SetNamespaceId("namespace-a")
		row.SetSkillId("guide-id")
		row.SetRevision(number)
		row.SetBundleJson(body)
		row.SetContentSha256(hex.EncodeToString(sum[:]))
		row.SetCreatedAt(at)
		row.SetUpdatedAt(at)
		row.SetCreatedBy("owner")
		row.SetUpdatedBy("owner")
		return row
	}
	first := head(1, 0, 1)
	first.SetRevisions([]*StoredRevision{revision(1, "original")})
	if _, err = write(first); err != nil {
		t.Fatal(err)
	}
	if _, err = write(head(1, 1, 1)); err != nil {
		t.Fatal(err)
	}
	second := head(2, 1, 2)
	second.SetRevisions([]*StoredRevision{revision(2, "draft update")})
	if _, err = write(second); err != nil {
		t.Fatal(err)
	}
	var latest, published int
	var etag int64
	if err = db.QueryRow(`SELECT latest_revision,published_revision,etag FROM namespace_skills`).Scan(&latest, &published, &etag); err != nil || latest != 2 || published != 1 || etag != 3 {
		t.Fatalf("head latest=%d published=%d etag=%d err=%v", latest, published, etag, err)
	}
	stale := head(3, 1, 2)
	stale.SetRevisions([]*StoredRevision{revision(3, "stale")})
	if _, err = write(stale); err == nil {
		t.Fatal("stale token accepted")
	} else {
		var conflict *xhandler.Conflict
		if !errors.As(err, &conflict) {
			t.Fatalf("stale token returned %T: %v", err, err)
		}
	}
	tamper := head(2, 1, 3)
	tamper.SetRevisions([]*StoredRevision{revision(1, "overwritten")})
	if _, err = write(tamper); err == nil {
		t.Fatal("revision overwrite accepted")
	}
	invalid := head(3, 1, 3)
	bad := revision(3, "bad checksum")
	bad.SetContentSha256("incorrect")
	invalid.SetRevisions([]*StoredRevision{bad})
	if _, err = write(invalid); err == nil {
		t.Fatal("invalid revision accepted")
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM namespace_skill_revisions`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history count=%d err=%v", count, err)
	}
	if err = db.QueryRow(`SELECT latest_revision,published_revision,etag FROM namespace_skills`).Scan(&latest, &published, &etag); err != nil || latest != 2 || published != 1 || etag != 3 {
		t.Fatalf("failed write changed head: latest=%d published=%d etag=%d err=%v", latest, published, etag, err)
	}
	var original string
	if err = db.QueryRow(`SELECT bundle_json FROM namespace_skill_revisions WHERE revision=1`).Scan(&original); err != nil || original != `{"SKILL.md":"original"}` {
		t.Fatalf("original revision changed: %s err=%v", original, err)
	}
}
