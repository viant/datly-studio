package providererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/api/googleapi"
)

type catalogJobError struct{ reason string }

func (e catalogJobError) Error() string           { return "wrapper password=WRAPPER_CANARY" }
func (e catalogJobError) BigQueryReason() string  { return e.reason }
func (e catalogJobError) BigQueryMessage() string { return "SELECT 'SQL_CANARY'" }
func (e catalogJobError) BigQueryLocation() string {
	return "LOCATION_CANARY"
}

func TestAppendCatalogStatus(t *testing.T) {
	const fallback = "connector catalog is unavailable"
	for _, test := range []struct {
		cause error
		want  string
	}{
		{&googleapi.Error{Code: 403, Message: "Bearer TOKEN_CANARY", Errors: []googleapi.ErrorItem{{Reason: "accessDenied"}}}, fallback + "\nGoogle API error 403 (accessDenied)"},
		{&googleapi.Error{Code: 404, Message: "https://storage.example/object?X-Goog-Signature=SIGNATURE_CANARY", Errors: []googleapi.ErrorItem{{Reason: "notFound"}}}, fallback + "\nGoogle API error 404 (notFound)"},
		{&mysql.MySQLError{Number: 1045, Message: "Access denied for user 'USER_CANARY'@'HOST_CANARY' (using password: YES). reader:DSN_CANARY@/db"}, fallback + "\nMySQL error 1045"},
		{catalogJobError{reason: "notFound"}, fallback + "\nBigQuery error (notFound)"},
		{catalogJobError{reason: "UNRECOGNIZED_REASON_CANARY"}, fallback + "\nBigQuery error"},
		{errors.New("unsupported native capability: password=DRIVER_CANARY"), fallback},
	} {
		wrapped := fmt.Errorf("private connection password=WRAPPER_CANARY: %w", test.cause)
		if got := AppendCatalogStatus(fallback, wrapped); got != test.want {
			t.Errorf("message = %q, want %q", got, test.want)
		}
	}
}
