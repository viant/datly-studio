package datatest

import (
	"reflect"
	"testing"

	authreader "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/report_publication_events/listoptions"
	"github.com/viant/datly-studio/studio/reports/catalogpredicate"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/x"
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
		catalogpredicate.ReportCatalogRead{},
		listoptions.Options{},
	}
	for _, value := range types {
		if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(value))); err != nil {
			t.Fatalf("register Studio authorization type %T: %v", value, err)
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
