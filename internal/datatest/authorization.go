package datatest

import (
	"reflect"
	"testing"

	"github.com/viant/datly-studio/sdk"
	authreader "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/authorization"
	_ "github.com/viant/datly-studio/studio/connectors/accesspredicate"
	_ "github.com/viant/datly-studio/studio/namespaces/accesspredicate"
	"github.com/viant/datly-studio/studio/report_publication_events/listoptions"
	_ "github.com/viant/datly-studio/studio/report_publications/otheractivepredicate"
	versionlistoptions "github.com/viant/datly-studio/studio/report_versions/listoptions"
	_ "github.com/viant/datly-studio/studio/report_versions/mcpnamepredicate"
	"github.com/viant/datly-studio/studio/reports/catalogpredicate"
	_ "github.com/viant/datly-studio/studio/reports/globalpredicate"
	_ "github.com/viant/datly-studio/studio/runtime_generations/catalogpredicate"
	_ "github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/x"
	xpredicate "github.com/viant/xdatly/predicate"
	"github.com/viant/xunsafe"
)

// StudioAuthorizationTypes mirrors the linked type authority supplied by
// datly.yaml and internal/dependencylink for focused DQL compiler tests.
func StudioAuthorizationTypes(t testing.TB) *typecatalog.Catalog {
	t.Helper()
	catalog := typecatalog.NewCatalog()
	types := []any{
		jwt.Claims{},
		authreader.AuthContext{}, authreader.Output{},
		authorization.ConnectorRead{}, authorization.ConnectorEdit{}, authorization.NamespaceRead{},
		authorization.ReportRead{}, authorization.ReportEdit{}, authorization.ReportPublish{},
		authorization.ReportVersionRead{}, authorization.ReportVersionMetadataRead{}, authorization.ReportVersionEdit{}, authorization.ReportViewRead{},
		authorization.ReportParameterRead{}, authorization.ReportParameterEdit{},
		authorization.ReportCubeRead{}, authorization.ReportCubeEdit{},
		authorization.ReportMCPRead{}, authorization.ReportMCPEdit{},
		authorization.ReportResourceFileRead{}, authorization.ReportResourceFileEdit{},
		authorization.ReportResourceFolderRead{}, authorization.ReportResourceFolderEdit{},
		authorization.ReportSkillRead{}, authorization.ReportSkillEdit{},
		authorization.PublicationRead{}, authorization.PublicationEdit{}, authorization.PublicationEventRead{},
		authorization.ACLRead{}, authorization.ACLEdit{},
		authorization.RuntimeRead{}, authorization.RuntimeEdit{},
		authorization.WarmupRead{}, authorization.SessionRead{}, authorization.SessionRevoke{},
		catalogpredicate.ReportCatalogRead{},
		listoptions.Options{},
		versionlistoptions.Options{},
		sdk.WarmupRun{}, sdk.WarmupTarget{}, sdk.Diagnostic{},
	}
	for _, value := range types {
		if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(value))); err != nil {
			t.Fatalf("register Studio authorization type %T: %v", value, err)
		}
	}
	// Server-only predicates are selected by their linked package identity,
	// never by a second per-type test registry. Transcription must see the same
	// compiled authority that the executable will use.
	for _, packagePath := range []string{
		"github.com/viant/datly-studio/studio/connectors/accesspredicate",
		"github.com/viant/datly-studio/studio/namespaces/accesspredicate",
		"github.com/viant/datly-studio/studio/report_publications/otheractivepredicate",
		"github.com/viant/datly-studio/studio/report_versions/mcpnamepredicate",
		"github.com/viant/datly-studio/studio/reports/globalpredicate",
		"github.com/viant/datly-studio/studio/runtime_generations/catalogpredicate",
		"github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate",
	} {
		linked := 0
		for _, typ := range xunsafe.PackageTypes(packagePath) {
			for typ != nil && typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			if typ == nil || typ.PkgPath() != packagePath || !reflect.PointerTo(typ).Implements(reflect.TypeFor[xpredicate.Handler]()) {
				continue
			}
			if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(typ)); err != nil {
				t.Fatalf("register linked predicate %s.%s: %v", packagePath, typ.Name(), err)
			}
			linked++
		}
		if linked == 0 {
			t.Fatalf("predicate package %s has no compiled handler type; run datly link sync and rebuild", packagePath)
		}
	}
	if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(jwt.Claims{}), x.WithPkgPath("jwt"), x.WithName("Claims"))); err != nil {
		t.Fatalf("register JWT import alias: %v", err)
	}
	if err := catalog.Register(typecatalog.TypeOriginPackage, &x.Type{Type: reflect.TypeOf(jwt.Claims{}), Name: "Claims"}); err != nil {
		t.Fatalf("register JWT import lookup name: %v", err)
	}
	if err := catalog.Register(typecatalog.TypeOriginPackage, &x.Type{Type: reflect.TypeOf((*jwt.Claims)(nil)), Name: "*jwt.Claims"}); err != nil {
		t.Fatalf("register JWT pointer expression: %v", err)
	}
	if resolved, ok, err := catalog.Resolve(typecatalog.PackageAuthority, "Claims"); err != nil || !ok || resolved == nil || resolved.Type != reflect.TypeOf(jwt.Claims{}) {
		t.Fatalf("resolve JWT import lookup name: type=%+v found=%v err=%v", resolved, ok, err)
	}
	return catalog
}
