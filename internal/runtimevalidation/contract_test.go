package runtimevalidation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/readerinspection"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "modernc.org/sqlite"
)

func TestContractNativeOperationsHaveSafeRepairDiagnostics(t *testing.T) {
	secret := "https://user:password@source.invalid/table?token=TOKEN SELECT private_value FROM private_record"
	for _, tc := range []struct{ operation, code, stage, field string }{
		{"materialize input marker: ", "runtime_input_marker", "contract.input", "authoredDql"},
		{"materialize input contract: ", "runtime_input_type", "contract.input", "authoredDql"},
		{"materialize output contract: ", "runtime_output_type", "contract.output", "authoredDql"},
		{"evaluate schema-discovery source: ", "runtime_discovery_template", "contract.discovery", "authoredDql"},
		{"re-evaluate named identity source: ", "runtime_discovery_template", "contract.discovery", "authoredDql"},
		{"bind discovery SQL: ", "runtime_discovery_bind", "contract.discovery", "authoredDql"},
		{"build schema-discovery query: ", "runtime_discovery_query_shape", "contract.discovery", "authoredDql"},
		{"SQLX discovery failed for ", "runtime_discovery_query", "contract.discovery", "authoredDql"},
		{"resolve SQL dialect: ", "runtime_discovery_dialect", "contract.discovery", "defaultConnectorName"},
	} {
		t.Run(tc.code+tc.operation, func(t *testing.T) {
			cause := errors.New(tc.operation + secret)
			err := Wrap("contract", fmt.Errorf("private wrapper: %w", cause))
			d := Diagnostics(err)[0]
			if d.Code != tc.code || d.Stage != tc.stage || d.Field != tc.field || d.Hint == "" || d.Message == "" || !errors.Is(err, cause) {
				t.Fatalf("diagnostic=%+v", d)
			}
			for _, allowed := range []bool{true, false} {
				projected := readerinspection.RedactDiagnostics(Diagnostics(err), allowed)
				wire, _ := json.Marshal(projected)
				for _, token := range []string{"user", "password", "source.invalid", "TOKEN", "private_", "SELECT", "https://"} {
					if strings.Contains(string(wire), token) {
						t.Fatalf("cause leaked: %s", wire)
					}
				}
				if !allowed && (projected[0].Message != "" || projected[0].Hint != "") {
					t.Fatal("delegated caller received diagnostic detail")
				}
			}
		})
	}
}

func TestContractClassificationDoesNotChangeOtherBoundaries(t *testing.T) {
	cause := errors.New("bind discovery SQL: private parameter")
	for _, stage := range []string{"connector", "definition", "artifact", "reader", "resources"} {
		if got := Diagnostics(Wrap(stage, cause))[0]; got.Code != "runtime_"+stage {
			t.Fatalf("cross-stage classification: %+v", got)
		}
	}
	for _, message := range []string{"unknown private failure", "prefix bind discovery SQL: private parameter", "SQLX discovery failure", "materialize output contract"} {
		if got := Diagnostics(Wrap("contract", errors.New(message)))[0]; got.Code != "runtime_contract" {
			t.Fatalf("unrecognized cause classified: %+v", got)
		}
	}
	for _, tc := range []struct {
		cause error
		code  string
	}{{context.Canceled, "validation_canceled"}, {context.DeadlineExceeded, "validation_deadline"}} {
		if got := Diagnostics(Wrap("contract", fmt.Errorf("bind discovery SQL: %w", tc.cause)))[0]; got.Code != tc.code {
			t.Fatalf("cancellation lost: %+v", got)
		}
	}
	if got := Diagnostics(&Error{Stage: "contract", Code: "runtime_package_authority", Cause: cause})[0]; got.Code != "runtime_package_authority" {
		t.Fatalf("authority lost: %+v", got)
	}
}

// Exercise an actual native discovery failure, rather than defining the
// acceptance oracle solely with hand-authored error strings.
func TestContractNativeDiscoveryFailure(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = transcribe.NewCompiler().RuntimeContractsInModule(ctx, "example.com/app", &transcribe.Source{
		Scope: "example.com/app/records", Name: "records", Connector: "source",
		ColumnRefiner: column.New(column.Connections{"source": db}),
		Text: `#package('example.com/app/records')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT records.*, type(records, 'Row') FROM (SELECT ID FROM UNAVAILABLE_RECORDS) records`,
	})
	if err == nil {
		t.Fatal("missing source unexpectedly compiled")
	}
	d := Diagnostics(Wrap("contract", err))[0]
	if d.Code != "runtime_discovery_query" || d.Stage != "contract.discovery" {
		t.Fatalf("native failure projected as %+v (type %T)", d, err)
	}
	wire, _ := json.Marshal(d)
	if strings.Contains(string(wire), "UNAVAILABLE_RECORDS") || strings.Contains(string(wire), "SELECT") {
		t.Fatalf("source leaked: %s", wire)
	}
}
