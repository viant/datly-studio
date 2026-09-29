package readerinspection

import (
	"github.com/viant/datly/spec"
	"testing"
)

func TestHydrateColumnsPreservesAuthoredGraph(t *testing.T) {
	source := &spec.ViewSource{SQL: "SELECT parent.* FROM (${embed:sql/parent.sql}) parent"}
	child := &spec.View{Name: "child", Namespace: "lookup", Source: &spec.ViewSource{SQL: "SELECT ID,NAME FROM fixture"}}
	link := &spec.Relation{View: child}
	authored := &spec.View{Name: "parent", Source: source, Relations: []*spec.Relation{link}}
	resolved := &spec.View{Name: "runtime-parent", Columns: []*spec.Column{{Name: "publisher_id"}}, Relations: []*spec.Relation{{View: &spec.View{Name: "runtime-child", Namespace: "lookup", Columns: []*spec.Column{{Name: "NAME"}}}}}}
	HydrateColumns(authored, resolved)
	if authored.Source != source || authored.Relations[0] != link || authored.Source.SQL != "SELECT parent.* FROM (${embed:sql/parent.sql}) parent" {
		t.Fatal("authored source or relation identity replaced")
	}
	if len(authored.Columns) != 1 || authored.Columns[0].Name != "publisher_id" || len(child.Columns) != 1 || child.Columns[0].Name != "NAME" {
		t.Fatal("resolved columns missing from graph")
	}
}
