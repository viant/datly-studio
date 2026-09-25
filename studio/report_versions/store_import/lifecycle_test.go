package store_import

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func importedVersion(source string, files ...*ImportedResourceFile) *ImportedVersion {
	authored, generated := source, source
	return &ImportedVersion{ReportId: "r1", VersionNo: 3, State: "draft", AuthoringMode: "dql", AuthoredDql: &authored, GeneratedDql: &generated,
		ComponentSpecJson: json.RawMessage(`{}`), SpecFormatVersion: "studio.v1", SpecHash: "hash", TypeManifestJson: json.RawMessage(`{}`),
		CompileStatus: "pending", DatlyVersion: "v1", CompilerVersion: "studio.v1", SourceRevision: 1, CreatedBy: "owner",
		CreatedAt: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), File: files, Has: &ImportedVersionHas{}}
}

func linkedFile(path string, content []byte) *ImportedResourceFile {
	return &ImportedResourceFile{ReportId: "r1", VersionNo: 3, Namespace: "owner.imports.abcd", ResourcePath: path, Content: content, Has: &ImportedResourceFileHas{}}
}

func TestImportRulesDeriveFileIdentityAndDefaults(t *testing.T) {
	hooks := new(ImportRules)
	state := xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]{}
	source := "SELECT 1"
	version := &ImportedVersion{ReportId: "r1", VersionNo: 1, AuthoredDql: &source, SpecFormatVersion: "studio.v1", SpecHash: "h", DatlyVersion: "v1",
		CompilerVersion: "studio.v1", CreatedBy: "owner", CreatedAt: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		File: []*ImportedResourceFile{linkedFile("main.dql", []byte(source)), linkedFile("assets/a.bin", []byte{0xff, 0x00})}}
	version.File[0].VersionNo, version.File[1].VersionNo = 1, 1
	if err := hooks.Init(context.Background(), version, state); err != nil {
		t.Fatal(err)
	}
	if version.State != "draft" || version.AuthoringMode != "dql" || version.CompileStatus != "pending" || version.SourceRevision != 1 ||
		version.GeneratedDql == nil || *version.GeneratedDql != source || string(version.ComponentSpecJson) != "{}" || string(version.TypeManifestJson) != "{}" {
		t.Fatalf("defaults=%+v", version)
	}
	if !version.Has.State || !version.Has.GeneratedDql || !version.Has.ComponentSpecJson {
		t.Fatalf("defaults must be marked supplied: %+v", version.Has)
	}
	text, binary := version.File[0], version.File[1]
	if text.ResourceId != resourceIdentity("main.dql") || text.ContentSha256 != contentDigest([]byte(source)) || text.ContentSize != 8 || text.IsBinary || !text.CreatedAt.Equal(version.CreatedAt) {
		t.Fatalf("text file=%+v", text)
	}
	if !binary.IsBinary || binary.ContentSize != 2 || !binary.Has.IsBinary || !binary.Has.ContentSize || !binary.Has.ResourceId || !binary.Has.ContentSha256 {
		t.Fatalf("binary file=%+v has=%+v", binary, binary.Has)
	}
	if err := hooks.Validate(context.Background(), version, state); err != nil {
		t.Fatal(err)
	}
}

func TestImportRulesValidateDeniesEveryContractViolation(t *testing.T) {
	hooks := new(ImportRules)
	ctx, state := context.Background(), xhandler.LifecycleContext[ImportedVersion, xhandler.NoParent, Output]{}
	valid := func() *ImportedVersion {
		version := importedVersion("SELECT 1", linkedFile("main.dql", []byte("SELECT 1")), linkedFile("sql/q.sql", []byte("SELECT 2")))
		if err := hooks.Init(ctx, version, state); err != nil {
			t.Fatal(err)
		}
		return version
	}
	if err := hooks.Validate(ctx, valid(), state); err != nil {
		t.Fatalf("valid import denied: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(v *ImportedVersion)
		want string
	}{
		{"missing report", func(v *ImportedVersion) { v.ReportId = " " }, "report id is required"},
		{"unallocated version", func(v *ImportedVersion) { v.VersionNo = 0 }, "must be allocated"},
		{"not draft", func(v *ImportedVersion) { v.State = "validated" }, "must be drafts"},
		{"not dql", func(v *ImportedVersion) { v.AuthoringMode = "sql" }, "dql authoring mode"},
		{"compiled", func(v *ImportedVersion) { v.CompileStatus = "valid" }, "pending compile status"},
		{"revision", func(v *ImportedVersion) { v.SourceRevision = 2 }, "source revision 1"},
		{"hash", func(v *ImportedVersion) { v.SpecHash = "" }, "spec_hash is required"},
		{"created by", func(v *ImportedVersion) { v.CreatedBy = "" }, "created_by is required"},
		{"created at", func(v *ImportedVersion) { v.CreatedAt = time.Time{} }, "created_at is required"},
		{"blank source", func(v *ImportedVersion) { blank := " "; v.AuthoredDql = &blank }, "requires DQL source"},
		{"generated drift", func(v *ImportedVersion) { other := "SELECT 5"; v.GeneratedDql = &other }, "must equal the authored"},
		{"spec not object", func(v *ImportedVersion) { v.ComponentSpecJson = json.RawMessage(`[]`) }, "JSON objects"},
		{"no files", func(v *ImportedVersion) { v.File = nil }, "at least one resource file"},
		{"unsafe path", func(v *ImportedVersion) { v.File[1].ResourcePath = "../q.sql" }, "unsafe resource path"},
		{"absolute path", func(v *ImportedVersion) { v.File[1].ResourcePath = "/etc/passwd" }, "unsafe resource path"},
		{"windows path", func(v *ImportedVersion) { v.File[1].ResourcePath = `sql\q.sql` }, "unsafe resource path"},
		{"duplicate path", func(v *ImportedVersion) { v.File[1].ResourcePath = "main.dql" }, "duplicate resource path"},
		{"foreign file", func(v *ImportedVersion) { v.File[1].ReportId = "other" }, "must belong to the imported version"},
		{"namespace", func(v *ImportedVersion) { v.File[1].Namespace = "" }, "namespace is required"},
		{"identity", func(v *ImportedVersion) { v.File[1].ResourceId = "0" }, "SHA-256 of its path"},
		{"digest", func(v *ImportedVersion) { v.File[1].ContentSha256 = "0" }, "content digest"},
		{"size", func(v *ImportedVersion) { v.File[1].ContentSize = 99 }, "content size"},
		{"binary marker", func(v *ImportedVersion) { v.File[1].IsBinary = true }, "binary marker"},
		{"file created at", func(v *ImportedVersion) { v.File[1].CreatedAt = time.Time{} }, "created_at is required"},
		{"source not retained", func(v *ImportedVersion) {
			for _, file := range v.File {
				file.Content = []byte("SELECT 9")
				file.ContentSha256 = contentDigest(file.Content)
			}
		}, "retained as a resource file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			version := valid()
			test.edit(version)
			err := hooks.Validate(ctx, version, state)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}
