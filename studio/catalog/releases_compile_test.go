package catalog

import (
	"reflect"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	rhandler "github.com/viant/datly/runtime/handler"
	dtag "github.com/viant/datly/tag"
)

func TestNativeReleaseComponentsCompileWithVerifiedBodyDTO(t *testing.T) {
	jwt := datatest.NewJWTFixture(t)
	for _, item := range []struct {
		holder, input, output reflect.Type
		handler               func() (rhandler.TypedHandler, error)
	}{
		{reflect.TypeFor[ReleaseListComponent](), reflect.TypeFor[ReleaseListInput](), reflect.TypeFor[ReleaseListOutput](), (ReleaseListComponent{}).DatlyHandler("NewReleaseList")},
		{reflect.TypeFor[ReleaseCandidatesComponent](), reflect.TypeFor[ReleaseCandidatesInput](), reflect.TypeFor[ReleaseCandidatesOutput](), (ReleaseCandidatesComponent{}).DatlyHandler("NewReleaseCandidates")},
		{reflect.TypeFor[ReleasePublishComponent](), reflect.TypeFor[ReleasePublishInput](), reflect.TypeFor[ReleaseResultOutput](), (ReleasePublishComponent{}).DatlyHandler("NewReleasePublish")},
		{reflect.TypeFor[ReleasePromoteComponent](), reflect.TypeFor[ReleasePromoteInput](), reflect.TypeFor[ReleaseResultOutput](), (ReleasePromoteComponent{}).DatlyHandler("NewReleasePromote")},
		{reflect.TypeFor[ReleaseHistoryComponent](), reflect.TypeFor[ReleaseHistoryInput](), reflect.TypeFor[ReleaseHistoryOutput](), (ReleaseHistoryComponent{}).DatlyHandler("NewReleaseHistory")},
	} {
		field, ok := item.holder.FieldByName("Contract")
		if !ok || item.handler == nil {
			t.Fatal("release component handler missing")
		}
		metadata, present, err := dtag.ParseComponent(field.Tag)
		if err != nil || !present {
			t.Fatalf("component metadata: %v", err)
		}
		component, err := (&bootstrap.RouteSource{HolderType: item.holder.Name(), FieldName: field.Name, PackageName: item.holder.Name(), PackagePath: item.holder.PkgPath(), Tag: metadata}).Resolve(item.input, item.output)
		if err != nil {
			t.Fatal(err)
		}
		handler, err := item.handler()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: item.input, OutputType: item.output, Resources: resource.New(), Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: handler, HandlerOwnedOutput: true}); err != nil {
			t.Fatalf("%s release DTO compile: %v", item.holder.Name(), err)
		}
	}
}
