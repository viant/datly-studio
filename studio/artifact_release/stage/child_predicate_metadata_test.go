package stage_test

import (
	complete "github.com/viant/datly-studio/studio/artifact_release/complete"
	fail "github.com/viant/datly-studio/studio/artifact_release/fail"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/runtime/handler/writer"
	"reflect"
	"testing"
)

func TestGeneratedReleaseChildPredicateSurvivesNativeCompilation(t *testing.T) {
	for _, holder := range []any{complete.CompleteComponent{}, fail.FailComponent{}} {
		typ := reflect.TypeOf(holder)
		t.Run(typ.Name(), func(t *testing.T) {
			reflected, err := bootstrap.ReflectPackages([]string{typ.PkgPath()})
			if err != nil {
				t.Fatal(err)
			}
			if len(reflected.Components) != 1 {
				t.Fatalf("components=%d", len(reflected.Components))
			}
			source := reflected.Components[0]
			component, err := source.Resolve(source.LinkedInputType, source.LinkedOutputType)
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := writer.Compile(component, source.LinkedInputType, source.LinkedOutputType, "patch")
			if err != nil {
				t.Fatal(err)
			}
			if metadata.Root.MutationPredicateGroup == nil || *metadata.Root.MutationPredicateGroup != 7 {
				t.Fatal("root predicate group7 lost")
			}
			if len(metadata.Root.Relations) != 1 {
				t.Fatalf("relations=%d", len(metadata.Root.Relations))
			}
			operation := metadata.Root.Relations[0].Child
			if operation.MutationPredicateGroup == nil {
				t.Fatal("generated operation mutationPredicate=8 tag lost in native child record")
			}
			if *operation.MutationPredicateGroup != 8 {
				t.Fatalf("child group=%d", *operation.MutationPredicateGroup)
			}
		})
	}
}
