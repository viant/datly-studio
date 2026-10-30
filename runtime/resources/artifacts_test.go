package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/viant/forge/backend/artifactcohort"
)

func TestNativeSkillArtifactsRetainExactVersionAndSupportingFiles(t *testing.T) {
	f := newSnapshotFixture(t)
	for _, v := range []int{1, 2} {
		f.version("component", v)
		text := "First version"
		if v == 2 {
			text = "Second version"
		}
		f.file("component", v, "entry", "docs", "guide/SKILL.md", "---\nname: guide\ndescription: Example guide\n---\n"+text)
		f.file("component", v, "notes", "docs", "guide/notes.md", text)
		f.file("component", v, "binary", "docs", "guide/sample.bin", string([]byte{0, 255, 1}))
		f.folder("component", v, "folder", "docs", "guide", "skill://native/component/guide/", 0)
		f.skill("component", v, "guide", "folder", ".", 0)
	}
	loaded, err := Load(context.Background(), f.db, []Version{{ReportID: "component", VersionNo: 1}})
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE component_resource_files SET content='changed after snapshot' WHERE report_id='component' AND version_no=1 AND resource_id='notes'`)
	artifacts, err := loaded.SkillArtifacts(context.Background(), "example", []Version{{ReportID: "component", VersionNo: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts=%d", len(artifacts))
	}
	got := artifacts[0]
	if got.Source != (Version{ReportID: "component", VersionNo: 1}) {
		t.Fatal("source provenance lost")
	}
	if got.Artifact.Root.URI != "skill://example/component/guide" || got.Artifact.Root.Version != 1 {
		t.Fatalf("root=%+v", got.Artifact.Root)
	}
	if artifactcohort.ValidateArtifact(got.Artifact) != nil {
		t.Fatal("invalid artifact")
	}
	var value struct {
		Document []byte            `json:"document"`
		Files    map[string][]byte `json:"files"`
	}
	if err = json.Unmarshal(got.Artifact.Payload, &value); err != nil {
		t.Fatal(err)
	}
	if string(value.Files["notes.md"]) != "First version" {
		t.Fatalf("wrong supporting bytes: %q", value.Files["notes.md"])
	}
	if string(value.Files["sample.bin"]) != string([]byte{0, 255, 1}) {
		t.Fatal("binary support file changed")
	}
	if _, err = loaded.SkillArtifacts(context.Background(), "example", []Version{{ReportID: "component", VersionNo: 2}}); err == nil {
		t.Fatal("accepted a version outside the captured snapshot")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = loaded.SkillArtifacts(canceled, "example", []Version{{ReportID: "component", VersionNo: 1}}); err == nil {
		t.Fatal("ignored cancellation")
	}
	if len(value.Document) == 0 {
		t.Fatal("entrypoint missing")
	}
}
