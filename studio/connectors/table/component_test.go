package table

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk/connectivity"
	xresponse "github.com/viant/xdatly/response"
	"google.golang.org/api/googleapi"
)

func TestDiscoveryErrorKeepsPrimaryClassification(t *testing.T) {
	permission := &googleapi.Error{Code: 403, Message: "CANARY_ACCESS", Errors: []googleapi.ErrorItem{{Reason: "accessDenied"}}}
	syntax := &googleapi.Error{Code: 400, Message: "CANARY_SQL", Errors: []googleapi.ErrorItem{{Reason: "invalidQuery"}}}
	unavailable := &mysql.MySQLError{Number: 2003, Message: "CANARY_CONNECTION"}
	for _, tc := range []struct {
		primary, probe error
		status         int
		reason         string
	}{
		{permission, syntax, 403, "accessDenied"},
		{permission, unavailable, 403, "accessDenied"},
		{syntax, permission, 400, "invalidQuery"},
		{permission, context.Canceled, 499, "canceled"},
		{syntax, context.DeadlineExceeded, 504, "deadline"},
	} {
		native := &connectivity.ColumnDiscoveryError{Primary: tc.primary, Probe: tc.probe, Phase: "zero-row probe"}
		err := discoveryError(native)
		if !errors.Is(err, tc.primary) || !errors.Is(err, tc.probe) || xresponse.ErrorStatusCode(err, 0) != tc.status {
			t.Fatalf("original failure changed: %v", err)
		}
		var original *googleapi.Error
		if !errors.As(err, &original) || original != tc.primary {
			t.Fatalf("original typed cause lost: %v", err)
		}
		body, _ := xresponse.ErrorBody(err)
		raw, _ := json.Marshal(body)
		if !strings.Contains(string(raw), tc.reason) || strings.Contains(string(raw), "CANARY") {
			t.Fatalf("unsafe or missing primary classification: %s", raw)
		}
	}
}
