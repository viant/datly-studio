package test_sql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
	"google.golang.org/api/googleapi"
)

type terminalJobDetail struct{ message, reason, location string }

func (e terminalJobDetail) Error() string {
	return "unsafe wrapper DSN=user:WRAPPER_PASSWORD@tcp(host)/db"
}
func (e terminalJobDetail) BigQueryReason() string   { return e.reason }
func (e terminalJobDetail) BigQueryMessage() string  { return e.message }
func (e terminalJobDetail) BigQueryLocation() string { return e.location }

func TestSQLExecutionReturnsUsefulNativeProviderDetails(t *testing.T) {
	for _, tc := range []struct {
		name                                             string
		cause                                            error
		provider, code, reason, location, state, message string
	}{
		{"BigQuery job", terminalJobDetail{"Access Denied: Table project.dataset.records: permission bigquery.tables.getData denied at [3:7]", "accessDenied", "query"}, "BigQuery", "accessDenied", "accessDenied", "query", "", "bigquery.tables.getData denied at [3:7]"},
		{"Google API", &googleapi.Error{Code: 400, Message: "Unrecognized name: advertser_id at [2:8]", Errors: []googleapi.ErrorItem{{Reason: "invalidQuery", Message: "Unrecognized name: advertser_id at [2:8]"}}, Body: `{"error":{"errors":[{"message":"Check column advertiser_id","reason":"invalidQuery","location":"q"}],"private":"BODY_SECRET"}}`}, "Google API", "400", "invalidQuery", "q", "", "Unrecognized name: advertser_id at [2:8]"},
		{"MySQL", &mysql.MySQLError{Number: 1146, SQLState: [5]byte{'4', '2', 'S', '0', '2'}, Message: "Table 'adlogs.missing_table' doesn't exist"}, "MySQL", "1146", "", "", "42S02", "Table 'adlogs.missing_table' doesn't exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := executionError(context.Background(), fmt.Errorf("connector wrapper SECRET: %w", &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "private DSN WRAPPER_PASSWORD", Cause: tc.cause}), time.Minute)
			var wire *xresponse.Error
			if !errors.As(err, &wire) {
				t.Fatal(err)
			}
			data, _ := json.Marshal(wire.Payload)
			var payload struct {
				Message       string        `json:"message"`
				ProviderError ProviderError `json:"providerError"`
			}
			if e := json.Unmarshal(data, &payload); e != nil {
				t.Fatal(e)
			}
			d := payload.ProviderError
			if d.Provider != tc.provider || d.Code != tc.code || d.Reason != tc.reason || d.Location != tc.location || d.SQLState != tc.state || !strings.Contains(d.Message, tc.message) || !strings.Contains(payload.Message, tc.message) {
				t.Fatalf("details missing: %s", data)
			}
			if tc.name == "Google API" && !strings.Contains(d.Message, "Check column advertiser_id") {
				t.Fatalf("additional native detail lost: %s", data)
			}
			for _, secret := range []string{"WRAPPER_PASSWORD", "BODY_SECRET", "wrapper SECRET"} {
				if strings.Contains(string(data), secret) || strings.Contains(err.Error(), secret) {
					t.Fatalf("nonprovider secret leaked: %s", data)
				}
			}
		})
	}
}

func TestNativeProviderCredentialRedactionAndBounds(t *testing.T) {
	native := `Syntax error at [5:2]. password="PASSWORD_CANARY" client_secret: 'CLIENT_CANARY' token=TOKEN_CANARY Authorization: Bearer BEARER_CANARY
mysql://user:URL_PASSWORD@host/db?%74oken=QUERY_TOKEN user:MYSQL_PASSWORD@tcp(host:3306)/db
{"private_key":"PRIVATE_CANARY"} -----BEGIN PRIVATE KEY-----KEY_CANARY-----END PRIVATE KEY-----
Use project.dataset.records and column advertiser_id.`
	d := providerExecutionError(terminalJobDetail{native, "invalidQuery", "password=LOCATION_CANARY"})
	raw, _ := json.Marshal(d)
	for _, secret := range []string{"PASSWORD_CANARY", "CLIENT_CANARY", "TOKEN_CANARY", "BEARER_CANARY", "URL_PASSWORD", "QUERY_TOKEN", "MYSQL_PASSWORD", "PRIVATE_CANARY", "KEY_CANARY", "LOCATION_CANARY"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("credential leaked: %s", raw)
		}
	}
	if !strings.Contains(d.Message, "project.dataset.records") || !strings.Contains(d.Message, "[5:2]") {
		t.Fatalf("useful provider text lost: %s", raw)
	}
	huge := providerExecutionError(&mysql.MySQLError{Number: 1064, Message: strings.Repeat("界", 30000)})
	if !huge.Truncated || len(huge.Message) > maxProviderMessage+len("\n[provider message truncated]") || !utf8.ValidString(huge.Message) {
		t.Fatalf("message not safely bounded: bytes=%d truncated=%v", len(huge.Message), huge.Truncated)
	}
}

func TestNonProviderErrorsKeepSafeFallback(t *testing.T) {
	for _, cause := range []error{errors.New("dial tcp user:NETWORK_PASSWORD@host failed"), errors.New("bigquery accessDenied password=SECRET"), errors.New("mysql error 1064 PASSWORD")} {
		err := executionError(context.Background(), cause, time.Minute)
		var wire *xresponse.Error
		if !errors.As(err, &wire) {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(wire.Payload)
		if strings.Contains(string(raw), "PASSWORD") || strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "providerError") {
			t.Fatalf("untyped error leaked: %s", raw)
		}
	}
}
