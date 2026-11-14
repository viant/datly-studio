package sqltransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/viant/datly-studio/internal/componentremoval/removaltest"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/httptransport"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
)

func TestDeletionComponentTransport(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	f.Seed(ctx, "sdk-delete", 2)
	// Deliberately grant the outer transport authority: the native operation must
	// still enforce exact ownership for editors, publishers and administrators.
	transport := &sqltransport.Transport{DB: f.DB, ComponentInvoker: f.Server, Authorizer: sqltransport.AuthorizerFunc(func(context.Context, sqltransport.AuthorizationRequest) error { return nil })}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := f.Context(ctx, "alice")
	assertCode := func(t testing.TB, err error, want sdk.ErrorCode) {
		t.Helper()
		var actual *sdk.Error
		if !errors.As(err, &actual) || actual.Code != want || actual.Message == "" {
			t.Fatalf("error=%T %v, want SDK %s with safe message", err, err, want)
		}
	}
	t.Run("native conflict mapping", func(t *testing.T) {
		_, err := client.Components().Delete(owner, "sdk-delete", sdk.DeleteComponentInput{ETag: 2})
		assertCode(t, err, sdk.ErrorConflict)
		_, err = client.Versions().Delete(owner, "sdk-delete", 2, sdk.DeleteVersionInput{ExpectedSourceRevision: 6})
		assertCode(t, err, sdk.ErrorConflict)
	})
	for _, subject := range []string{"editor", "publisher", "admin"} {
		t.Run(subject+" cannot delete", func(t *testing.T) {
			f.T = t
			f.Hydrate(ctx, datatest.Table{Name: "component_acl", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": "sdk-delete", "subject_type": "user", "subject_id": subject, "can_view": true, "can_edit": subject == "editor", "can_publish": subject != "editor"}}})
			delegated := f.Context(ctx, subject, subject)
			_, err := client.Components().Delete(delegated, "sdk-delete", sdk.DeleteComponentInput{ETag: 1})
			assertCode(t, err, sdk.ErrorForbidden)
			_, err = client.Versions().Delete(delegated, "sdk-delete", 2, sdk.DeleteVersionInput{ExpectedSourceRevision: 7})
			assertCode(t, err, sdk.ErrorForbidden)
			f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='sdk-delete'", 2)
		})
	}
	t.Run("generic gateway preserves native error status", func(t *testing.T) {
		for _, tc := range []struct {
			subject, operation string
			body               any
			want               int
		}{
			{"alice", "components.delete", removaltest.Component("sdk-delete", 2), 409},
			{"alice", "versions.delete", removaltest.Version("sdk-delete", 2, 6), 409},
			{"admin", "components.delete", removaltest.Component("sdk-delete", 1), 403},
			{"admin", "versions.delete", removaltest.Version("sdk-delete", 2, 7), 403},
		} {
			authenticated := f.Context(ctx, tc.subject, tc.subject)
			gateway := httptransport.Gateway{Transport: transport, Config: httptransport.Config{Mode: httptransport.Authenticated, RequireNamespace: true, Authenticator: httptransport.AuthenticatorFunc(func(context.Context, *http.Request) (context.Context, error) { return authenticated, nil })}}
			raw, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/"+tc.operation, bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Studio-Namespace", f.Namespace)
			response := httptest.NewRecorder()
			gateway.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("%s as %s: %d want %d: %s", tc.operation, tc.subject, response.Code, tc.want, response.Body.String())
			}
			var wire sdk.Error
			if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire.Message == "" {
				t.Fatalf("safe SDK error missing: %s err=%v", response.Body.String(), err)
			}
		}
	})
	t.Run("SDK owner removes exact version then component", func(t *testing.T) {
		f.T = t
		result, err := client.Versions().Delete(owner, "sdk-delete", 2, sdk.DeleteVersionInput{ExpectedSourceRevision: 7})
		if err != nil || result == nil || result.ComponentID != "sdk-delete" || result.VersionNo != 2 || result.DeletedVersions != 1 {
			t.Fatalf("SDK version receipt=%+v err=%v", result, err)
		}
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='sdk-delete' AND last_version_no=2 AND current_draft_version IS NULL AND etag=2", 1)
		result, err = client.Components().Delete(owner, "sdk-delete", sdk.DeleteComponentInput{ETag: 2})
		if err != nil || result == nil || result.ComponentID != "sdk-delete" || result.VersionNo != 0 || result.DeletedVersions != 1 {
			t.Fatalf("SDK component receipt=%+v err=%v", result, err)
		}
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='sdk-delete' AND owner_id='alice'", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='sdk-delete'", 0)
	})
}
