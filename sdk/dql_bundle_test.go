package sdk

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"reflect"
	"strings"
	"testing"
)

func TestDQLArchiveFormatsPreserveAllDependencies(t *testing.T) {
	files := map[string]string{"a.dql": "SELECT 1", "b.dql": "SELECT 2", "sql/dependency.dql": "SELECT 3", "sql/query.sql": "SELECT 4", "assets/data.bin": "\x00\xff"}
	for _, format := range []string{"zip", "tar", "tar.gz"} {
		t.Run(format, func(t *testing.T) {
			var buffer bytes.Buffer
			if format == "zip" {
				writer := zip.NewWriter(&buffer)
				for name, content := range files {
					file, err := writer.Create(name)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = file.Write([]byte(content)); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				var writer *tar.Writer
				var compressed *gzip.Writer
				if format == "tar.gz" {
					compressed = gzip.NewWriter(&buffer)
					writer = tar.NewWriter(compressed)
				} else {
					writer = tar.NewWriter(&buffer)
				}
				for name, content := range files {
					if err := writer.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0600}); err != nil {
						t.Fatal(err)
					}
					if _, err := writer.Write([]byte(content)); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				if compressed != nil {
					if err := compressed.Close(); err != nil {
						t.Fatal(err)
					}
				}
			}
			bundle, err := ReadDQLArchive(&buffer, format)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bundle.Entries, []string{"a.dql", "b.dql"}) {
				t.Fatalf("entries=%v", bundle.Entries)
			}
			for name, content := range files {
				if string(bundle.Files[name]) != content {
					t.Fatalf("dependency %s changed", name)
				}
			}
		})
	}
}

func TestDQLArchiveRejectsUnsafeAndAmbiguousPaths(t *testing.T) {
	for _, names := range [][]string{{"../main.dql"}, {"/main.dql"}, {"C:/main.dql"}, {"dir\\main.dql"}, {"main.dql", "main.dql"}, {"main.dql", "a", "a/b.sql"}, {"nested/main.dql"}} {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		for _, name := range names {
			file, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			file.Write([]byte("SELECT 1"))
		}
		writer.Close()
		if _, err := ReadDQLArchive(&buffer, "zip"); err == nil {
			t.Errorf("accepted %v", names)
		}
	}
}

func TestDQLArchiveRejectsTARLinks(t *testing.T) {
	for _, kind := range []byte{tar.TypeLink, tar.TypeSymlink} {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		writer.WriteHeader(&tar.Header{Name: "main.dql", Linkname: "outside", Typeflag: kind})
		writer.Close()
		if _, err := ReadDQLArchive(&buffer, "tar"); err == nil {
			t.Fatalf("accepted link type %v", kind)
		}
	}
}

func TestReadDQLPreservesSourceAndRejectsInvalidText(t *testing.T) {
	source := "  #package('example/reader')\nSELECT 1\n"
	bundle, err := ReadDQL(strings.NewReader(source))
	if err != nil || string(bundle.Files["main.dql"]) != source {
		t.Fatalf("source changed: %v", err)
	}
	for _, source := range []string{"", " \n", "\xff", strings.Repeat("x", MaxDQLArchiveBytes+1)} {
		if _, err := ReadDQL(strings.NewReader(source)); err == nil {
			t.Fatal("accepted invalid source")
		}
	}
}
