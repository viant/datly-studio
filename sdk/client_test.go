package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSDKUsesCanonicalJSONFieldNames(t *testing.T) {
	payload, err := json.Marshal(struct {
		Report  Report
		Version ReportVersion
	}{
		Report:  Report{DefaultConnectorName: "main", ComponentScope: "reports"},
		Version: ReportVersion{AuthoringMode: "dql", AuthoredDQL: "SELECT 1", SourceRevision: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{"DefaultConnectorName", "ComponentScope", "AuthoringMode", "AuthoredDQL", "SourceRevision"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("SDK JSON contains Go field name %q: %s", forbidden, text)
		}
	}
	for _, required := range []string{"defaultConnectorName", "componentScope", "authoringMode", "authoredDql", "sourceRevision"} {
		if !strings.Contains(text, required) {
			t.Fatalf("SDK JSON misses canonical field name %q: %s", required, text)
		}
	}
}

type transportCall struct {
	operation string
	input     any
}

type recordingTransport struct{ calls []transportCall }

func (t *recordingTransport) Invoke(_ context.Context, operation string, input, output any) error {
	t.calls = append(t.calls, transportCall{operation: operation, input: input})
	switch actual := output.(type) {
	case *Connector:
		actual.Name, actual.Status = "main", "active"
	case *ReportVersion:
		actual.ReportID, actual.VersionNo = "r-1", 2
	case *RuntimeStatus:
		actual.ActiveGeneration, actual.Status = 7, "active"
	}
	return nil
}

func TestClientRoutesTypedOperationsThroughTransport(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Fatal("nil transport accepted")
	}
	transport := &recordingTransport{}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Connectors().Create(context.Background(), CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil || created.Name != "main" || created.Status != "active" {
		t.Fatalf("create connector = %+v, %v", created, err)
	}
	version, err := client.Versions().Get(context.Background(), "r-1", 2)
	if err != nil || version.ReportID != "r-1" || version.VersionNo != 2 {
		t.Fatalf("get version = %+v, %v", version, err)
	}
	status, err := client.Runtime().Status(context.Background())
	if err != nil || status.ActiveGeneration != 7 {
		t.Fatalf("runtime status = %+v, %v", status, err)
	}
	want := []string{OperationConnectorCreate, OperationVersionGet, OperationRuntimeStatus}
	if len(transport.calls) != len(want) {
		t.Fatalf("calls = %+v", transport.calls)
	}
	for i, operation := range want {
		if transport.calls[i].operation != operation {
			t.Fatalf("call %d operation = %q, want %q", i, transport.calls[i].operation, operation)
		}
	}
}

func TestSDKConnectorNeverExposesResolvedDSN(t *testing.T) {
	typeOf := reflect.TypeFor[Connector]()
	if _, ok := typeOf.FieldByName("DSN"); ok {
		t.Fatal("SDK connector exposes resolved DSN")
	}
	if field, ok := typeOf.FieldByName("DSNTemplate"); !ok || field.Tag.Get("json") != "-" {
		t.Fatalf("DSN template field = %+v", field)
	}
	if field, ok := typeOf.FieldByName("DSNConfigured"); !ok || field.Tag.Get("json") != "dsnConfigured" {
		t.Fatalf("DSN configured field = %+v", field)
	}
	if field, ok := typeOf.FieldByName("SecretRef"); !ok || field.Tag.Get("json") != "-" {
		t.Fatalf("secret reference field = %+v", field)
	}
	if field, ok := typeOf.FieldByName("SecretConfigured"); !ok || field.Tag.Get("json") != "secretConfigured" {
		t.Fatalf("secret configured field = %+v", field)
	}
}

func TestSDKHasNoDatlyControlOrStoreImports(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve SDK path")
	}
	directory := filepath.Dir(filename)
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		for _, imported := range file.Imports {
			path, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil {
				t.Fatal(unquoteErr)
			}
			for _, forbidden := range []string{"github.com/viant/datly", "github.com/viant/datly-studio/control", "github.com/viant/datly-studio/store", "database/sql"} {
				if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
					t.Fatal(fmt.Sprintf("SDK file %s imports forbidden implementation package %s", entry.Name(), path))
				}
			}
		}
	}
}
